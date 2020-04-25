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

type RetrieveRespond struct {
	ID            ID
	RID           uint64
	Total         uint16
	Offset        uint16
	Payload       []byte
	PayloadLength uint16
}

func (d RetrieveRespond) Build(id ID, errcode byte, b *reader.Pusher) error {
	var err error
	// rType
	if !pusherPush(b, &err, NewRequestType(RetrieveType, errcode).Byte()) {
		return err
	}
	// id
	d.ID = id
	if !pusherPush(b, &err, d.ID[:]...) {
		return err
	}
	// rid
	if !pusherU64(b, &err, d.RID) {
		return err
	}
	// total_size
	if !pusherU16(b, &err, d.Total) {
		return err
	}
	// offset
	if !pusherU16(b, &err, d.Offset) {
		return err
	}
	// payload
	if uint16(len(d.Payload)) != d.PayloadLength {
		panic("Invalid payload length")
	}
	if !pusherU16(b, &err, d.PayloadLength) {
		return err
	}
	if !pusherPush(b, &err, d.Payload...) {
		return err
	}
	return nil
}

func (d *RetrieveRespond) Parse(r *reader.Fetcher, rr func(d *RetrieveRespond, r *reader.Fetcher) error) error {
	// id
	_, err := io.ReadFull(r, d.ID[:])
	if err != nil {
		return err
	}
	// rid
	d.RID, err = readU64(r)
	if err != nil {
		return err
	}
	// total_size
	d.Total, err = readU16(r)
	if err != nil {
		return err
	}
	// offset
	d.Offset, err = readU16(r)
	if err != nil {
		return err
	}
	// payload
	d.PayloadLength, err = readU16(r)
	if err != nil {
		return err
	}
	rrr := reader.NewFetcher(reader.SizeLimitedFetch(int(d.PayloadLength), r))
	defer reader.FetchAll(int(d.PayloadLength), &rrr, func(b []byte) {})
	return rr(d, &rrr)
}
