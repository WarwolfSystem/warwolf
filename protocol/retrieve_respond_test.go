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

func TestRetrieveRespond(t *testing.T) {
	id := ID{9, 8, 7, 6, 5, 4, 3, 2, 1, 0}
	r := RetrieveRespond{
		ID:            id,
		RID:           991029291932123,
		Total:         30,
		Offset:        5,
		Payload:       []byte("Test1Test2Test3Test4Test5"),
		PayloadLength: 25,
	}
	p := reader.NewPusher(make([]byte, 128))
	e := r.Build(r.ID, 0, &p)
	if e != nil {
		t.Error("Error:", e)
		return
	}
	p.Write([]byte("Data"))
	r1 := RetrieveRespond{}
	payload := make([]byte, 0, 128)
	e = r1.Parse(
		newReadSource(p.Data()[1:]),
		func(d *RetrieveRespond, r *reader.Fetcher) error {
			b, _ := r.FetchMax(reader.MaxFetchSize)
			payload = append(payload, b...)
			return nil
		})
	if e != nil {
		t.Error("Error:", e)
		return
	}
	if r1.ID != id ||
		r1.RID != 991029291932123 ||
		r1.Total != 30 ||
		r1.Offset != 5 ||
		r1.PayloadLength != 25 ||
		!bytes.Equal(payload, []byte("Test1Test2Test3Test4Test5")) {
		t.Error("Invalid data")
		return
	}
}
