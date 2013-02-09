package main

import (
	"code.google.com/p/nat"
	"fmt"
	. "github.com/colemickens/xbtunnel/common"
	"log"
	"net"
	"time"
)

type ShimConn struct {
	to       PeerId
	readChan chan []byte
	kill     chan int
	dead     chan int
}

func newShimConn(to PeerId) *ShimConn {
	return &ShimConn{to, make(chan []byte), make(chan int, 1), make(chan int, 1)}
}

func (sc *ShimConn) Write(bytes []byte) (int, error) {
	signal := new(PcSignal)
	signal.To = sc.to
	signal.Payload = make([]byte, len(bytes))
	copy(signal.Payload, bytes)

	serverChan <- ServerReq{
		PcSignal: signal,
	}

	return len(bytes), nil
}

func (sc *ShimConn) Read(bytes []byte) (int, error) {
	select {
	case tmp := <-sc.readChan:
		n := copy(bytes, tmp)
		return n, nil
	case <-sc.kill:
		defer func() { sc.dead <- 1 }()
		return -1, fmt.Errorf("got killed")
	}
	return -1, nil
}

func (sc *ShimConn) LocalAddr() net.Addr                { return nil }
func (sc *ShimConn) RemoteAddr() net.Addr               { return nil }
func (sc *ShimConn) SetDeadline(t time.Time) error      { return nil }
func (sc *ShimConn) SetReadDeadline(t time.Time) error  { return nil }
func (sc *ShimConn) SetWriteDeadline(t time.Time) error { return nil }
func (sc *ShimConn) Close() error                       { close(sc.readChan); return nil }

type PeerConn struct {
	peerId   PeerId
	sideband *ShimConn
	udpConn  net.Conn
	out      chan UdpPacket
	kill     chan int
	dead     chan int
}

func dialPeerConnection(peerId PeerId) {
	_dialPeerConnection(peerId, true)
}

func receivePeerConnection(peerId PeerId) {
	_dialPeerConnection(peerId, false)
}

func _dialPeerConnection(peerId PeerId, initiator bool) {
	pc := &PeerConn{
		peerId:   peerId,
		sideband: newShimConn(peerId),
		udpConn:  nil,
		out:      make(chan UdpPacket, 10), // how to choose buffer size?
		kill:     make(chan int, 1),
		dead:     make(chan int, 1),
	}

	newPeerConnChan <- pc

	go func() {
		var err error
		pc.udpConn, err = nat.Connect(pc.sideband, initiator)
		// TODO : do i close shimconn?
		if err == nil {
			// it's okay, leave the connection in there
			// though it may be null because 
			// it's not connected yet
			// and we're eagerly sending packets
		} else {
			// we still need to do this, don't we?
			// or don't we need to output the peerc
			// first so that it gets there
			// so that it can be found?

			removePeerChan <- pc.peerId
			log.Println("failed to add new pc", pc.peerId)
		}
	}()
}

func (pc *PeerConn) run() { // pointers for func params?
	log.Println("run() peer conn for id:", pc.peerId)

	defer func() {
		// see this pattern in daemon.go
		close(pc.kill)
		close(pc.out)

		pc.sideband.kill <- 1
		<-pc.sideband.dead

		pc.dead <- 1
		// TODO:ASKML: do I need to close dead
	}()

	go func() {
		data := make([]byte, 65535)
		for {
			if n, err := pc.udpConn.Read(data); err == nil {
				packet := ParseRemoteUdpPacket(data[:n])
				localChan <- packet
			} else {
				pc.kill <- 1
				return
			}
		}
	}()

	for {
		select {
		case pkt := <-pc.out:
			if _, err := pc.udpConn.Write(pkt.AsUdpData()); err != nil {
				return
			}
		case <-pc.kill:
			return
		}
	}
}

func resolvePeerConns(myUserId PeerId, peerConns PeerConnMap, peerList []Peer) {
	// TODO: on shutdown, resolve with nil to kill conns
	if peerList == nil {
		for _, pc := range peerConns {
			<-pc.kill
			// TODO: should we wait to return?
			// that way we exit 100% gracefully
		}
	}

	// Dial to new peers or receive from new peers
	for _, peer := range peerList {
		if _, ok := peerConns[peer.Id]; ok {
			if peer.Id < myUserId {
				log.Println("dial to:", peer.Id)
				dialPeerConnection(peer.Id)
			} else {
				log.Println("receive from:", peer.Id)
				receivePeerConnection(peer.Id)
			}
		}
	}

	// Remove existing pcs that are no longer in the room
	for testPeerId, testPeerConn := range peerConns {
		testPeerIsGood := false
		for _, goodPeer := range peerList {
			if testPeerId == goodPeer.Id {
				testPeerIsGood = true
			}
		}
		if !testPeerIsGood {
			log.Println("peer id no longer valid, killing:", testPeerId)
			testPeerConn.kill <- 1
		}
	}
	log.Println("leaving resolvePeerConns")
}

/*
// We no longer do this because we now can receive() and it will be there
// ready to receive this payload when it comes in
// because we do an id comparison for top/bottom, I mean, sender/receiver.
func distributePcSignal(signal PcSignal) {
	peerConnLock.RLock()
	pc, exist := peerConnMap[signal.From]
	peerConnLock.RUnlock()

	if !exist {
		log.Println("remote:rpc:make:pc:", signal.From, false)
		pc = pcMake(signal.From, false)
	}

	pc.sideband.readChan <- signal.Payload
}
*/
