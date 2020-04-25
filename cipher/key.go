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
	"crypto/hmac"
	"crypto/sha256"
	"encoding/binary"
	"time"
)

const (
	KeySwitchInterval = 120 * time.Second
)

type Time [binary.MaxVarintLen64]byte

func timeByte() Time {
	t := Time{}
	s := time.Now().Truncate(KeySwitchInterval).Unix()
	binary.PutVarint(t[:], s)
	return t
}

type KeyGen struct {
	Key []byte
}

func (k KeyGen) Get() ([]byte, Time) {
	mac := hmac.New(sha256.New, k.Key)
	t := timeByte()
	mac.Write(t[:])
	return mac.Sum(nil)[:16], t
}
