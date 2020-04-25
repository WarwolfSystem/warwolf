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
	"math"
	"warwolf/reader"
)

const (
	DialType             = 0
	DialSafeOverheadSize = IDSize + 1 + 255 + 2 + 2 + 2
)

type DialRequest struct {
	ID             ID
	ATyp           AddressType
	Addr           []byte
	Port           uint16
	MaxRetrieveLen uint16
	Request        []byte
	RequestLength  uint16
}

func (d *DialRequest) Build(id ID, b *reader.Pusher) error {
	var err error
	// rType
	if !pusherPush(b, &err, NewRequestType(DialType, byte(d.ATyp)).Byte()) {
		return err
	}
	// id
	d.ID = id
	if !pusherPush(b, &err, d.ID[:]...) {
		return err
	}
	// addr
	switch d.ATyp {
	case TCPIPv4:
		fallthrough
	case UDPIPv4:
		if len(d.Addr) != 4 {
			return ErrInvalidAddress
		}
		if !pusherPush(b, &err, d.Addr...) {
			return err
		}

	case TCPIPv6:
		fallthrough
	case UDPIPv6:
		if len(d.Addr) != 16 {
			return ErrInvalidAddress
		}
		if !pusherPush(b, &err, d.Addr...) {
			return err
		}

	case TCPHost:
		fallthrough
	case UDPHost:
		alen := len(d.Addr)
		if alen > math.MaxUint8 {
			return ErrInvalidAddress
		}
		if !pusherPush(b, &err, byte(alen)) {
			return err
		}
		if !pusherPush(b, &err, d.Addr...) {
			return err
		}

	default:
		panic("Unknown address type")
	}
	// port
	if !pusherU16(b, &err, d.Port) {
		return err
	}
	// max_retrieve_len
	if !pusherU16(b, &err, d.MaxRetrieveLen) {
		return err
	}
	// request
	if len(d.Request) > math.MaxUint16 {
		panic("Invalid request length")
	}
	if len(d.Request) != int(d.RequestLength) {
		panic("Invalid request length")
	}
	if !pusherU16(b, &err, d.RequestLength) {
		return err
	}
	if !pusherPush(b, &err, d.Request...) {
		return err
	}
	return nil
}

func (d *DialRequest) Parse(atyp AddressType, r *reader.Fetcher, rr func(d *DialRequest, r []byte) error) error {
	var err error
	var bb []byte
	// id
	_, err = io.ReadFull(r, d.ID[:])
	if err != nil {
		return err
	}
	d.ATyp = atyp
	// addr
	switch d.ATyp {
	case UDPIPv4:
		fallthrough
	case TCPIPv4:
		d.Addr, err = r.Fetch(4)
		if err != nil {
			return err
		}

	case UDPIPv6:
		fallthrough
	case TCPIPv6:
		d.Addr, err = r.Fetch(16)
		if err != nil {
			return err
		}

	case UDPHost:
		fallthrough
	case TCPHost:
		bb, err = r.Fetch(1)
		if err != nil {
			return err
		}
		d.Addr, err = r.Fetch(int(bb[0]))
		if err != nil {
			return err
		}

	default:
		return ErrInvalidAddress
	}
	// port
	d.Port, err = readU16(r)
	if err != nil {
		return err
	}
	// max_retrieve_len
	d.MaxRetrieveLen, err = readU16(r)
	if err != nil {
		return err
	}
	// request
	d.RequestLength, err = readU16(r)
	if err != nil {
		return err
	}
	b, berr := r.Fetch(int(d.RequestLength))
	if berr != nil {
		return berr
	}
	return rr(d, b)
}

func (d *DialRequest) Respond(
	rid uint64,
	total uint16,
	respond []byte,
) DialRespond {
	return DialRespond{
		ID:            d.ID,
		RID:           rid,
		Total:         total,
		Respond:       respond,
		RespondLength: uint16(len(respond)),
	}
}

func (d *DialRequest) RetrieveRequest() RetrieveRequest {
	return RetrieveRequest{
		ID:     d.ID,
		RID:    0,
		Offset: 0,
	}
}

func (d *DialRequest) RetrieveRespond(r RetrieveRespond) DialRespond {
	return d.Respond(r.RID, r.Total, r.Payload)
}
