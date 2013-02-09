package main

import (
	"archive/zip"
	"code.google.com/p/go.crypto/bcrypt"
	"encoding/gob"
	"encoding/json"
	"fmt"
	. "github.com/colemickens/xbtunnel/common"
	"labix.org/v2/mgo/bson"
	"log"
	"net/http"
	"sync"
)

func listRooms(w http.ResponseWriter) {
	enc := json.NewEncoder(w)
	res := []Room{}
	db.C("rooms").Find(nil).All(&res)
	enc.Encode(res)
}

func register(nickname, password string) (*User, error) {
	// TODO: enforce uniqueness

	var hashed_pw []byte
	var err error

	if hashed_pw, err = bcrypt.GenerateFromPassword([]byte(password), 10); err != nil {
		return nil, err
	}
	user := &User{
		Nickname: nickname,
		Password: string(hashed_pw),
		Token:    "token",
	}
	if err = db.C("Users").Insert(&user); err != nil {
		return nil, err
	}
	return user, nil
}

func login(w http.ResponseWriter, nickname, password string) (*Session, error) {
	log.Println("login():", nickname, password)

	var user User
	err := db.C("xbtusers").
		Find(bson.M{"nickname": nickname}).
		One(&user)

	if err != nil {
		log.Println(err)
		return nil, fmt.Errorf("err: " + err.Error())
	}

	if user.Id == "" {
		return nil, fmt.Errorf("Unknown nickname")
	}

	if err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return nil, fmt.Errorf("Wrong password")
	}

	return _setSessionCookie(w, user.Id), nil
}

func _setSessionCookie(w http.ResponseWriter, id PeerId) *Session {
	s := ss.getSessionByPeerId(id)
	if s == nil {
		s = ss.createSession(id)
	}
	sid_str := bson.ObjectId(s.Id).Hex()
	c := http.Cookie{
		Name:  "sessionid",
		Value: sid_str,
	}
	log.Println("set cookie", c.String())
	http.SetCookie(w, &c)

	return s
}

func download(w http.ResponseWriter, os, arch string, u *User) {
	binaryName := "xbtunnel"
	if os == "windows" {
		binaryName += ".exe"
	}
	configName := "config.json"

	binary := []byte{0x00, 0x01, 0x02}
	config := []byte("{'token':'" + u.Token + "'")

	w.Header().Set("Content-Disposition", "attachment")

	zw := zip.NewWriter(w)
	zw.Create(binaryName)
	w.Write(binary)
	zw.Create(configName)
	w.Write(config)

	err := zw.Close()
	if err != nil {
		log.Println(err)
	}
}

// /////////////////////////////
// Middle-Helpers for Session
// /////////////////////////////

///////////////////////////////////////////////////
///////////////////////////////////////////////////
///////////////////////////////////////////////////

type Session struct {
	sync.RWMutex
	Id     SessionId `bson:"_id,omitempty" json:"id"`
	UserId PeerId
	//LocalState LocalState `bson:"omit" json"omit"`
	Xboxes []Xbox `bson:"xboxes" json:"xboxes"`

	encoder *gob.Encoder
}

type SessionId bson.ObjectId

func (sid SessionId) String() string { return bson.ObjectId(sid).Hex() }

func (s *Session) makeAndChangeRoom(roomName, roomDesc string) error {
	r := rms.newRoom(roomName, roomDesc)

	s.LocalState.CurRoom = r

	rms.notifyRoom(r.Id)

	return nil
}

func xbld(s_one []Xbox, s_two []Xbox) bool {
	if len(s_one) != len(s_two) {
		return true
	}
	for _, v1 := range s_one {
		found := false
		for _, v2 := range s_two {
			if v2 == v1 {
				found = true
				break
			}
		}
		if found != true {
			return true
		}
	}
	return false
}
