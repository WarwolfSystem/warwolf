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

const ResumeType = 4

type ResumeRequest struct {
	ID  ID
	RID uint64
}

func (d *ResumeRequest) Build(id ID, b *reader.Pusher) error {
	var err error
	// rType
	if !pusherPush(b, &err, NewRequestType(ResumeType, 0).Byte()) {
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
	return nil
}

func (d *ResumeRequest) Parse(r *reader.Fetcher) error {
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
	return nil
}

func (d *ResumeRequest) RetrieveRequest() RetrieveRequest {
	return RetrieveRequest{
		ID:     d.ID,
		RID:    d.RID,
		Offset: 0,
	}
}

func (d *ResumeRequest) Respond(
	newRID uint64,
	total uint16,
	payload []byte,
) ResumeRespond {
	return ResumeRespond{
		ID:            d.ID,
		NewRID:        newRID,
		Total:         total,
		Payload:       payload,
		PayloadLength: uint16(len(payload)),
	}
}

func (d *ResumeRequest) RetrieveRespond(dd RetrieveRespond) ResumeRespond {
	return ResumeRespond{
		ID:            d.ID,
		NewRID:        dd.RID,
		Total:         dd.Total,
		Payload:       dd.Payload,
		PayloadLength: dd.PayloadLength,
	}
}
