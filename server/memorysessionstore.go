package main

import (
	. "github.com/colemickens/xbtunnel/common"
	"labix.org/v2/mgo/bson"
	"log"
	"net/http"
	"sync"
)

type SessionStore struct {
	sync.RWMutex
	s map[SessionId]*Session
}

func newSessionStore() *SessionStore {
	return &SessionStore{
		s: make(map[SessionId]*Session),
	}
}

func (ss *SessionStore) createSession(pid PeerId) *Session {
	s := &Session{
		Id: SessionId(bson.NewObjectId()),
		LocalState: LocalState{
			UserId: pid,
		},
	}
	ss.Lock()
	ss.s[s.Id] = s
	ss.Unlock()
	return s
}

func (ss *SessionStore) getSessionById(id SessionId) *Session {
	ss.RLock()
	defer ss.RUnlock()
	return ss.s[id]
}

func (ss *SessionStore) getSessionByToken(token Token) *Session {
	var user *User
	db.C("xbtusers").Find(bson.M{"token": token}).One(user)

	ss.RLock()
	defer ss.RUnlock()

	for _, s := range ss.s {
		s.RLock()
		defer s.RUnlock()
		if s.LocalState.UserId == user.Id {
			return s
		}
	}

	return nil
}

func (ss *SessionStore) getSessionByPeerId(pid PeerId) *Session {
	ss.RLock()
	defer ss.RUnlock()

	for _, s := range ss.s {
		s.RLock()
		defer s.RUnlock()
		if s.LocalState.UserId == pid {
			return s
		}
	}

	return nil
}

func (ss *SessionStore) getSessionByRequest(r *http.Request) *Session {
	for _, c := range r.Cookies() {
		log.Println("c:", c)
	}
	sid, err := r.Cookie("sessionid")
	if err != nil {
		log.Println("err!", err)
		return nil
	}

	ss.RLock()
	defer ss.RUnlock()
	for _, s := range ss.s {
		s.RLock()
		defer s.RUnlock()
		if s.Id.String() == sid.Value {
			return s
		}
	}

	return nil
}

func (ss *SessionStore) getSessionByUserId(id PeerId) *Session {
	ss.RLock()
	for _, s := range ss.s {
		s.RLock()
		if s.LocalState.UserId == id {
			return s
		}
		s.RUnlock()
	}
	ss.RUnlock()

	return nil
}
