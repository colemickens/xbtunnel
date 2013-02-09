package main

import (
	"testing"
)

func pkt_to_bytes(s string) ([]byte, error) {
	out := []byte{}
	_, err := fmt.Sscanf(s, "%x", &out)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func TestPcapMaker(t *testing.T) {
	id_pkt1 := []byte{0x14, 0xc2}
	data_pkt1_str := "68000000bf89470eeb95edc62fea068b9fc124805b08534d01010000007f392002dc7a32ae8b0711d741ce739e"
	data_pkt1, _ := pkt_to_bytes(data_pkt1_str)

	actual_pkt1 := UdpPacket{
		Payload: data_pkt1,
		Src:     NewXbox([]byte{0x00, 0x1d, 0xd8, 0xa9, 0x4e, 0x62}),
		Dst:     NewXbox([]byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff}),
	}

	actual_pcap_pkt1, _ := actual_pkt1.AsTransmittablePcap(id_pkt1)

	expected_complete_pkt1 := "ffffffffffff001dd8a94e6208004500004914c20000401165e200000001ffffffff0c020c020035457c68000000bf89470eeb95edc62fea068b9fc124805b08534d01010000007f392002dc7a32ae8b0711d741ce739e"
	actual_complete_pkt1 := fmt.Sprintf("%x", actual_pcap_pkt1)

	if actual_complete_pkt1 != expected_complete_pkt1 {
		t.Fatalf("Packet mismatch")
		t.Logf("actual", actual_complete_pkt1)
		t.Logf("expect", expected_complete_pkt1)
	}
}
