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
	"errors"
	"io"
	"net"
	"sync"
	"time"
	"warwolf/buffer"
	"warwolf/log"
	"warwolf/protocol"
	"warwolf/reader"
)

const maxSocks5UDPClients = 12

var (
	ErrNonFirstUDPFragmentNewRequest = errors.New("Socks5 UDP: Received non-first fragment for a new request, ignored")
	ErrInvalidUDPFragmentSequence    = errors.New("Socks5 UDP: Invalid UDP fragment sequence, packet ignored")
	ErrUDPSendNoReceiver             = errors.New("Socks5 UDP: Packet is ignored because there are no receivers")
	ErrTooManyUDPClient              = errors.New("Socks5 UDP: Too many Socks5 UDP clients")
)

func socks5UDPFrag(f byte) (bool, uint8) {
	if f == 0 {
		return true, 0
	}
	const (
		firstBit  byte = 1 << 7
		resetBits      = ^firstBit
	)
	return f&firstBit != 0, f & resetBits
}

type socks5UDPConnReceive struct {
	d    []byte
	done func()
}

type socks5UDPConn struct {
	writeHeader []byte
	client      *net.UDPAddr
	conn        *net.UDPConn
	receive     chan socks5UDPConnReceive
	closer      chan struct{}
	buf         []byte
	readEOF     bool
	closed      bool
	lock        sync.Mutex
}

func (s *socks5UDPConn) Read(b []byte) (int, error) {
	if s.readEOF || len(b) == 0 {
		return 0, io.EOF
	}
	select {
	case bb, ok := <-s.receive:
		if !ok {
			s.readEOF = true
			s.lock.Lock()
			defer s.lock.Unlock()
			s.closed = true
			return 0, io.EOF
		}
		defer bb.done()
		return copy(b, bb.d), nil
	case <-s.closer:
		return 0, io.EOF
	}
}

func (s *socks5UDPConn) Write(b []byte) (int, error) {
	if len(b) == 0 {
		return 0, io.EOF
	}
	l := copy(s.buf, s.writeHeader)
	l += copy(s.buf[l:], b)
	_, _, e := s.conn.WriteMsgUDP(s.buf[:l], nil, s.client)
	if e != nil {
		s.Close()
		return 0, e
	}
	return len(b), nil
}

func (s *socks5UDPConn) Close() error {
	s.lock.Lock()
	defer s.lock.Unlock()
	if s.closed {
		return nil
	}
	s.closed = true
	close(s.closer)
	return nil
}

type socks5UDPClient struct {
	expire  time.Time
	receive chan socks5UDPConnReceive
	frag    byte
}

func (s *socks5UDPClient) handle(client *net.UDPAddr, conn *net.UDPConn, d *dial, writeHeader []byte, tatype protocol.AddressType, taddr []byte, tport uint16, frag byte, reqData []byte, reqDataLen int, bb *buffer.Buffer) error {
	push := bb.Request()
	pushReturned := false
	defer func() {
		if pushReturned {
			return
		}
		bb.Return(push)
	}()
	p := reader.NewPusher(push[:])
	return d.dial(tatype, taddr, tport, reqData, reqDataLen, &p, &socks5UDPConn{
		writeHeader: writeHeader,
		client:      client,
		conn:        conn,
		receive:     s.receive,
		closer:      make(chan struct{}),
		buf:         reqData,
		readEOF:     false,
		lock:        sync.Mutex{},
	}, func() {
		pushReturned = true
		bb.Return(push)
		push = nil
		p = reader.Pusher{}
	})
}

func (s *socks5UDPClient) send(b *buffer.Buffer, frag byte, data []byte) error {
	end, f := socks5UDPFrag(frag)
	if !end && f != s.frag+1 {
		return ErrInvalidUDPFragmentSequence
	}
	bb := b.Request()
	l := copy(bb, data)
	d := socks5UDPConnReceive{
		d:    bb[:l],
		done: func() { b.Return(bb) },
	}
	select {
	case s.receive <- d:
		if end {
			s.frag = 0
		} else {
			s.frag = f
		}
		return nil
	default:
		d.done()
		return ErrUDPSendNoReceiver
	}
}

func (s *socks5UDPClient) close() {
	close(s.receive)
	for d := range s.receive {
		d.done()
	}
}

type socks5UDPServer struct {
	idleTimeout time.Duration
	source      net.IP
	clients     map[string]*socks5UDPClient
	lock        sync.Mutex
	wg          *sync.WaitGroup
}

func (s *socks5UDPServer) create(l *net.UDPConn, d *dial, client *net.UDPAddr, id string, tatype byte, taddr []byte, tport uint16, frag byte, reqData []byte, bb *buffer.Buffer) error {
	fend, f := socks5UDPFrag(frag)
	if !fend || f != 0 {
		return ErrNonFirstUDPFragmentNewRequest
	}
	c := &socks5UDPClient{
		expire:  time.Now().Add(s.idleTimeout),
		receive: make(chan socks5UDPConnReceive, maxSocks5UDPClients),
		frag:    f,
	}
	s.clients[id] = c
	s.wg.Add(1)
	buf := bb.Request()
	ll := copy(buf[:reqDataSafeSize], reqData)
	writeHeader := make([]byte, len(taddr)+4)
	writeHeader[3] = tatype
	copy(writeHeader[4:], taddr)
	go func(l *net.UDPConn, d *dial, client *net.UDPAddr, id string, whd []byte, tatype byte, taddr []byte, tport uint16, frag byte, reqData []byte, reqDataLen int, bb *buffer.Buffer) {
		defer func() {
			bb.Return(buf)
			s.lock.Lock()
			defer s.lock.Unlock()
			delete(s.clients, id)
			s.wg.Done()
		}()
		c.handle(client, l, d, whd, socks5AtypeToProtocolUDPAtype(tatype), taddr, tport, frag, reqData, reqDataLen, bb)
	}(l, d, client, id, writeHeader, tatype, taddr[:len(taddr)-2], tport, frag, buf, ll, bb)
	return nil
}

func (s *socks5UDPServer) dispatch(l *net.UDPConn, d *dial, client *net.UDPAddr, data []byte, bb *buffer.Buffer) error {
	r := reader.NewFetcher(reader.ByteFetch(data, io.EOF))
	b, err := r.Fetch(4)
	if err != nil {
		return err
	}
	frag := b[2]
	atype := b[3]
	addr, port, err := socks5Addr(atype, &r)
	if err != nil {
		return err
	}
	payload, err := r.FetchMax(reader.MaxFetchSize)
	if err != nil || len(payload) == 0 {
		return err
	}
	id := client.String() + " # " + string(addr)
	s.lock.Lock()
	defer s.lock.Unlock()
	c, ex := s.clients[id]
	now := time.Now()
	if !ex {
		for k, v := range s.clients {
			if v.expire.After(now) {
				continue
			}
			v.close()
			delete(s.clients, k)
		}
		if len(s.clients) > maxSocks5UDPClients {
			return ErrTooManyUDPClient
		}
		return s.create(l, d, client, id, atype, addr, port, frag, payload, bb)
	}
	c.expire = time.Now().Add(s.idleTimeout)
	return c.send(bb, frag, payload)
}

func (s *socks5UDPServer) Listen(lg log.Log, d *dial, l *net.UDPConn, bb *buffer.Buffer) error {
	defer func() {
		for _, v := range s.clients {
			v.close()
		}
		s.wg.Wait()
	}()
	b := bb.Request()
	defer bb.Return(b)
	for {
		ll, caddr, e := l.ReadFrom(b)
		if e != nil {
			return e
		}
		if !s.source.Equal(caddr.(*net.UDPAddr).IP) {
			continue
		}
		e = s.dispatch(l, d, caddr.(*net.UDPAddr), b[:ll], bb)
		if e == nil {
			lg("UDP packet from %s was dispatched", caddr)
			continue
		}
		lg("UDP packet from %s was failed to dispatched: %s", caddr, e)
	}
}

func socks5UDP(lg log.Log, d *dial, b *buffer.Buffer, laddr *net.TCPAddr, bb []byte, atype byte, addr []byte, port uint16, r net.Conn) error {
	l, e := net.ListenUDP("udp", &net.UDPAddr{
		IP:   laddr.IP,
		Port: 0,
		Zone: laddr.Zone,
	})
	if e != nil {
		return e
	}
	defer l.Close()
	udpaddr := l.LocalAddr().(*net.UDPAddr)
	lg("Listening UDP on %s", udpaddr)
	defer lg("UDP server is closed")
	resp, isip4 := socks5BuildAddrFromIP(4, udpaddr.IP, uint16(udpaddr.Port))
	resp[0] = 5
	if isip4 {
		resp[3] = socks5ATypeIPv4
	} else {
		resp[3] = socks5ATypeIPv6
	}
	_, e = r.Write(resp)
	if e != nil {
		return e
	}
	r.SetDeadline(time.Time{})
	wg := sync.WaitGroup{}
	defer wg.Wait()
	wg.Add(1)
	go func() {
		defer func() {
			l.Close()
			wg.Done()
		}()
		buf := [1]byte{}
		for {
			_, e := r.Read(buf[:])
			if e == nil {
				continue
			}
			return
		}
	}()
	listen := socks5UDPServer{
		idleTimeout: 60 * time.Second,
		source:      r.RemoteAddr().(*net.TCPAddr).IP,
		clients:     make(map[string]*socks5UDPClient, maxSocks5UDPClients),
		lock:        sync.Mutex{},
		wg:          &wg,
	}
	return listen.Listen(lg, d, l, b)
}
