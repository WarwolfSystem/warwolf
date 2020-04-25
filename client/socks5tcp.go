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

package client

import (
	"net"
	"time"
	"warwolf/buffer"
	"warwolf/log"
	"warwolf/reader"
)

func socks5TCP(lg log.Log, d *dial, b *buffer.Buffer, laddr *net.TCPAddr, bb []byte, atype byte, addr []byte, port uint16, r net.Conn) error {
	resp, isip4 := socks5BuildAddrFromIP(4, net.IPv4(0, 0, 0, 0), 0)
	resp[0] = 5
	if isip4 {
		resp[3] = socks5ATypeIPv4
	} else {
		resp[3] = socks5ATypeIPv6
	}
	_, e := r.Write(resp)
	if e != nil {
		return e
	}
	atyp := socks5AtypeToProtocolTCPAtype(atype)
	push := b.Request()
	pushReturned := false
	defer func() {
		if pushReturned {
			return
		}
		b.Return(push)
	}()
	p := reader.NewPusher(push[:])
	r.SetDeadline(time.Now().Add(reqDataReadDelay))
	l, _ := r.Read(bb[:reqDataSafeSize])
	r.SetDeadline(time.Time{})
	return d.dial(atyp, addr, port, bb, l, &p, r, func() {
		pushReturned = true
		b.Return(push)
		push = nil
		p = reader.Pusher{}
	})
}
