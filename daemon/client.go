package main

import (
	"encoding/gob"
	"fmt"
	. "github.com/colemickens/xbtunnel/common"
	"log"
	"net"
	"time"
)

type Client struct {
	addr *net.TCPAddr
	kill chan error
	out  chan interface{}
}

func newClient(server, token string) (*Client, error) {
	addr, err := net.ResolveTCPAddr("tcp", server)
	if err != nil {
		return nil, err
	}

	return &Client{
		addr,
		make(chan error, 1), // must be buffered
		make(chan interface{}, 10),
	}, nil
}

func (c *Client) run() error {
	_ = time.Now

	tcpconn, err := net.DialTCP("tcp", nil, c.addr)
	if err != nil {
		return err
	}

	defer func() {
		// again daemon.go for this pattern
		if tcpconn != nil {
			tcpconn.Close()
			close(c.out)
			close(c.kill)
		}
	}()

	pinger := time.NewTicker(2 * time.Second)

	encoder := gob.NewEncoder(tcpconn)
	decoder := gob.NewDecoder(tcpconn)

	go func() {
		t := Token(token)
		serverChan <- ServerReq{Token: &t}
	}()

	resetTimeout := func() { // TODO: factor this out reusable timeout-able tcpconn
		t := time.Now().Add(5 * time.Second)
		tcpconn.SetReadDeadline(t)
	}

	// auth
	go func() {
		
	}()

	go func() {
		var resp ServerResp
		for {
			resp.PcSignal = nil
			resp.LocalState = nil

			resetTimeout()

			/*var i interface{}
			err := decoder.Decode(&i)
			log.Fatal(err)*/

			if err := decoder.Decode(&resp); err != nil {
				c.kill <- err
				log.Println("decode err")
				return
			}

			// apparently an error is fatal
			if resp.Error != nil {
				log.Println("error", resp.Error)
				c.kill <- fmt.Errorf("error: %s", resp.Error)
				return
			}

			if resp.LocalState != nil {
				log.Println("client received new local state commandment", resp.LocalState)
				// TODO: We want to "allow" nil don't we?
				// Or only in other direction?
				// Or never after not-nil?
				localStateChan <- resp.LocalState
			}

			if resp.PcSignal != nil {
				log.Println("client receives pcsignal", resp.PcSignal)
				pcSignalChan <- *resp.PcSignal
			}
		}
	}()

	for {
		select {
		case msg := <-c.out:
			if err := encoder.Encode(msg); err != nil {
				log.Println("client encoder err", err)
				return err
			}

		case <-pinger.C:
			log.Print(".")
			c.out <- &ServerReq{}

		case err := <-c.kill:
			// this won't be seen if I run this in a goroutine
			log.Println("bailing")
			return err
		}
	}

	return nil
}
