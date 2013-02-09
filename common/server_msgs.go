package common

import (
	"labix.org/v2/mgo/bson"
)

// CLIENT -> SERVER REQUESTS

type Token string

type ServerReq struct {
	// send periodically
	Token    *Token
	Xboxes   []Xbox
	PcSignal *PcSignal
}

// SERVER -> CLIENT REQUESTS

type ServerResp struct {
	LocalState *LocalState
	PcSignal   *PcSignal
}

type User struct {
	Id       PeerId `bson:"_id,omitempty" json:"id"`
	Nickname string `bson:"nickname" json:"nickname"`
	Password string `bson:"password" json:"-"`
	Token    string `bson:"token" json:"-"`
}

type Room struct {
	Id          RoomId `bson:"_id,omitempty" json:"id"`
	Name        string `bson:"name" json:"name"`
	Description string `bson:"description" json:"description"`
}

// BOTH

type LocalState struct {
	UserId      PeerId
	XboxPeerMap map[Xbox]PeerId
	PeerList    []PeerId
}

type PcSignal struct {
	From    PeerId
	To      PeerId
	Payload []byte
}

type Peer struct {
	Id     PeerId `bson:"_id,omitempty" json:"id"`
	Xboxes []Xbox `bson:"xboxes" json:"xboxes"`
}

type PeerId bson.ObjectId
type RoomId bson.ObjectId

func (pid PeerId) String() string { return bson.ObjectId(pid).Hex() }
func (rid RoomId) String() string { return bson.ObjectId(rid).Hex() }

const (
	EmptyPeerId PeerId = ""
)
