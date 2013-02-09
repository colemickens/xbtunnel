package main

import (
	"errors"
	"flag"
	. "github.com/colemickens/xbtunnel/common"
	"log"
	"os"
	"os/signal"
	"time"
)

// features, webrtc video/audio chat
// made as a plugin from camego source

var (
	main_server string
	token       string

	stateManager *StateManager
	localManager *LocalManager
	client       *Client

	serverChan      chan ServerReq
	localChan       chan UdpPacket
	remoteChan      chan UdpPacket
	localStateChan  chan *LocalState
	pcSignalChan    chan PcSignal
	newPeerConnChan chan *PeerConn
	removePeerChan  chan PeerId
	newXboxChan     chan Xbox
)

func init() {
	flag.Parse()

	serverChan = make(chan ServerReq, 10)       // TODO: size?
	localChan = make(chan UdpPacket, 10)        // TODO: size?
	remoteChan = make(chan UdpPacket, 10)       // TODO: size?
	localStateChan = make(chan *LocalState, 10) // TODO: size?
	pcSignalChan = make(chan PcSignal, 10)      // TODO: size?
	newPeerConnChan = make(chan *PeerConn, 10)  // TODO: size?
	removePeerChan = make(chan PeerId, 10)
	newXboxChan = make(chan Xbox, 10)
}

func main() {
	server := *flag.String("s", "xbtunnel.com:9000", "server to connect to")
	token := *flag.String("t", "test_token", "token to auth with")

	var err error
	stateManager = newStateManager()
	localManager, err = newLocalManager()
	if err != nil {
		panic(err)
	}

	go stateManager.run()
	go localManager.run()

	kill, dead := make(chan int, 1), make(chan int, 1)

	go func() {
		defer func() {
			// reroute serverChan to nil?
			// do we still router out to serverChan when we're disco'd anyway? I'd imagine so.

		}()
		for {
			select {
			case <-kill:
				log.Println("not trying to reconnect, breaking")
				dead <- 1
				return

			default:
				log.Println("creating client to ", server)
				client, err = newClient(server, token)
				if err != nil {
					log.Printf("daemon can't lookup server (%s).\n", server)
				}

				err := client.run()
				if err != nil {
					log.Println("daemon was disconnected or we killed it.")
				}
			}
			time.Sleep(5 * time.Second)
		}
	}()
	defer func() {}()

	log.Println("wait for interrupt:")

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	<-c

	deadcount := 0

	kill <- 1
	deadcount++
	log.Println("killed")

	stateManager.kill <- 1
	deadcount++
	log.Println("stateManager killed")

	localManager.kill <- 1
	deadcount++
	log.Println("localManager killed")

	// TODO: TIE THIS TO OUR CHANNEL COPY
	// THIS IS NOT VALID AFTER CLIENT DISCOS

	// (how did this ever work?, we must have a default empty client)
	client.kill <- errors.New("interrupt kill client")
	// client doesn't use dead, it returns an err above

	for {
		select {
		case <-stateManager.dead:
			log.Println("stateManager died")
			deadcount--
		case <-localManager.dead:
			log.Println("localManager died")
			deadcount--
		case <-dead:
			log.Println("died")
			deadcount--
		default:
			if deadcount <= 0 {
				return
			}
			time.Sleep(500 * time.Millisecond)
		}
	}

	// TODO: copy from remoteChan to client.out as well
	// copy from localChan to lm.out ? (why not just do it directly)
}

type StateManager struct {
	kill chan int
	dead chan int
}

func newStateManager() *StateManager {
	return &StateManager{
		kill: make(chan int, 1),
		dead: make(chan int, 1),
	}
}

type PeerConnMap map[PeerId]*PeerConn
type XboxPeerMap map[Xbox]PeerId

func (sm *StateManager) run() {
	peerConnMap := make(PeerConnMap)
	xboxPeerMap := make(XboxPeerMap)
	localState := &LocalState{
	/*
		type LocalState struct {
			UserId    UserID
			CurRoomId RoomId
			Peers     []struct {
				Id        PeerId
				Xboxes    []Xbox
				Connected bool
			}
			Xboxes    []Xbox
		}
	*/
	}

	defer func() {
		// by doing clean up here
		// we prevent leaking peerConnMap higher
		// which would allow other functions to access it
		// and violate mutual exclusion
		for _, pc := range peerConnMap {
			pc.kill <- 1
		}
		for _, pc := range peerConnMap {
			<-pc.dead
		}
		sm.dead <- 1
	}()

	for {
		sendUpdate := true

		select {
		case ls := <-localStateChan:
			localState.UserId = ls.UserId
			localState.CurRoomId = ls.CurRoomId
			// what do we do with the xbox list
			// - it could include new ones
			// (we don't store that state)
			// - it could also wipeout new ones?
			// -- sure, easy to imagine a race condition

			resolvePeerConns(localState.UserId, peerConnMap, localState.Peers)
			updateXboxPeerMap(&xboxPeerMap, localState.Peers)

		case pcSignal := <-pcSignalChan:
			if peer, ok := peerConnMap[pcSignal.To]; ok {
				peer.sideband.readChan <- pcSignal.Payload
			}
			sendUpdate = false

		case packet := <-remoteChan:
			if localState == nil {
				continue
			}

			if packet.Dst.IsBroadcast() {
				for _, p := range peerConnMap {
					p.out <- packet
				}
			} else {
				if peerId, ok := xboxPeerMap[packet.Dst]; ok {
					if peer, ok2 := peerConnMap[peerId]; ok2 {
						peer.out <- packet
					}
				}
			}

		case newPc := <-newPeerConnChan:
			peerConnMap[newPc.peerId] = newPc
			setPeerConnected(localState.Peers, newPc.peerId, true)

		case pid := <-removePeerChan:
			delete(peerConnMap, pid)

		case newXbox := <-newXboxChan:
			localState.Xboxes = append(localState.Xboxes, newXbox)

		case <-sm.kill:
			return
		}

		if sendUpdate {
			serverChan <- ServerReq{LocalState: localState}
		}
	}
}

func setPeerConnected(peers []Peer, peerId PeerId, connected bool) {
	for i, p := range peers {
		if p.Id == peerId {
			peers[i].Connected = connected
			break
		}
	}
}

func updateXboxPeerMap(replaceMap *XboxPeerMap, peers []Peer) {
	replacementMap := make(map[Xbox]PeerId)

	for _, p := range peers {
		for _, x := range p.Xboxes {
			replacementMap[x] = p.Id
		}
	}

	*replaceMap = replacementMap
}
