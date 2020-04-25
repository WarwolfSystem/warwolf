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
	"encoding/hex"
	"errors"
	"warwolf/reader"
)

var (
	ErrInvalidAddress           = errors.New("Invalid address")
	ErrInvalidDataPayloadLength = errors.New("Invalid payload data length")
)

const (
	ResourceErrorSuccess     = 0
	ResourceErrorNotFound    = 1
	ResourceErrorNotReady    = 2
	ResourceErrorExpired     = 3
	ResourceErrorBroken      = 4
	ResourceErrorClosed      = 5
	ResourceErrorSendFailure = 6
	ResourceErrorUnknown     = 7
)

const (
	HeaderSize         = 1
	IDSize             = 32
	GreatestHeaderSize = DialSafeOverheadSize + DialSafeOverheadSize // Manually updated
)

type ID [IDSize]byte

func (i ID) String() string {
	return hex.EncodeToString(i[:])
}

type Builder func(id ID, p *reader.Pusher) error

type AddressType byte

const (
	TCPIPv4 AddressType = 0
	TCPIPv6 AddressType = 1
	TCPHost AddressType = 2
	UDPIPv4 AddressType = 3
	UDPIPv6 AddressType = 4
	UDPHost AddressType = 5
)

type RequestType byte

func (r RequestType) Byte() byte {
	return byte(r)
}

func NewRequestType(t byte, data byte) RequestType {
	if t > 15 {
		panic("Invalid type")
	}
	if data > 15 {
		panic("Invalid data")
	}
	return RequestType(t<<4 | data)
}

func ParseRequestType(r RequestType) (t byte, data byte) {
	t = byte(r) >> 4
	data = byte(r & 15)
	return
}

func writeU16(n uint16, b *reader.Pusher) error {
	return b.Push(byte(n>>8), byte(n))
}

func readU16(b *reader.Fetcher) (uint16, error) {
	bb, err := b.Fetch(2)
	if err != nil {
		return 0, err
	}
	return uint16(bb[0])<<8 | uint16(bb[1]), nil
}

func writeU64(n uint64, b *reader.Pusher) error {
	return b.Push(
		byte(n>>56), byte(n>>48), byte(n>>40), byte(n>>32),
		byte(n>>24), byte(n>>16), byte(n>>8), byte(n),
	)
}

func readU64(b *reader.Fetcher) (uint64, error) {
	bb, err := b.Fetch(8)
	if err != nil {
		return 0, err
	}
	return uint64(bb[0])<<56 |
		uint64(bb[1])<<48 |
		uint64(bb[2])<<40 |
		uint64(bb[3])<<32 |
		uint64(bb[4])<<24 |
		uint64(bb[5])<<16 |
		uint64(bb[6])<<8 |
		uint64(bb[7]), nil
}

func pusherU16(p *reader.Pusher, e *error, u uint16) bool {
	err := writeU16(u, p)
	if err == nil {
		return true
	}
	*e = err
	return false
}

func pusherU64(p *reader.Pusher, e *error, u uint64) bool {
	err := writeU64(u, p)
	if err == nil {
		return true
	}
	*e = err
	return false
}

func pusherPush(p *reader.Pusher, e *error, d ...byte) bool {
	err := p.Push(d...)
	if err == nil {
		return true
	}
	*e = err
	return false
}
