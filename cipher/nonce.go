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
	"sync"
)

type Nonces struct {
	size    int
	t       Time
	records map[string]struct{}
	l       *sync.Mutex
}

func (n *Nonces) clear() {
	n.records = make(map[string]struct{}, n.size)
}

func (n *Nonces) Verify(nonce []byte, t Time) bool {
	n.l.Lock()
	defer n.l.Unlock()
	if n.t != t {
		n.clear()
	}
	nn := string(nonce)
	_, ex := n.records[nn]
	if ex {
		return false
	}
	n.records[nn] = struct{}{}
	return true
}

func NewNonces(size int, l *sync.Mutex) Nonces {
	return Nonces{
		size:    size,
		t:       Time{},
		records: make(map[string]struct{}, size),
		l:       l,
	}
}
