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
	RetrieveType            = 3
	RetrieveRequestSize     = IDSize + 8 + 2
	RetrieveRequestOverhead = HeaderSize + RetrieveRequestSize
)

type RetrieveRequest struct {
	ID     ID
	RID    uint64
	Offset uint16
}

func (d *RetrieveRequest) Build(id ID, b *reader.Pusher) error {
	var err error
	// rType
	if !pusherPush(b, &err, NewRequestType(RetrieveType, 0).Byte()) {
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
	// offset
	if !pusherU16(b, &err, d.Offset) {
		return err
	}
	return nil
}

func (d *RetrieveRequest) Parse(r *reader.Fetcher) error {
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
	// offset
	d.Offset, err = readU16(r)
	if err != nil {
		return err
	}
	return nil
}

func (d *RetrieveRequest) Respond(rid uint64, total uint16, offset uint16, dd []byte) RetrieveRespond {
	return RetrieveRespond{
		ID:            d.ID,
		RID:           rid,
		Total:         total,
		Offset:        offset,
		Payload:       dd,
		PayloadLength: uint16(len(dd)),
	}
}
