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

package protocol

import (
	"io"
	"warwolf/reader"
)

const (
	SendType           = 5
	SendHeaderSize     = IDSize + 8 + 2
	SendHeaderOverhead = HeaderSize + SendHeaderSize
)

type SendRequest struct {
	ID            ID
	WID           uint64
	Payload       []byte
	PayloadLength uint16
}

func (d *SendRequest) BuildHeader(id ID, b *reader.Pusher) error {
	var err error
	// rType
	if !pusherPush(b, &err, NewRequestType(SendType, 0).Byte()) {
		return err
	}
	// id
	d.ID = id
	if !pusherPush(b, &err, d.ID[:]...) {
		return err
	}
	// wid
	if !pusherU64(b, &err, d.WID) {
		return err
	}
	// payload
	if !pusherU16(b, &err, d.PayloadLength) {
		return err
	}
	return nil
}

func (d *SendRequest) Build(id ID, b *reader.Pusher) error {
	err := d.BuildHeader(id, b)
	if err != nil {
		return err
	}
	// payload
	if len(d.Payload) != int(d.PayloadLength) {
		panic("Invalid payload length")
	}
	if !pusherPush(b, &err, d.Payload...) {
		return err
	}
	return nil
}

func (d *SendRequest) Parse(r *reader.Fetcher, rr func(d *SendRequest, r []byte) error) error {
	// id
	_, err := io.ReadFull(r, d.ID[:])
	if err != nil {
		return err
	}
	// wid
	d.WID, err = readU64(r)
	if err != nil {
		return err
	}
	// payload
	d.PayloadLength, err = readU16(r)
	if err != nil {
		return err
	}
	b, berr := r.Fetch(int(d.PayloadLength))
	if berr != nil {
		return berr
	}
	return rr(d, b)
}

func (d *SendRequest) Respond(newWID uint64, wlen uint16) SendRespond {
	return SendRespond{
		ID:     d.ID,
		NewWID: newWID,
		Sent:   wlen,
	}
}
