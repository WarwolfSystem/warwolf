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

func TestDialRequest(t *testing.T) {
	req := DialRequest{
		ID:             ID{9, 8, 7, 6, 5, 4, 3, 2, 1, 0},
		ATyp:           TCPHost,
		Addr:           []byte("www.google.com"),
		Port:           443,
		MaxRetrieveLen: 10101,
		Request:        []byte("Client Hello"),
		RequestLength:  12,
	}
	p := reader.NewPusher(make([]byte, 128))
	err := req.Build(req.ID, &p)
	if err != nil {
		t.Error("Error:", err)
		return
	}
	p.Write([]byte("Data"))
	req2 := DialRequest{}
	payload := make([]byte, 0, 128)
	e := req2.Parse(TCPHost, newReadSource(p.Data()[1:]), func(d *DialRequest, r []byte) error {
		payload = append(payload, r...)
		return nil
	})
	if e != nil {
		t.Error("Error:", e)
		return
	}
	expectedID := ID{9, 8, 7, 6, 5, 4, 3, 2, 1, 0}
	if req2.ATyp != TCPHost ||
		!bytes.Equal(req2.Addr, []byte("www.google.com")) ||
		req2.MaxRetrieveLen != 10101 ||
		req2.RequestLength != 12 ||
		!bytes.Equal(payload, []byte("Client Hello")) ||
		req2.ID != expectedID ||
		req2.Port != 443 {
		t.Error("Invalid data")
		return
	}
}
