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

import (
	"errors"
	"net"
	"strconv"
	"warwolf/protocol"
)

var (
	ErrInvalidAddress = errors.New("Session: Invalid address")
)

type addrr struct {
	netw string
	addr string
}

func (a addrr) Network() string {
	return a.netw
}

func (a addrr) String() string {
	return a.addr
}

func buildAddr(
	t protocol.AddressType,
	a []byte,
	p uint16,
) (net.Addr, error) {
	switch t {
	case protocol.TCPIPv4:
		return &net.TCPAddr{
			IP:   net.IPv4(a[0], a[1], a[2], a[3]),
			Port: int(p),
		}, nil
	case protocol.UDPIPv4:
		return &net.UDPAddr{
			IP:   net.IPv4(a[0], a[1], a[2], a[3]),
			Port: int(p),
		}, nil
	case protocol.TCPIPv6:
		ip := make([]byte, 16)
		copy(ip, a)
		return &net.UDPAddr{
			IP:   ip,
			Port: int(p),
		}, nil
	case protocol.UDPIPv6:
		ip := make([]byte, 16)
		copy(ip, a)
		return &net.TCPAddr{
			IP:   ip,
			Port: int(p),
		}, nil
	case protocol.TCPHost:
		return addrr{
			netw: "tcp",
			addr: net.JoinHostPort(string(a), strconv.FormatUint(uint64(p), 10)),
		}, nil
	case protocol.UDPHost:
		return addrr{
			netw: "udp",
			addr: net.JoinHostPort(string(a), strconv.FormatUint(uint64(p), 10)),
		}, nil
	default:
		return nil, ErrInvalidAddress
	}
}
