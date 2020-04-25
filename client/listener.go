// The Warwolf System
// Copyright (C) 2020 The Warwolf Authors

// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.

// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.

// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package client

import (
	ll "log"
	"net"
	"net/url"
	"sync"
	"time"
	"warwolf/buffer"
	"warwolf/cipher"
	"warwolf/dispatch"
	"warwolf/log"
	"warwolf/protocol"
	"warwolf/reader"
	"warwolf/session"
)

const (
	reqDataSize               = requestMaxReqPayloadSize
	reqDefaultNonceVerifySize = 512
	reqDataSafeSize           = reqDataSize - protocol.DialSafeOverheadSize
	reqDataReadDelay          = 100 * time.Millisecond
)

func client(lg log.Log, a socks5Auth, t time.Duration, addr *net.TCPAddr, cc net.Conn, d *dial, b *buffer.Buffer) {
	defer cc.Close()
	req := b.Request()
	defer b.Return(req)
	cc.SetDeadline(time.Now().Add(t))
	lg("Accepted")
	err := socks5(lg, d, b, req, addr, cc, a, socks5TCP, socks5UDP)
	if err != nil {
		lg("Request failed: %s", err)
	} else {
		lg("Request successful")
	}
}

func serve(l *net.TCPListener, d *dial, a socks5Auth, t time.Duration, b *buffer.Buffer) error {
	wg := sync.WaitGroup{}
	defer wg.Wait()
	conns := make(map[uint64]net.Conn, 128)
	connsLock := sync.Mutex{}
	defer func() {
		connsLock.Lock()
		defer connsLock.Unlock()
		for _, v := range conns {
			v.Close()
		}
	}()
	id := uint64(0)
	for {
		conn, e := l.AcceptTCP()
		if e != nil {
			continue
		}
		connsLock.Lock()
		for {
			_, ex := conns[id]
			if ex {
				id++
				continue
			}
			conns[id] = conn
			break
		}
		connsLock.Unlock()
		wg.Add(1)
		go func(i uint64, d *dial, conn *net.TCPConn, b *buffer.Buffer, swg *sync.WaitGroup) {
			defer func() {
				connsLock.Lock()
				defer connsLock.Unlock()
				delete(conns, i)
				swg.Done()
			}()
			conn.SetKeepAlive(true)
			conn.SetKeepAlivePeriod(60 * time.Second)
			addr := conn.LocalAddr().(*net.TCPAddr)
			client(func(format string, v ...interface{}) {
				ll.Printf(conn.RemoteAddr().String()+": "+format, v...)
			}, a, t, addr, reader.NewNetConn(conn), d, b)
		}(id, d, conn, b, &wg)
	}
}

type Listener struct{}

func (s Listener) Listen(config Config) error {
	ll.Printf("Warwolf System is starting up as local socks5 server")
	ll.Printf("(C) 2020 The Warwolf Authors. All rights reserved")
	ll.Printf("The right to communicate freely, privately and securely is essential for everyone")
	c, err := config.Load().Verify()
	if err != nil {
		ll.Fatalf("Configuration error: %s", err)
		return err
	}
	u, e := url.Parse(c.Backend)
	if e != nil {
		ll.Printf("Invalid Backend URL %s: %s", c.Backend, e)
		return e
	}
	ll.Printf("Backend interface: %s", u)
	buf := buffer.New(reqDataSize, c.MaxClientConnections)
	sess := session.NewRetrievers(c.MaxClientConnections)
	defer sess.CloseAll()
	nonce := cipher.NewNonces(reqDefaultNonceVerifySize, &sync.Mutex{})
	dis := dispatch.NewRequester(&sess)
	cc := newDial(func(format string, v ...interface{}) {
		ll.Printf(format, v...)
	}, &buf, u, &sess, &dis, nonce.Verify, c)
	cc.Start()
	defer cc.Stop()
	l, e := net.Listen("tcp", c.Listen)
	if e != nil {
		ll.Printf("Socks5 listen failed: %s", e)
		return e
	}
	ll.Printf("Start Socks5 listening on %s", l.Addr())
	var auth socks5Auth
	if len(c.Username) > 0 || len(c.Password) > 0 {
		auth = func(u, p string) bool {
			return u == c.Username && p == c.Password
		}
		ll.Printf("Socks5 Auth enabled: %s:%s", c.Username, c.Password)
	} else {
		auth = nil
		ll.Printf("Socks5 Auth disabled")
	}
	defer ll.Printf("Shutting down")
	return serve(l.(*net.TCPListener), &cc, auth, c.RequestTimeout, &buf)
}

func New() Listener {
	return Listener{}
}
