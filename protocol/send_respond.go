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
	SendErrorSuccess  = 0
	SendErrorTimeout  = 1
	SendErrorNotFound = 2
	SendErrorUnknown  = 3
)

type SendRespond struct {
	ID     ID
	NewWID uint64
	Sent   uint16
}

func (d SendRespond) Build(id ID, errcode byte, b *reader.Pusher) error {
	var err error
	// rType
	if !pusherPush(b, &err, NewRequestType(SendType, errcode).Byte()) {
		return err
	}
	// id
	d.ID = id
	if !pusherPush(b, &err, d.ID[:]...) {
		return err
	}
	// new_wid
	if !pusherU64(b, &err, d.NewWID) {
		return err
	}
	// sent
	if !pusherU16(b, &err, d.Sent) {
		return err
	}
	return nil
}

func (d *SendRespond) Parse(r *reader.Fetcher) error {
	// id
	_, err := io.ReadFull(r, d.ID[:])
	if err != nil {
		return err
	}
	// new_wid
	d.NewWID, err = readU64(r)
	if err != nil {
		return err
	}
	// sent
	d.Sent, err = readU16(r)
	if err != nil {
		return err
	}
	return nil
}
