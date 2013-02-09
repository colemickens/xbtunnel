package main

import (
	pcap "github.com/colemickens/gopcap"
	. "github.com/colemickens/xbtunnel/common"
	"log"
)

type Handle struct {
	name  string
	iface *pcap.Interface
	pcap  *pcap.Pcap
}

type XboxHandle struct {
	x Xbox
	h *Handle
}

type LocalManager struct {
	handles      []*Handle
	out          chan UdpPacket
	xboxes       chan XboxHandle
	kill         chan int
	killChildren chan int
	dead         chan int
}

func newLocalManager() (*LocalManager, error) {
	lm := &LocalManager{
		getHandles(),
		make(chan UdpPacket, 5),
		make(chan XboxHandle, 5),
		make(chan int, 1),
		make(chan int, 10),
		make(chan int, 1),
	}

	for _, handle := range lm.handles {
		var err error
		handle.pcap, err = pcap.Openlive(handle.iface.Name, 65536, true, 1000)
		if err != nil {
			log.Fatalln(err)
		}
		handle.pcap.Setfilter("ip && udp && host 0.0.0.1")
	}

	return lm, nil
}

func (lm *LocalManager) run() {
	children := 0
	xboxHandleMap := make(map[Xbox]*Handle)

	for _, handle := range lm.handles {
		children++
		go lm.runpcap(handle) // TODO: clean  this up?
	}

	defer func() {
		for children > 0 {
			lm.killChildren <- 1
			children--
		}
		lm.dead <- 1
	}()

	for {
		select {
		case pkt := <-lm.out:
			pcapPacket, _ := pkt.AsTransmittablePcap([]byte{0x00, 0x00})

			if pkt.Dst.IsBroadcast() {
				for _, h := range xboxHandleMap {
					h.pcap.Inject(pcapPacket)
				}
			} else if h, ok := xboxHandleMap[pkt.Dst]; ok {
				h.pcap.Inject(pcapPacket)
			}

		case xh := <-lm.xboxes:
			_, ok := xboxHandleMap[xh.x]

			if !ok {
				xboxHandleMap[xh.x] = xh.h
			}
			newXboxChan <- xh.x

		case <-lm.kill:
			return
		}
	}
}

type xboxHandle struct {
	x Xbox
	h *Handle
}

func (lm *LocalManager) runpcap(handle *Handle) {
	for {
		select {
		case <-lm.killChildren:
			return
		default:
			pkt := handle.pcap.Next()
			if pkt != nil {
				packet := ParsePcapPacket(pkt)
				if packet.Dst.IsBroadcast() {
					// only check if not known?
					// won't this read their's?

					lm.xboxes <- XboxHandle{packet.Dst, handle}
				}
				remoteChan <- packet
			}
		}
	}
}

func getHandles() (handles []*Handle) {
	ifs, err := pcap.Findalldevs()
	if err != nil {
		panic(err)
	}
	for _, iface := range ifs {
		if len(iface.Addresses) > 0 && iface.Name[0:2] != "lo" {
			handles = append(handles, &Handle{
				iface: &iface,
			})
		}
	}
	return
}
