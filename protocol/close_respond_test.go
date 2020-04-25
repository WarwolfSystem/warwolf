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

func TestCloseRespond(t *testing.T) {
	c := CloseRespond{
		ID: ID{9, 8, 7, 6, 5, 4, 3, 2, 1, 0},
	}
	p := reader.NewPusher(make([]byte, 128))
	e := c.Build(c.ID, 0, &p)
	if e != nil {
		t.Error("Error:", e)
		return
	}
	c2 := CloseRespond{}
	e = c2.Parse(newReadSource(p.Data()[1:]))
	if e != nil {
		t.Error("Error:", e)
		return
	}
	if c2.ID != c.ID {
		t.Error("Invalid data")
		return
	}
}
