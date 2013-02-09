package common

import (
	"fmt"
)

type Xbox [6]byte

func NewXbox(mac []byte) Xbox {
	typed_mac := [6]byte{}
	copy(typed_mac[0:], mac[0:])
	x := Xbox(typed_mac)
	return x
}

func (x *Xbox) String() string {
	// important to deference
	return fmt.Sprintf("%0x:%0x:%0x:%0x:%0x:%0x", x[0], x[1], x[2], x[3], x[4], x[5])
}

func (x Xbox) AsSlice() []byte {
	b := make([]byte, 6)
	copy(b[0:], x[0:])
	return b
}

func (x *Xbox) IsBroadcast() bool {
	if x[0] == 0xFF &&
		x[1] == 0xFF &&
		x[2] == 0xFF &&
		x[3] == 0xFF &&
		x[4] == 0xFF &&
		x[5] == 0xFF {
		return true
	}
	return false
}
