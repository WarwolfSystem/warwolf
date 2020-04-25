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
	"sync"
	"warwolf/log"
	"warwolf/protocol"
	"warwolf/reader"
	"warwolf/session"
)

type Requester struct {
	retrievers *session.Retrievers
}

func (r *Requester) handle(lg log.Log, rType byte, rData byte, rr *reader.Fetcher, pp Pusher, wg *sync.WaitGroup, retrieverCancels *session.RetrieverCancels, c *Config) error {
	switch rType {
	case protocol.DialType:
		lg("Dial respond received")
		rsp := protocol.DialRespond{}
		err := rsp.Parse(rr, func(d *protocol.DialRespond, rr *reader.Fetcher) error {
			return r.retrievers.Registered(rData, d, rr, retrieverCancels)
		})
		if err != nil {
			lg("Invalid dial respond: %s", err)
		}
		return err

	case protocol.RetrieveType:
		lg("Retrieve respond received")
		rsp := protocol.RetrieveRespond{}
		err := rsp.Parse(rr, func(d *protocol.RetrieveRespond, rr *reader.Fetcher) error {
			return r.retrievers.Retrieved(rData, d, rr, retrieverCancels)
		})
		if err != nil {
			lg("Invalid retrieve respond: %s", err)
		}
		return err

	case protocol.ResumeType:
		lg("Resume respond received")
		rsp := protocol.ResumeRespond{}
		err := rsp.Parse(rr, func(d *protocol.ResumeRespond, rr *reader.Fetcher) error {
			return r.retrievers.Resumed(rData, d, rr, retrieverCancels)
		})
		if err != nil {
			lg("Invalid resume respond: %s", err)
		}
		return err

	case protocol.SendType:
		lg("Send respond received")
		rsp := protocol.SendRespond{}
		err := rsp.Parse(rr)
		if err != nil {
			lg("Invalid send respond: %s", err)
			return err
		}
		return r.retrievers.Sent(rData, &rsp, retrieverCancels)

	case protocol.CloseType:
		lg("Close respond received")
		rsp := protocol.CloseRespond{}
		err := rsp.Parse(rr)
		if err != nil {
			lg("Invalid close respond: %s", err)
			return err
		}
		return r.retrievers.Closed(rData, &rsp, retrieverCancels)

	default:
		return ErrUnknownRequestType
	}
}

func (r *Requester) Dispatch(lg log.Log, req []byte, retrieverCancels *session.RetrieverCancels) error {
	return dispatch(lg, req, r.handle, nil, nil, false, retrieverCancels, Config{})
}

func NewRequester(retrievers *session.Retrievers) Requester {
	return Requester{
		retrievers: retrievers,
	}
}
