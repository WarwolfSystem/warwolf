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

package session

import "sync"

type lock struct {
	l        *sync.Mutex
	unlocked bool
}

func lll(l *sync.Mutex) lock {
	return lock{
		l:        l,
		unlocked: true,
	}
}

func (l *lock) lock() {
	l.l.Lock()
	l.unlocked = false
}

func (l *lock) unlock() {
	if l.unlocked {
		return
	}
	l.unlocked = true
	l.l.Unlock()
}
