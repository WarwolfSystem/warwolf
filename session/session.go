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

package session

import (
	"errors"
	"net"
	"sync"
	"time"
	"warwolf/buffer"
	"warwolf/protocol"
	"warwolf/relay"
)

var (
	ErrInvalidProtocol = errors.New("Session: Invalid protocol")
	ErrClosed          = errors.New("Session: Resouce already been released")
	ErrExpired         = errors.New("Session: Resouce already expired")
)

const (
	initialConnectWait = 300 * time.Millisecond
)

type session struct {
	expired time.Time
	relay   relay.Relay
	wg      sync.WaitGroup
	l       sync.Mutex
	maxrlen uint16
	rid     uint64
	rbusy   bool
	read    []byte
	readLen uint16
	rpaused bool
	wid     uint64
	wbusy   bool
	wlen    uint16
	closed  bool
}

func (s *session) serve(c func() relay.Error, after func(e relay.Error)) {
	s.wg.Add(1)
	go func() {
		done := false
		defer func() {
			if done {
				return
			}
			s.wg.Done()
		}()
		e := c()
		done = true
		s.wg.Done()
		after(e)
	}()
}

func (s *session) readData(offset uint16, max int) (uint16, uint16, []byte) {
	rstart := offset
	if rstart > s.readLen {
		rstart = s.readLen
	}
	maxlen := s.readLen - rstart
	if maxlen > s.maxrlen {
		maxlen = s.maxrlen
	}
	if maxlen > uint16(max) {
		maxlen = uint16(max)
	}
	return s.readLen, rstart, s.read[rstart : rstart+maxlen]
}

func (s *session) start(r *protocol.DialRequest, d []byte, b *buffer.Buffer, rconfig relay.Config, result func(byte, protocol.DialRespond), remover func(), maxresplen int) {
	resultOnce := sync.Once{}
	var connectionErr error
	s.serve(func() relay.Error {
		rbuf := b.Request()
		defer b.Return(rbuf)
		return s.relay.Serve(rbuf, rconfig, func(c net.Conn, err error) {
			if err != nil {
				connectionErr = err
				return
			}
			if len(d) == 0 {
				return
			}
			c.Write(d)
		})
	}, func(e relay.Error) {
		if connectionErr == nil {
			return
		}
		remover()
		resultOnce.Do(func() {
			result(protocol.DialErrorUnreachable, r.Respond(0, 0, nil))
		})
		return
	})
	s.retrieve(r.RetrieveRequest(), func(s *session) byte {
		return 0
	}, initialConnectWait, func(e byte, rsp protocol.RetrieveRespond) {
		resultOnce.Do(func() {
			result(0, r.RetrieveRespond(rsp))
		})
	}, maxresplen)
}

func (s *session) retrieve(d protocol.RetrieveRequest, call func(s *session) byte, timeout time.Duration, result func(byte, protocol.RetrieveRespond), maxlen int) {
	ll := lll(&s.l)
	ll.lock()
	defer ll.unlock()
	if s.closed {
		rsp := d.Respond(s.rid, s.readLen, 0, nil)
		ll.unlock()
		result(protocol.ResourceErrorNotFound, rsp)
		return
	}
	if s.rbusy {
		rsp := d.Respond(s.rid, s.readLen, 0, nil)
		ll.unlock()
		result(protocol.ResourceErrorNotReady, rsp)
		return
	}
	if d.RID < s.rid {
		rsp := d.Respond(s.rid, s.readLen, 0, nil)
		ll.unlock()
		result(protocol.ResourceErrorExpired, rsp)
		return
	}
	cerr := call(s)
	if cerr > 0 {
		rsp := d.Respond(s.rid, s.readLen, 0, nil)
		ll.unlock()
		result(cerr, rsp)
		return
	}
	if s.rpaused {
		total, offset, data := s.readData(d.Offset, maxlen)
		rsp := d.Respond(s.rid, total, offset, data)
		ll.unlock()
		result(0, rsp)
		return
	}
	s.rpaused = true
	s.rbusy = true
	ll.unlock()
	s.relay.Retrieve(func(r []byte, err relay.Error) {
		ll := lll(&s.l)
		ll.lock()
		defer ll.unlock()
		s.rbusy = false
		if err.IsError() {
			s.read = nil
			s.readLen = 0
			rsp := d.Respond(s.rid, s.readLen, 0, nil)
			ll.unlock()
			if err.IsTimeout {
				result(0, rsp)
			} else {
				result(protocol.ResourceErrorBroken, rsp)
			}
			return
		}
		s.read = r
		s.readLen = uint16(len(r))
		total, offset, data := s.readData(d.Offset, maxlen)
		rsp := d.Respond(s.rid, total, offset, data)
		ll.unlock()
		result(0, rsp)
	}, timeout)
}

func (s *session) resume(d protocol.ResumeRequest, timeout time.Duration, result func(byte, protocol.ResumeRespond), maxlen int) {
	dd := d.RetrieveRequest()
	s.retrieve(dd, func(s *session) byte {
		if d.RID != s.rid {
			return protocol.ResourceErrorExpired
		}
		if !s.rpaused {
			return protocol.ResourceErrorNotReady
		}
		s.rid++
		s.rpaused = false
		return protocol.ResourceErrorSuccess
	}, timeout, func(cerr byte, rsp protocol.RetrieveRespond) {
		result(cerr, d.RetrieveRespond(rsp))
	}, maxlen)
}

func (s *session) send(d protocol.SendRequest, r []byte, timeout time.Duration) (byte, protocol.SendRespond) {
	ll := lll(&s.l)
	ll.lock()
	defer ll.unlock()
	if s.closed {
		return protocol.ResourceErrorNotFound, d.Respond(s.wid, s.wlen)
	}
	if d.WID < s.wid {
		return protocol.ResourceErrorExpired, d.Respond(s.wid, s.wlen)
	}
	if s.wbusy {
		return protocol.ResourceErrorNotReady, d.Respond(s.wid, s.wlen)
	}
	s.wbusy = true
	ll.unlock()
	wlen, werr := s.relay.Send(r)
	if werr.IsError() {
		ll.lock()
		s.wbusy = false
		return protocol.ResourceErrorSendFailure, d.Respond(s.wid, s.wlen)
	}
	ll.lock()
	s.wbusy = false
	s.wid++
	s.wlen = uint16(wlen)
	return 0, d.Respond(s.wid, s.wlen)
}

func (s *session) kill() {
	ll := lll(&s.l)
	ll.lock()
	defer ll.unlock()
	if s.closed {
		return
	}
	s.closed = true
	s.relay.Close()
}

func (s *session) release() {
	s.kill()
	s.wg.Wait()
}

func (s *session) close(d protocol.CloseRequest) (byte, protocol.CloseRespond) {
	s.release()
	return 0, d.Respond()
}

func newSession(relay relay.Relay, maxrlen uint16, expired time.Time) *session {
	return &session{
		expired: expired,
		relay:   relay,
		wg:      sync.WaitGroup{},
		l:       sync.Mutex{},
		maxrlen: maxrlen,
		rid:     0,
		rbusy:   false,
		read:    nil,
		readLen: 0,
		rpaused: false,
		wid:     0,
		wbusy:   false,
		wlen:    0,
		closed:  false,
	}
}
