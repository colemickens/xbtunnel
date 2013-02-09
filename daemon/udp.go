package main

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	pcap "github.com/colemickens/gopcap"
	"github.com/colemickens/xbtunnel/common"
	"log"
)

type LocalPacket struct {
	Handle *Handle
	Packet UdpPacket
}

type UdpPacket struct {
	Payload []byte
	Src     common.Xbox
	Dst     common.Xbox
}

func (r *UdpPacket) String() string {
	res := "\nsrc: " + r.Src.String()
	res += "\ndst: " + r.Dst.String()
	res += "\n" + printBytes(r.Payload)
	return res
}

func ParseRemoteUdpPacket(data []byte) UdpPacket {
	l := len(data)
	if l < 12 {
		log.Println("got malnurished packet :[")
		return UdpPacket{}
	}
	return UdpPacket{
		data[:l-12],
		common.NewXbox(data[l-12 : l-6]),
		common.NewXbox(data[l-6:]),
	}
}

func ParsePcapPacket(pkt *pcap.Packet) UdpPacket {
	pkt.Decode()
	return UdpPacket{
		pkt.Payload,
		common.NewXbox(pkt.Data[6:12]),
		common.NewXbox(pkt.Data[0:6]),
	}
}

func (r *UdpPacket) AsUdpData() []byte {
	return append(r.Payload,
		append(r.Src[:], r.Dst[:]...)...)
}

func (r *UdpPacket) AsTransmittablePcap(id []byte) ([]byte, error) {
	if len(id) != 2 {
		return nil, errors.New("ID is too long")
	}

	ether_buffer := new(bytes.Buffer)
	ip_buffer := new(bytes.Buffer)
	udp_buffer := new(bytes.Buffer)
	udp_psuedo_buffer := new(bytes.Buffer)

	// ETHERNET
	ether_buffer.Write(r.Dst.AsSlice())    // Destination
	ether_buffer.Write(r.Src.AsSlice())    // Source
	ether_buffer.Write([]byte{0x08, 0x00}) // Type: IP (0x0800)

	ether_bytes := ether_buffer.Bytes()

	// IPV4
	src_ip := []byte{0x00, 0x00, 0x00, 0x01}
	dst_ip := []byte{0x00, 0x00, 0x00, 0x01}
	if r.Dst.IsBroadcast() {
		dst_ip = []byte{0xFF, 0xFF, 0xFF, 0xFF}
	}

	ip_buffer.WriteByte(0x45)                                              // IPv4 + Header_Length (always 20)
	ip_buffer.WriteByte(0x00)                                              // Differentiated Service Field (always zero)
	binary.Write(ip_buffer, binary.BigEndian, uint16(len(r.Payload)+8+20)) // Total Length
	ip_buffer.Write(id)                                                    // Identifcation (always zero, zero)
	ip_buffer.Write([]byte{0x00})                                          // Flags
	ip_buffer.Write([]byte{0x00})                                          // Fragment Offset
	ip_buffer.WriteByte(0x40)                                              // TTL
	ip_buffer.WriteByte(0x11)                                              // Protocol: UDP (17)
	ip_buffer.Write([]byte{0x00, 0x00})                                    // Header checksum (gets replaced later)
	ip_buffer.Write(src_ip)                                                // Source
	ip_buffer.Write(dst_ip)                                                // Destination

	ip_bytes := ip_buffer.Bytes()

	// UDP

	udp_buffer.Write([]byte{0x0c, 0x02})                                 // Source port
	udp_buffer.Write([]byte{0x0c, 0x02})                                 // Destination port
	binary.Write(udp_buffer, binary.BigEndian, uint16(len(r.Payload)+8)) // Length
	udp_buffer.Write([]byte{0x00, 0x00})                                 // Checksum
	udp_buffer.Write(r.Payload)                                          // Data

	udp_bytes := udp_buffer.Bytes()

	udp_psuedo_buffer.Write(src_ip)
	udp_psuedo_buffer.Write(dst_ip)
	udp_psuedo_buffer.WriteByte(0x00)
	udp_psuedo_buffer.WriteByte(0x11)
	binary.Write(udp_psuedo_buffer, binary.BigEndian, uint16(len(udp_buffer.Bytes())))
	udp_psuedo_buffer.Write(udp_buffer.Bytes())

	// Checksums
	ip_checksum := oneAdd(ip_buffer.Bytes())
	udp_checksum := oneAdd(udp_psuedo_buffer.Bytes())

	binary.BigEndian.PutUint16(ip_bytes[10:12], ip_checksum)
	binary.BigEndian.PutUint16(udp_bytes[6:8], udp_checksum)

	_ = udp_checksum

	ret := append(ether_bytes, ip_bytes...)
	return append(ret, udp_bytes...), nil
}

func oneAdd(bs ...[]byte) uint16 {
	sum := uint64(0)

	for _, b := range bs {
		for i := 0; i < (len(b) - 1); i += 2 {
			sum += uint64(b[i])<<8 + uint64(b[i+1])
		}
		if len(b)%2 == 1 {
			sum += uint64(b[len(b)-1]) << 8
		}
	}

	for sum>>16 != 0 {
		sum = (sum & 0xffff) + (sum >> 16)
	}

	return uint16(^sum)
}

func printBytes(bytes []byte) string {
	return fmt.Sprintf("%#x", bytes)
}
