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
	"net"
	"sync"
	"time"
	"warwolf/buffer"
	"warwolf/protocol"
	"warwolf/relay"
)

type Sessions struct {
	idleTimeout time.Duration
	sessions    map[protocol.ID]*session
	lock        sync.Mutex
	capacity    int
}

func New(capacity int, idleTimeout time.Duration) Sessions {
	return Sessions{
		idleTimeout: idleTimeout,
		sessions:    make(map[protocol.ID]*session, capacity),
		lock:        sync.Mutex{},
		capacity:    capacity,
	}
}

func (s *Sessions) reactToError(id protocol.ID, e error) error {
	if isErrorRecoverable(e) {
		return e
	}
	s.forceRemove(id)
	return e
}

func (s *Sessions) Register(r *protocol.DialRequest, d []byte, laddr net.Addr, rconfig relay.Config, b *buffer.Buffer, result func(byte, protocol.DialRespond), maxresplen int) {
	ll := lll(&s.lock)
	ll.lock()
	defer ll.unlock()
	ss, ex := s.sessions[r.ID]
	if ex {
		ll.unlock()
		s.lock.Lock()
		rid := ss.rid
		s.lock.Unlock()
		if rid == 0 {
			s.doRetrieve(ss, r.RetrieveRequest(), func(e byte, rsp protocol.RetrieveRespond) {
				result(0, r.RetrieveRespond(rsp))
			}, maxresplen)
			return
		}
		result(protocol.DialErrorAlreadyDialed, r.Respond(rid, 0, nil))
		return
	}
	if len(s.sessions) >= s.capacity {
		result(protocol.DialErrorOverCapacity, r.Respond(0, 0, nil))
		return
	}
	addr, e := buildAddr(r.ATyp, r.Addr, r.Port)
	if e != nil {
		result(protocol.DialErrorInvalidRequest, r.Respond(0, 0, nil))
		return
	}
	relay, re := buildRelay(r, laddr, addr)
	if re.IsError() {
		result(protocol.DialErrorInternalFailure, r.Respond(0, 0, nil))
		return
	}
	ss = newSession(
		relay,
		r.MaxRetrieveLen,
		time.Now().Add(s.idleTimeout),
	)
	s.sessions[r.ID] = ss
	ll.unlock()
	ss.start(r, d, b, rconfig, func(b byte, d protocol.DialRespond) {
		result(b, d)
		if b == 0 {
			return
		}
		s.reactToError(r.ID, getRetrieverDialError(b))
	}, func() {
		s.forceRemove(r.ID)
	}, maxresplen)
}

func (s *Sessions) doRetrieve(ss *session, d protocol.RetrieveRequest, r func(byte, protocol.RetrieveRespond), maxlen int) {
	ss.retrieve(d, func(s *session) byte {
		return protocol.ResourceErrorSuccess
	}, 0, func(b byte, d protocol.RetrieveRespond) {
		r(b, d)
		if b == 0 {
			return
		}
		s.reactToError(d.ID, getRetrieverResourceError(b))
	}, maxlen)
}

func (s *Sessions) Retrieve(d protocol.RetrieveRequest, r func(byte, protocol.RetrieveRespond), maxlen int) {
	ll := lll(&s.lock)
	ll.lock()
	defer ll.unlock()
	ss, ex := s.sessions[d.ID]
	if !ex {
		r(protocol.ResourceErrorNotFound, d.Respond(0, 0, 0, nil))
		return
	}
	ss.expired = time.Now().Add(s.idleTimeout)
	ll.unlock()
	s.doRetrieve(ss, d, r, maxlen)
}

func (s *Sessions) Resume(d protocol.ResumeRequest, r func(byte, protocol.ResumeRespond), maxlen int) {
	ll := lll(&s.lock)
	ll.lock()
	defer ll.unlock()
	ss, ex := s.sessions[d.ID]
	if !ex {
		r(protocol.ResourceErrorNotFound, d.Respond(0, 0, nil))
		return
	}
	ss.expired = time.Now().Add(s.idleTimeout)
	ll.unlock()
	ss.resume(d, 0, func(b byte, d protocol.ResumeRespond) {
		r(b, d)
		if b == 0 {
			return
		}
		s.reactToError(d.ID, getRetrieverResourceError(b))
	}, maxlen)
}

func (s *Sessions) Send(r protocol.SendRequest, d []byte) (byte, protocol.SendRespond) {
	ll := lll(&s.lock)
	ll.lock()
	defer ll.unlock()
	ss, ex := s.sessions[r.ID]
	if !ex {
		return protocol.ResourceErrorNotFound, r.Respond(0, 0)
	}
	ll.unlock()
	return ss.send(r, d, 0)
}

func (s *Sessions) kill(id protocol.ID) *session {
	ll := lll(&s.lock)
	ll.lock()
	defer ll.unlock()
	ss, ex := s.sessions[id]
	if !ex {
		return nil
	}
	delete(s.sessions, id)
	return ss
}

func (s *Sessions) forceRemove(id protocol.ID) {
	k := s.kill(id)
	if k == nil {
		return
	}
	k.kill()
}

func (s *Sessions) Close(r protocol.CloseRequest) (byte, protocol.CloseRespond) {
	ss := s.kill(r.ID)
	if ss == nil {
		return protocol.ResourceErrorNotFound, r.Respond()
	}
	cerr, cc := ss.close(r)
	return cerr, cc
}

func (s *Sessions) Recycle() error {
	n := time.Now()
	ll := lll(&s.lock)
	ll.lock()
	defer ll.unlock()
	recycled := make([]*session, 0, len(s.sessions))
	for k, v := range s.sessions {
		if !n.After(v.expired) {
			continue
		}
		delete(s.sessions, k)
		recycled = append(recycled, v)
	}
	ll.unlock()
	for i := range recycled {
		recycled[i].release()
	}
	return nil
}

func (s *Sessions) CloseAll() {
	ll := lll(&s.lock)
	ll.lock()
	defer ll.unlock()
	recycled := make([]*session, 0, len(s.sessions))
	for k, v := range s.sessions {
		delete(s.sessions, k)
		recycled = append(recycled, v)
	}
	ll.unlock()
	for i := range recycled {
		recycled[i].release()
	}
}
