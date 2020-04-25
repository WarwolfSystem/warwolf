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

package reader

import (
	"errors"
)

var (
	ErrOverCapacity = errors.New("Pusher: Cannot write: Over capacity")
)

type Push interface {
	Push(i int, d []byte) error
	Remain() int
}

type Pusher struct {
	b []byte
	i int
}

func NewPusher(b []byte) Pusher {
	return Pusher{
		b: b,
		i: 0,
	}
}

func (p *Pusher) Push(b ...byte) error {
	_, err := p.Write(b)
	return err
}

func (p *Pusher) Write(b []byte) (int, error) {
	bLen := len(b)
	if bLen > (len(p.b) - p.i) {
		return 0, ErrOverCapacity
	}
	copied := copy(p.b[p.i:], b)
	p.i += copied
	return copied, nil
}

func (p *Pusher) Data() []byte {
	return p.b[:p.i]
}

func (p *Pusher) Truncate(n int) {
	p.i = n
}

func (p *Pusher) Size() int {
	return p.i
}
