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

package dispatch

import (
	"net"
	"sync"
	"warwolf/buffer"
	"warwolf/log"
	"warwolf/protocol"
	"warwolf/reader"
	"warwolf/relay"
	"warwolf/session"
)

type Responder struct {
	sessions *session.Sessions
	laddr    net.Addr
	rconfig  relay.Config
	buffer   *buffer.Buffer
}

func NewResponder(
	sessions *session.Sessions,
	laddr net.Addr,
	rconfig relay.Config,
	buffer *buffer.Buffer,
) Responder {
	return Responder{
		sessions: sessions,
		laddr:    laddr,
		rconfig:  rconfig,
		buffer:   buffer,
	}
}

func (r *Responder) handle(lg log.Log, rType byte, rData byte, rr *reader.Fetcher, pp Pusher, wg *sync.WaitGroup, retrieverCancels *session.RetrieverCancels, c *Config) error {
	switch rType {
	case protocol.DialType:
		req := protocol.DialRequest{}
		err := req.Parse(protocol.AddressType(rData), rr, func(d *protocol.DialRequest, rr []byte) error {
			lg("%s: Dial", d.ID)
			wg.Add(1)
			r.sessions.Register(d, rr, r.laddr, r.rconfig, r.buffer, func(rerrcode byte, rsp protocol.DialRespond) {
				defer wg.Done()
				rerr := pp(func(p *reader.Pusher) error {
					return rsp.Build(d.ID, rerrcode, p)
				})
				if rerr != nil {
					lg("%s: Dial: Error: %s", d.ID, rerr)
				} else {
					lg("%s: Dial: Successful(%d)", d.ID, rerrcode)
				}
			}, c.MaxRetrieveLen)
			return nil
		})
		if err != nil {
			lg("Invalid dial: %s", err)
		}
		return err

	case protocol.RetrieveType:
		req := protocol.RetrieveRequest{}
		err := req.Parse(rr)
		if err != nil {
			lg("Invalid retrieve request: %s", err)
			return err
		}
		lg("%s: Retrieve request", req.ID)
		wg.Add(1)
		r.sessions.Retrieve(req, func(rerrcode byte, rsp protocol.RetrieveRespond) {
			defer wg.Done()
			rerr := pp(func(p *reader.Pusher) error {
				return rsp.Build(req.ID, rerrcode, p)
			})
			if rerr != nil {
				lg("%s: Retrieve request: Error %s", req.ID, rerr)
			} else {
				lg("%s: Retrieve request: Responded(%d)", req.ID, rerrcode)
			}
		}, c.MaxRetrieveLen)
		return nil

	case protocol.ResumeType:
		req := protocol.ResumeRequest{}
		err := req.Parse(rr)
		if err != nil {
			lg("Invalid resume request: %s", err)
			return err
		}
		lg("%s: Resume request", req.ID)
		wg.Add(1)
		r.sessions.Resume(req, func(rerrcode byte, rsp protocol.ResumeRespond) {
			defer wg.Done()
			rerr := pp(func(p *reader.Pusher) error {
				return rsp.Build(req.ID, rerrcode, p)
			})
			if rerr != nil {
				lg("%s: Resume request: Error %s", req.ID, rerr)
			} else {
				lg("%s: Resume request: Responded(%d)", req.ID, rerrcode)
			}
		}, c.MaxRetrieveLen)
		return nil

	case protocol.SendType:
		req := protocol.SendRequest{}
		err := req.Parse(rr, func(d *protocol.SendRequest, rr []byte) error {
			wg.Add(1)
			defer wg.Done()
			lg("%s: Send request", d.ID)
			rerrcode, rsp := r.sessions.Send(*d, rr)
			werr := pp(func(p *reader.Pusher) error {
				return rsp.Build(d.ID, rerrcode, p)
			})
			if werr != nil {
				lg("%s: Send request: Error: %s", d.ID, werr)
			} else {
				lg("%s: Send request: Responded(%d)", d.ID, rerrcode)
			}
			return nil
		})
		if err != nil {
			lg("Invalid send request: %s", err)
		}
		return err

	case protocol.CloseType:
		req := protocol.CloseRequest{}
		err := req.Parse(rr)
		if err != nil {
			lg("Invalid close request: %s", err)
			return err
		}
		lg("%s: Close request", req.ID)
		rerrcode, rsp := r.sessions.Close(req)
		err = pp(func(p *reader.Pusher) error {
			return rsp.Build(req.ID, rerrcode, p)
		})
		if err != nil {
			lg("%s: Close request: Error: %s", req.ID, err)
		} else {
			lg("%s: Close request: Responded(%d)", rsp.ID, rerrcode)
		}
		return err

	default:
		return ErrUnknownRequestType
	}
}

func (r *Responder) Dispatch(lg log.Log, req []byte, p Pusher, c Config) error {
	wg := sync.WaitGroup{}
	defer wg.Wait()
	return dispatch(lg, req, r.handle, p, &wg, true, nil, c)
}
