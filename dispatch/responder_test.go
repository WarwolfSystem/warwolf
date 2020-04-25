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

package dispatch

import (
	"bytes"
	"io"
	"net"
	"strconv"
	"testing"
	"time"
	"warwolf/buffer"
	"warwolf/protocol"
	"warwolf/reader"
	"warwolf/relay"
	"warwolf/session"
)

func TestResponder(t *testing.T) {
	s := session.New(10, 10*time.Second)
	b := buffer.New(10, 6)
	l, e := net.ListenTCP("tcp", &net.TCPAddr{
		IP:   net.IPv4(127, 0, 0, 1),
		Port: 0,
		Zone: "",
	})
	if e != nil {
		t.Error("Error:", e)
		return
	}
	defer l.Close()
	type ccData struct {
		d []byte
		e error
	}
	ccDataChan := make(chan ccData, 1)
	go func() {
		buf := make([]byte, 9)
		for {
			cc, ce := l.Accept()
			if ce != nil {
				return
			}
			io.ReadFull(cc, buf[:8])
			cc.Write([]byte("Connect"))
			_, ce = io.ReadFull(cc, buf[8:])
			cc.Write([]byte("Connected"))
			ccDataChan <- ccData{
				d: buf,
				e: ce,
			}
			cc.Close()
			return
		}
	}()
	rsp := NewResponder(&s, &net.TCPAddr{
		IP:   net.IPv4(127, 0, 0, 1),
		Port: 0,
		Zone: "",
	}, relay.Config{
		DialTimeout:     1 * time.Second,
		RetrieveTimeout: 10 * time.Second,
	}, &b)
	_, port, _ := net.SplitHostPort(l.Addr().String())
	portn, _ := strconv.ParseUint(port, 10, 16)
	var builders []protocol.Builder
	builders = append(builders, (&protocol.DialRequest{
		ID:             protocol.ID{},
		ATyp:           protocol.TCPIPv4,
		Addr:           []byte{127, 0, 0, 1},
		Port:           uint16(portn),
		MaxRetrieveLen: 128,
		Request:        []byte("Test"),
		RequestLength:  4,
	}).Build)
	builders = append(builders, (&protocol.SendRequest{
		ID:            protocol.ID{},
		WID:           0,
		Payload:       []byte("Send"),
		PayloadLength: 4,
	}).Build)
	builders = append(builders, (&protocol.RetrieveRequest{
		ID:     protocol.ID{},
		RID:    0,
		Offset: 0,
	}).Build)
	builders = append(builders, (&protocol.SendRequest{
		ID:            protocol.ID{},
		WID:           1,
		Payload:       []byte("A"),
		PayloadLength: 1,
	}).Build)
	builders = append(builders, (&protocol.ResumeRequest{
		ID:  protocol.ID{},
		RID: 0,
	}).Build)
	builders = append(builders, (&protocol.RetrieveRequest{
		ID:     protocol.ID{},
		RID:    0,
		Offset: 0,
	}).Build)
	builders = append(builders, (&protocol.CloseRequest{
		ID: protocol.ID{},
	}).Build)
	p := reader.NewPusher(make([]byte, 1024))
	pp := reader.NewPusher(make([]byte, 1024))
	for i := range builders {
		p.Truncate(0)
		pp.Truncate(0)
		e = builders[i](protocol.ID{}, &p)
		if e != nil {
			t.Error("Build failed")
			return
		}
		rsp.Dispatch(
			func(format string, v ...interface{}) {},
			p.Data(),
			func(e PusherExecuter) error {
				return e(&pp)
			},
			Config{
				MaxRetrieveLen: 1024,
			},
		)
	}
	bb := <-ccDataChan
	if !bytes.Equal(bb.d, []byte{'T', 'e', 's', 't', 'S', 'e', 'n', 'd', 'A'}) {
		t.Error("Invalid data")
		return
	}
	// fmt.Println("pp.Data()", pp.Data())
}
