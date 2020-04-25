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
	"sync"
	"warwolf/protocol"
	"warwolf/reader"
)

type dialedConn struct {
	id         protocol.ID
	maxSendLen uint16
	requester  *requester
	hosted     io.ReadWriteCloser
	rwg        sync.WaitGroup
	wg         *sync.WaitGroup
}

func (d *dialedConn) Serve(buf []byte) error {
	defer func() {
		p := reader.NewPusher(buf[:])
		d.requester.close(d.id, &p)
	}()
	maxSendLen := len(buf) - protocol.SendHeaderOverhead
	if maxSendLen > int(d.maxSendLen) {
		maxSendLen = int(d.maxSendLen)
	}
	maxSendEnd := protocol.SendHeaderOverhead + maxSendLen
	rbuf := buf[protocol.SendHeaderOverhead:maxSendEnd]
	for {
		l, err := d.hosted.Read(rbuf)
		if err != nil {
			return err
		}
		writeLen := 0
		for writeLen < l {
			p := reader.NewPusher(buf[:])
			wlen, err := d.requester.send(d.id, buf[writeLen:protocol.SendHeaderOverhead+l], &p)
			if err != nil {
				return err
			}
			writeLen += wlen
		}
	}
}

func (d *dialedConn) retriever() {
	defer func() {
		d.Close()
		d.wg.Done()
	}()
	buf := [protocol.RetrieveRequestOverhead]byte{}
	p := reader.NewPusher(buf[:])
	for {
		e := d.requester.retrieve(d.id, &p)
		if e == nil {
			continue
		}
		return
	}
}

func (d *dialedConn) Dialed() {
	d.wg.Add(1)
	go d.retriever()
}

func (d *dialedConn) Retrieving(on bool) {
	if on {
		d.rwg.Add(1)
		return
	}
	d.rwg.Done()
}

func (d *dialedConn) Retrieved(data []byte) {
	_, e := d.hosted.Write(data)
	if e == nil {
		return
	}
	d.hosted.Close()
}

func (d *dialedConn) Close() {
	d.rwg.Wait()
	d.hosted.Close()
}
