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

type CloseRespond struct {
	ID ID
}

func (d *CloseRespond) Build(id ID, errcode byte, b *reader.Pusher) error {
	var err error
	if !pusherPush(b, &err, NewRequestType(CloseType, errcode).Byte()) {
		return err
	}
	d.ID = id
	if !pusherPush(b, &err, d.ID[:]...) {
		return err
	}
	return nil
}

func (d *CloseRespond) Parse(r *reader.Fetcher) error {
	_, rErr := io.ReadFull(r, d.ID[:])
	return rErr
}
