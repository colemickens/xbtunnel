package main

import (
	. "github.com/colemickens/xbtunnel/common"
	"labix.org/v2/mgo/bson"
	"log"
	"sync"
)

type RoomStore struct {
	sync.RWMutex
	rooms       map[RoomId]*Room
	userRoomMap map[PeerId]*Room
	userXboxMap map[PeerId][]Xbox
}

var roomIdGen chan RoomId

func init() {
	roomIdGen = make(chan RoomId, 1)
	go func() {
		for i := 100; i < 10000; i++ {
			roomIdGen <- RoomId(bson.NewObjectId())
		}
	}()
}

func newRoomStore() *RoomStore {
	return &RoomStore{
		sync.RWMutex{},
		make(map[RoomId]*Room),
		make(map[PeerId]*Room),
	}
}

func (rs *RoomStore) newRoom(name, desc string) *Room {
	rs.Lock()
	defer rs.Unlock()

	r := &Room{
		Id:          <-roomIdGen,
		Name:        name,
		Description: desc,
	}
	rs.rooms[r.Id] = r
	return r
}

func (rs *RoomStore) getRoom(id RoomId) (r *Room) {
	rs.RLock()
	defer rs.RUnlock()
	r = rs.rooms[id]
	return
}

func (rs *RoomStore) addUserToRoom(pid PeerId, r *Room) {
	rs.userRoomMap[pid] = r
}

func (rs *RoomStore) removeUserFromRoom(pid PeerId) {
	remove(rs.userRoomMap, pid)
}

func (rs *RoomStore) changeUserRoom(pid PeerId, r *Room) {
		rs.Lock()

		if rid, ok := rs.userRoomMap[pid]; ok {
			// remove from existing room
			rs.removeUserFromRoom(rid)
			rs.notifyRoom(rid)
		}

		if r != nil {
			// add user to new room
			rs.addUserToRoom(pid, r)
			rs.notifyRoom(r.Id)
		}

		s.Unlock()
}

func (rs *RoomStore) notifyRoom(id RoomId) {
	ss.RLock()
	for _, os := range ss.s {
		if os.LocalState.CurRoom != nil && os.LocalState.CurRoom.Id == id {
			err := os.encoder.Encode(ServerResp{
				LocalState: makeLocalState(),
			})
			if err != nil {
				log.Println("notify, encode err:", err)
			}
		}
	}
	ss.RUnlock()
}

func ls2peer(ls LocalState) Peer {
	return Peer{
		Id:     ls.UserId,
		Xboxes: ls.Xboxes,
	}
}
