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

package buffer

type bufferChan chan []byte

type Buffer struct {
	size    int
	buffers bufferChan
}

func New(size, capacity int) Buffer {
	bf := make(bufferChan, capacity)
	for i := 0; i < capacity; i++ {
		bf <- make([]byte, size)
	}
	return Buffer{
		size:    size,
		buffers: bf,
	}
}

func (b *Buffer) Request() []byte {
	select {
	case b := <-b.buffers:
		return b
	default:
		return make([]byte, b.size)
	}
}

func (b *Buffer) Return(bb []byte) {
	select {
	case b.buffers <- bb:
	default:
	}
}
