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
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"io"
	"testing"
	"warwolf/reader"
)

func TestEncrypt(t *testing.T) {
	key := KeyGen{
		Key: []byte("TestKey"),
	}
	k, tt := key.Get()
	block, err := aes.NewCipher(k)
	if err != nil {
		panic(err.Error())
	}
	nonce := [12]byte{}
	if _, err := io.ReadFull(rand.Reader, nonce[:]); err != nil {
		panic(err.Error())
	}
	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		panic(err.Error())
	}
	var r []byte
	ee := Encrypt(aesgcm, nonce, []byte("                              ABC                "))
	eereader := reader.NewFetcher(reader.ByteFetch(ee, io.EOF))
	err = Decrypt(tt, func() (cipher.AEAD, error) {
		return aesgcm, nil
	}, func(b []byte, t Time) bool {
		return true
	}, &eereader, io.EOF, func(b []byte) error {
		r = b
		return nil
	})
	if err != nil {
		t.Error(err)
		return
	}
	if !bytes.Equal(r, []byte("ABC")) {
		t.Error("Invalid data")
	}
}
