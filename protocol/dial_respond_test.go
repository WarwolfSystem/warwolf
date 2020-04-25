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
	"bytes"
	"testing"
	"warwolf/reader"
)

func TestDialRespond(t *testing.T) {
	id := ID{9, 8, 7, 6, 5, 4, 3, 2, 1, 0}
	r1 := DialRespond{
		ID:            id,
		RID:           10,
		Total:         15,
		Respond:       []byte("Test1Test2Test3"),
		RespondLength: 15,
	}
	p := reader.NewPusher(make([]byte, 128))
	e := r1.Build(id, 0, &p)
	if e != nil {
		t.Error("Error:", e)
		return
	}
	p.Write([]byte("Data"))
	r2 := DialRespond{}
	payload := make([]byte, 0, 128)
	e = r2.Parse(newReadSource(p.Data()[1:]), func(d *DialRespond, r *reader.Fetcher) error {
		b, _ := r.FetchMax(reader.MaxFetchSize)
		payload = append(payload, b...)
		return nil
	})
	if e != nil {
		t.Error("Failed:", e)
		return
	}
	if r2.ID != id || r2.RID != 10 || r2.Total != 15 ||
		r2.RespondLength != 15 ||
		!bytes.Equal(payload, []byte("Test1Test2Test3")) {
		t.Error("Invalid data")
		return
	}
}
