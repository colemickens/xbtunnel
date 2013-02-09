package main

import (
	"code.google.com/p/gorilla/mux"
	"encoding/gob"
	"encoding/json"
	. "github.com/colemickens/xbtunnel/common"
	"labix.org/v2/mgo"
	"log"
	"net"
	"net/http"
	"time"
)

var (
	tcpsock *net.TCPListener
	db      *mgo.Database
	ss      *SessionStore // this should be all sessions
	rms     *RoomStore
)

const (
	VERSION_NUMBER string = "1"
)

func init() {
	addr, err := net.ResolveTCPAddr("tcp", ":9000")
	if err != nil {
		panic(err)
	}

	tcpsock, err = net.ListenTCP("tcp", addr)
	if err != nil {
		panic(err)
	}

	log.Println("listening on", addr)

	ss = newSessionStore()
	rms = newRoomStore()
	init_mgo()
}

func init_mgo() {
	session, err := mgo.Dial("localhost")
	if err != nil {
		panic(err)
	}
	session.SetMode(mgo.Monotonic, true)
	db = session.DB("xbtunnel")

	db.C("xbtusers").DropCollection()
	db.C("xbtsessions").DropCollection()
	db.C("xbtrooms").DropCollection()
}

func main() {
	go handleTcpConns()

	// TODO: add signal processing to gracefully shutdown
	// TCP is easy, close socket
	// HTTP I'll have to pull out to be able ot Close() it

	r := mux.NewRouter()

	r.HandleFunc("/api/register", func(w http.ResponseWriter, r *http.Request) {
		var data map[string]interface{}
		dec := json.NewDecoder(r.Body)
		dec.Decode(&data)
		r.Body.Close()

		nickname := data["nickname"].(string)
		password := data["password"].(string)

		user, err := register(nickname, password)
		if user == nil || err != nil {
			log.Println(err)
			w.WriteHeader(403)
			return
			// TODO: handle this error
		}

		login(w, nickname, password) // set "id=_id" cookie?
		w.WriteHeader(200)
	})
	r.HandleFunc("/api/login", func(w http.ResponseWriter, r *http.Request) {
		var data map[string]interface{}
		dec := json.NewDecoder(r.Body)
		dec.Decode(&data)
		r.Body.Close()

		nickname := data["nickname"].(string)
		password := data["password"].(string)

		session, err := login(w, nickname, password)

		if session == nil {
			log.Println("session is nillll", err)
		}
		if err != nil {
			w.WriteHeader(403)
			w.Write([]byte(err.Error()))
			return
		}

		w.WriteHeader(200)
	})
	r.HandleFunc("/api/download", func(w http.ResponseWriter, r *http.Request) {
		s := ss.getSessionByRequest(r)
		if s == nil {
			w.WriteHeader(403) // access denied? TODO: check this code
			return
		}

		os := "windows"
		arch := "32"

		var u *User
		db.C("xbtusers").FindId(s.LocalState.UserId).One(u)

		download(w, os, arch, u)
	})
	r.HandleFunc("/api/rooms", func(w http.ResponseWriter, r *http.Request) {
		s := ss.getSessionByRequest(r)

		// TODO: session is still valid even if I removed the user

		if s == nil {
			log.Println("session was nil")
			w.WriteHeader(403) // access denied? TODO: check this code
			return
		}
		if r.Method == "GET" {
			w.Header().Set("Content-Type", "application/json")
			listRooms(w)
		} else if r.Method == "POST" {
			data := make(map[string]interface{})
			dec := json.NewDecoder(r.Body)
			dec.Decode(&data)
			r.Body.Close()

			roomName := data["name"].(string)
			roomDesc := data["description"].(string)

			log.Println("name:", roomName)
			log.Println("description:", roomDesc)

			s.makeAndChangeRoom(roomName, roomDesc)

			log.Println("session after making/changing rooms: ", s)
		}
	})
	r.HandleFunc("/api/status", func(w http.ResponseWriter, r *http.Request) {
		s := ss.getSessionByRequest(r)
		if s == nil {
			w.WriteHeader(403) // access denied? TODO: check this code
			return
		}
		w.Header().Set("Content-Type", "application/json")
		jsonEnc := json.NewEncoder(w)
		jsonEnc.Encode(s)
	})
	r.HandleFunc("/api/cmd/changeroom", func(w http.ResponseWriter, r *http.Request) {
		s := ss.getSessionByRequest(r)
		if s == nil {
			w.WriteHeader(403) // access denied? TODO: check this code
			return
		}
		roomId := r.URL.Query().Get("roomId")
		room := rms.getRoom(RoomId(roomId))
		if room == nil {
			w.WriteHeader(502)
		} else {
			rms.changeUserRoom(s.UserId, room)
			w.WriteHeader(200)
		}
	})
	r.HandleFunc("/api/version", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("{'version':" + VERSION_NUMBER + "}"))
	})

	r.Handle("/{id:.*}", http.FileServer(http.Dir("./_web/")))

	http.Handle("/", r)

	log.Fatal(http.ListenAndServe(":81", r))
}

func shutdown() {
	tcpsock.Close()
}

func handleTcpConns() {
	for {
		conn, err := tcpsock.AcceptTCP()
		if err != nil {
			log.Println(err)
			return
		}

		go handleUserConn(conn)
		log.Println("user:", conn.RemoteAddr().String())
	}
}

func handleUserConn(conn *net.TCPConn) {
	encoder := gob.NewEncoder(conn)
	decoder := gob.NewDecoder(conn)

	var s *Session

	defer func() {
		if s != nil {
			s.encoder = nil
		}
	}()

	deadc := make(chan int, 1)

	resetTimeout := func() {
		t := time.Now().Add(50 * time.Second)
		conn.SetReadDeadline(t)
	}

	go func() {
		var req ServerReq
		for {
			req.PcSignal = nil
			req.LocalState = nil

			resetTimeout()

			req := ServerReq{}
			err := decoder.Decode(&req)
			log.Println("*")
			if err != nil {
				log.Println("tcp_decode_err", err)
				deadc <- 1
				return
			}

			if req.Token != nil {
				log.Println("check token", req.Token)
				s = ss.getSessionByToken(*req.Token)
				if s == nil {
					log.Println("bullshit token", req.Token)
				}
				s.encoder = encoder
			}

			if s != nil && req.PcSignal != nil {
				pc := req.PcSignal
				pc.From = s.LocalState.UserId
				ss.RLock()
				to_s, ok := ss.s[SessionId(pc.To)]
				ss.RUnlock()
				if ok {
					to_s.RLock()
					if to_s.encoder != nil {
						to_s.encoder.Encode(ServerResp{
							PcSignal: pc,
						})
					}
					to_s.RUnlock()
				}
			}

			if s != nil && req.Xboxes != nil {
				// change user's xbox
				// change room's cached xboxPeerIdMap ?
				// do we cache that ^
				log.Println(req.LocalState) // this is informative
			}
		}
	}()

	ticker := time.NewTicker(2 * time.Second)
	for {
		select {
		case <-deadc:
			return
		case <-ticker.C:
			if encoder != nil { // this is racy
				err := encoder.Encode(ServerResp{})
				if err != nil {
					log.Println(err)
				}
			}
		}
	}
}
