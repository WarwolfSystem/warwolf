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
	"testing"
	"warwolf/reader"
)

func newReadSource(b []byte) *reader.Fetcher {
	f := reader.NewFetcher(func(min int, o []byte, ou int) ([]byte, error) {
		return b, nil
	})
	return &f
}

func TestWriteU16(t *testing.T) {
	b := [2]byte{}
	p := reader.NewPusher(b[:])
	writeU16(12345, &p)
	if d, _ := readU16(newReadSource(b[:])); d != 12345 {
		t.Error("Conversion failed")
	}
}

func TestWriteU64(t *testing.T) {
	b := [8]byte{}
	p := reader.NewPusher(b[:])
	writeU64(123456789, &p)
	if d, _ := readU64(newReadSource(b[:])); d != 123456789 {
		t.Error("Conversion failed")
	}
}
