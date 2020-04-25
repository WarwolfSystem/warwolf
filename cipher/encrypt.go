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

package cipher

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"io"
	"math"
	"warwolf/reader"
)

const (
	NonceSize    = 12
	BlockSize    = aes.BlockSize
	HeaderSize   = NonceSize + BlockSize + 2
	OverheadSize = HeaderSize + BlockSize
)

var (
	ErrInvalidSize  = errors.New("Cipher: Invalid size")
	ErrInvalidNonce = errors.New("Cipher: Invalid nonce")
)

type NonceVerifier func(nonce []byte, t Time) bool

func Nonce() ([NonceSize]byte, error) {
	n := [NonceSize]byte{}
	_, e := io.ReadFull(rand.Reader, n[:])
	return n, e
}

func AEAD(k []byte) (cipher.AEAD, error) {
	b, e := aes.NewCipher(k)
	if e != nil {
		return nil, e
	}
	return cipher.NewGCM(b)
}

func Encrypt(gcm cipher.AEAD, nonce [NonceSize]byte, b []byte) []byte {
	if len(nonce) != gcm.NonceSize() {
		panic("Invalid nonce")
	}
	if len(b) < OverheadSize {
		panic("Given buffer is smaller than what's required")
	}
	if len(b) > math.MaxUint16 {
		panic("Invalid data length")
	}
	nlen := copy(b, nonce[:])
	blen := len(b) - OverheadSize
	b[nlen] = byte(blen >> 8)
	b[nlen+1] = byte(blen)
	hlen := len(gcm.Seal(b[nlen:nlen], nonce[:], b[nlen:nlen+2], nil)) + nlen
	if hlen != HeaderSize {
		panic("Generated an invalid header")
	}
	nonce[0]++
	elen := len(gcm.Seal(b[hlen:hlen], nonce[:], b[hlen:hlen+blen], nil))
	if elen != BlockSize+blen {
		panic("Generated an invalid body")
	}
	if elen+hlen != len(b) {
		panic("Invalid data length")
	}
	return b
}

func Decrypt(t Time, gcm func() (cipher.AEAD, error), nv NonceVerifier, f *reader.Fetcher, eoferr error, c func(b []byte) error) error {
	nonce := [NonceSize]byte{}
	for {
		d, e := f.Fetch(HeaderSize)
		if e != nil {
			if e == eoferr {
				return nil
			}
			return e
		}
		copy(nonce[:], d[:NonceSize])
		if !nv(nonce[:], t) {
			return ErrInvalidNonce
		}
		nlen := len(nonce)
		gg, e := gcm()
		if e != nil {
			return e
		}
		o, e := gg.Open(d[nlen:nlen], nonce[:], d[nlen:HeaderSize], nil)
		if e != nil {
			return e
		}
		if len(o) != 2 {
			return ErrInvalidSize
		}
		size := uint16(o[0])<<8 | uint16(o[1])
		d, e = f.Fetch(BlockSize + int(size))
		if e != nil {
			return e
		}
		nonce[0]++
		o, e = gg.Open(d[:0], nonce[:], d, nil)
		if e != nil {
			return e
		}
		e = c(o)
		if e != nil {
			return e
		}
	}
}
