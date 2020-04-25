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

const CloseType = 1

type CloseRequest struct {
	ID ID
}

func (d *CloseRequest) Build(id ID, b *reader.Pusher) error {
	var err error
	if !pusherPush(b, &err, NewRequestType(CloseType, 0).Byte()) {
		return err
	}
	d.ID = id
	if !pusherPush(b, &err, d.ID[:]...) {
		return err
	}
	return nil
}

func (d *CloseRequest) Parse(r *reader.Fetcher) error {
	_, rErr := io.ReadFull(r, d.ID[:])
	return rErr
}

func (d *CloseRequest) Respond() CloseRespond {
	return CloseRespond{
		ID: d.ID,
	}
}
