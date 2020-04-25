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
	"crypto/rand"
	"errors"
	"io"
	"sync"
	"warwolf/protocol"
	"warwolf/reader"
)

var (
	ErrRetrieverUndefined = errors.New("Session: Undefined retriever")
	ErrRetrieverBusy      = errors.New("Session: Retriever is busy")
	ErrNotReady           = errors.New("Session: Resource is not ready")
	ErrUnableToGenerateID = errors.New("Session: Unable to generate ID")
	ErrRetrieversFull     = errors.New("Session: Retrievers is full")
)

func newID() (protocol.ID, error) {
	id := protocol.ID{}
	_, err := io.ReadFull(rand.Reader, id[:])
	return id, err
}

type Requester interface {
	Retrieve(id protocol.ID, offset uint16, p *reader.Pusher) error
	Resume(id protocol.ID, p *reader.Pusher) error
	Close(id protocol.ID, p *reader.Pusher) error
}

type RetrieverBuilder func(id protocol.ID) Retriever
type RetrieverCancel func(e RetrieverError)

type RetrieverCancels map[protocol.ID]RetrieverCancel

func (r *RetrieverCancels) clear(id protocol.ID) {
	delete(*r, id)
}

func (r *RetrieverCancels) SettleAll(e RetrieverError) {
	for k, v := range *r {
		v(e)
		delete(*r, k)
	}
}

func (r *RetrieverCancels) Append(id protocol.ID, c RetrieverCancel) {
	(*r)[id] = c
}

type Retrievers struct {
	capacity int
	sessions map[protocol.ID]*retriever
	lock     *sync.Mutex
}

func (r *Retrievers) newID() (protocol.ID, error) {
	const retries = 1000
	for i := 0; i < retries; i++ {
		i, err := newID()
		if err != nil {
			return protocol.ID{}, err
		}
		_, ex := r.sessions[i]
		if ex {
			continue
		}
		return i, nil
	}
	return protocol.ID{}, ErrUnableToGenerateID
}

func (r *Retrievers) reactToError(id protocol.ID, e error) (func(), error) {
	if isErrorRecoverable(e) {
		return nil, e
	}
	u := r.forcefullyRelease(id)
	return u, e
}

func (r *Retrievers) runExec(exec func()) {
	if exec == nil {
		return
	}
	exec()
}

func (r *Retrievers) Retriever(builder RetrieverBuilder) (protocol.ID, Retriever, error) {
	ll := lll(r.lock)
	ll.lock()
	defer ll.unlock()
	if len(r.sessions) >= r.capacity {
		return protocol.ID{}, nil, ErrRetrieversFull
	}
	id, err := r.newID()
	if err != nil {
		return id, nil, err
	}
	rec := builder(id)
	ret := newRetriever(rec)
	r.sessions[id] = ret
	return id, rec, nil
}

func (r *Retrievers) Register(id protocol.ID, rr protocol.DialRequest, p *reader.Pusher, rec Retriever, c RetrieverDialResult) (RetrieverCancel, error) {
	ll := lll(r.lock)
	ll.lock()
	defer ll.unlock()
	s, ex := r.sessions[id]
	if !ex {
		c(newRetrieverError(ErrRetrieverUndefined, false))
		return nil, ErrRetrieverUndefined
	}
	if s.dialcb != nil {
		c(newRetrieverError(ErrRetrieverBusy, false))
		return nil, ErrRetrieverBusy
	}
	err := rr.Build(id, p)
	if err != nil {
		c(newRetrieverError(err, false))
		return nil, err
	}
	s.dialcb = c
	return func(e RetrieverError) {
		ll := lll(r.lock)
		ll.lock()
		defer ll.unlock()
		dialcb := s.dialcb
		s.dialcb = nil
		ll.unlock()
		if dialcb == nil {
			return
		}
		dialcb(e)
	}, nil
}

func (r *Retrievers) Registered(e byte, d *protocol.DialRespond, rr *reader.Fetcher, c *RetrieverCancels) error {
	var u func() = nil
	defer func() { r.runExec(u) }()
	ll := lll(r.lock)
	ll.lock()
	defer ll.unlock()
	s, ex := r.sessions[d.ID]
	if !ex {
		return ErrRetrieverUndefined
	}
	return s.dialed(e, d, rr, &ll, func(e error) {
		u, _ = r.reactToError(d.ID, e)
	}, c)
}

func (r *Retrievers) Retrieve(id protocol.ID, p *reader.Pusher, c RetrieverRetrieveResult) (RetrieverCancel, error) {
	ll := lll(r.lock)
	ll.lock()
	defer ll.unlock()
	s, ex := r.sessions[id]
	if !ex {
		c(newRetrieverError(ErrRetrieverUndefined, false))
		return nil, ErrRetrieverUndefined
	}
	if s.rcb != nil {
		c(newRetrieverError(ErrRetrieverBusy, false))
		return nil, ErrRetrieverBusy
	}
	var err error
	if s.roffset >= s.rtotal {
		rr := protocol.ResumeRequest{
			ID:  id,
			RID: s.rid,
		}
		err = rr.Build(id, p)
	} else {
		rr := protocol.RetrieveRequest{
			ID:     id,
			RID:    s.rid,
			Offset: s.roffset,
		}
		err = rr.Build(id, p)
	}
	if err != nil {
		c(newRetrieverError(err, false))
		return nil, err
	}
	s.rcb = c
	return func(e RetrieverError) {
		ll := lll(r.lock)
		ll.lock()
		defer ll.unlock()
		rcb := s.rcb
		s.rcb = nil
		ll.unlock()
		if rcb == nil {
			return
		}
		rcb(e)
	}, nil
}

func (r *Retrievers) Retrieved(e byte, d *protocol.RetrieveRespond, rr *reader.Fetcher, c *RetrieverCancels) error {
	var u func() = nil
	defer func() { r.runExec(u) }()
	ll := lll(r.lock)
	ll.lock()
	defer ll.unlock()
	s, ex := r.sessions[d.ID]
	if !ex {
		return ErrRetrieverUndefined
	}
	return s.retrieved(e, d, rr, &ll, func(e error) {
		u, _ = r.reactToError(d.ID, e)
	}, c)
}

func (r *Retrievers) Resumed(e byte, d *protocol.ResumeRespond, rr *reader.Fetcher, c *RetrieverCancels) error {
	var u func() = nil
	defer func() { r.runExec(u) }()
	ll := lll(r.lock)
	ll.lock()
	defer ll.unlock()
	s, ex := r.sessions[d.ID]
	if !ex {
		return ErrRetrieverUndefined
	}
	return s.resumed(e, d, rr, &ll, func(e error) {
		u, _ = r.reactToError(d.ID, e)
	}, c)
}

func (r *Retrievers) Send(id protocol.ID, data []byte, p *reader.Pusher, c RetrieverSendResult) (RetrieverCancel, error) {
	ll := lll(r.lock)
	ll.lock()
	defer ll.unlock()
	s, ex := r.sessions[id]
	if !ex {
		c(0, newRetrieverError(ErrRetrieverUndefined, false))
		return nil, ErrRetrieverUndefined
	}
	if s.wcb != nil {
		c(0, newRetrieverError(ErrRetrieverBusy, false))
		return nil, ErrRetrieverBusy
	}
	rr := protocol.SendRequest{
		ID:            id,
		WID:           s.wid,
		Payload:       data[protocol.SendHeaderOverhead:],
		PayloadLength: uint16(len(data) - protocol.SendHeaderOverhead),
	}
	err := rr.BuildHeader(id, p)
	if err != nil {
		c(0, newRetrieverError(err, false))
		return nil, err
	}
	s.wcb = c
	return func(e RetrieverError) {
		ll := lll(r.lock)
		ll.lock()
		defer ll.unlock()
		wcb := s.wcb
		s.wcb = nil
		ll.unlock()
		if wcb == nil {
			return
		}
		wcb(0, e)
	}, nil
}

func (r *Retrievers) Sent(e byte, d *protocol.SendRespond, c *RetrieverCancels) error {
	var u func() = nil
	defer func() { r.runExec(u) }()
	ll := lll(r.lock)
	ll.lock()
	defer ll.unlock()
	s, ex := r.sessions[d.ID]
	if !ex {
		return ErrRetrieverUndefined
	}
	return s.sent(e, d, &ll, func(e error) {
		u, _ = r.reactToError(d.ID, e)
	}, c)
}

func (r *Retrievers) Close(id protocol.ID, p *reader.Pusher, c RetrieverCloseResult) (RetrieverCancel, error) {
	ll := lll(r.lock)
	ll.lock()
	defer ll.unlock()
	s, ex := r.sessions[id]
	if !ex {
		rr := protocol.CloseRequest{
			ID: id,
		}
		err := rr.Build(id, p)
		if err != nil {
			return nil, err
		}
		c(newRetrieverError(ErrRetrieverUndefined, false))
		return func(e RetrieverError) {}, nil
	}
	if s.ccb != nil {
		c(newRetrieverError(ErrRetrieverBusy, false))
		return nil, ErrRetrieverBusy
	}
	rr := protocol.CloseRequest{
		ID: id,
	}
	err := rr.Build(id, p)
	if err != nil {
		c(newRetrieverError(err, false))
		return nil, err
	}
	s.ccb = c
	return func(e RetrieverError) {
		ll := lll(r.lock)
		ll.lock()
		defer ll.unlock()
		ccb := s.ccb
		s.ccb = nil
		ll.unlock()
		if ccb == nil {
			return
		}
		ccb(e)
	}, nil
}

func (r *Retrievers) release(id protocol.ID) (*retriever, error) {
	s, ex := r.sessions[id]
	if !ex {
		return nil, ErrRetrieverUndefined
	}
	delete(r.sessions, id)
	return s, nil
}

func (r *Retrievers) forcefullyRelease(id protocol.ID) func() {
	rr, err := r.release(id)
	if err != nil {
		return func() {}
	}
	return rr.release()
}

func (r *Retrievers) Closed(e byte, d *protocol.CloseRespond, c *RetrieverCancels) error {
	var u func() = nil
	defer func() { r.runExec(u) }()
	ll := lll(r.lock)
	ll.lock()
	defer ll.unlock()
	rr, err := r.release(d.ID)
	if err != nil {
		return err
	}
	return rr.closed(e, d, &ll, func(e error) {
		u, _ = r.reactToError(d.ID, e)
	}, c)
}

func (r *Retrievers) Release(id protocol.ID, c func(t Retriever) error) error {
	ll := lll(r.lock)
	ll.lock()
	defer ll.unlock()
	rr, err := r.release(id)
	if err != nil {
		return err
	}
	return c(rr.rec)
}

func (r *Retrievers) CloseAll() {
	ll := lll(r.lock)
	ll.lock()
	defer ll.unlock()
	cb := make([]func(), 0, len(r.sessions))
	for k, v := range r.sessions {
		u := v.release()
		delete(r.sessions, k)
		cb = append(cb, u)
	}
	ll.unlock()
	for i := range cb {
		cb[i]()
	}
}

func NewRetrievers(size int) Retrievers {
	return Retrievers{
		capacity: size,
		sessions: make(map[protocol.ID]*retriever, size),
		lock:     &sync.Mutex{},
	}
}
