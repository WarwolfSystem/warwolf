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
	"io"
	"net/url"
	"sync"
	"warwolf/buffer"
	"warwolf/cipher"
	"warwolf/dispatch"
	"warwolf/log"
	"warwolf/protocol"
	"warwolf/reader"
	"warwolf/session"
)

type dial struct {
	lg             log.Log
	requester      requester
	maxRetrieveLen uint16
}

func (d *dial) Start() {
	d.requester.init()
}

func (d *dial) Stop() {
	d.requester.kill()
}

func (d *dial) dial(
	aTyp protocol.AddressType,
	addr []byte,
	port uint16,
	reqData []byte,
	reqDataLen int,
	p *reader.Pusher,
	hosted io.ReadWriteCloser,
	after func(),
) error {
	rr := protocol.DialRequest{
		ID:             protocol.ID{},
		ATyp:           aTyp,
		Addr:           addr,
		Port:           port,
		MaxRetrieveLen: d.maxRetrieveLen,
		Request:        reqData[:reqDataLen],
		RequestLength:  uint16(reqDataLen),
	}
	wg := sync.WaitGroup{}
	defer wg.Wait()
	ret, err := d.requester.dial(rr, p, func(id protocol.ID) session.Retriever {
		return &dialedConn{
			id:         id,
			maxSendLen: d.maxRetrieveLen,
			requester:  &d.requester,
			hosted:     hosted,
			rwg:        sync.WaitGroup{},
			wg:         &wg,
		}
	}, after)
	if err != nil {
		return err
	}
	return ret.Serve(reqData)
}

func newDial(
	lg log.Log,
	b *buffer.Buffer,
	url *url.URL,
	session *session.Retrievers,
	dispatch *dispatch.Requester,
	nv cipher.NonceVerifier,
	c Config,
) dial {
	maxRetrieveLen := c.MaxRetrieveLength
	if maxRetrieveLen > requestMaxReqPayloadSize {
		maxRetrieveLen = requestMaxReqPayloadSize
	}
	return dial{
		lg:             lg,
		requester:      newRequester(lg, b, url, session, dispatch, nv, c),
		maxRetrieveLen: maxRetrieveLen,
	}
}
