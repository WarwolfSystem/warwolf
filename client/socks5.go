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
	"errors"
	"io"
	"net"
	"warwolf/buffer"
	"warwolf/log"
	"warwolf/protocol"
)

var (
	ErrNoSocks5AuthMethod       = errors.New("Socks5: No supported auth method")
	ErrNoSocks5AuthFailed       = errors.New("Socks5: Auth failed")
	ErrNoSocks5BadAddressType   = errors.New("Socks5: Bad address type")
	ErrUnsupportedSocks5Request = errors.New("Socks5: Unsupported request")
)

const (
	socks5MethodNoAuth           = 0x00
	socks5MethodGSSAPI           = 0x01
	socks5MethodUsernamePassword = 0x02
	socks5MethodIANAASSIGNED     = 0x03
	socks5MethodRESERVED         = 0x80
	socks5MethodNoAcceptable     = 0xFF
	socks5CmdConnect             = 0x01
	socks5CmdBind                = 0x02
	socks5CmdUDP                 = 0x03
	socks5ATypeIPv4              = 0x01
	socks5ATypeDomain            = 0x03
	socks5ATypeIPv6              = 0x04
)

type socks5Auth func(username, password string) bool
type socks5Exec func(lg log.Log, d *dial, b *buffer.Buffer, laddr *net.TCPAddr, bb []byte, atype byte, addr []byte, port uint16, r net.Conn) error

func socks5NoAuth(username, password string) bool {
	return true
}

func socks5BuildAddrFromIP(padsize int, a net.IP, port uint16) ([]byte, bool) {
	ipv4 := a.To4()
	if ipv4 != nil {
		addr := make([]byte, 4+2+padsize)
		l := copy(addr[padsize:], ipv4[:4])
		addr[padsize+l] = byte(port >> 8)
		addr[padsize+l+1] = byte(port)
		return addr, true
	}
	addr := make([]byte, 16+2+padsize)
	l := copy(addr[padsize:], a[:16])
	addr[padsize+l] = byte(port >> 8)
	addr[padsize+l+1] = byte(port)
	return addr, false
}

func socks5AtypeToProtocolTCPAtype(atype byte) protocol.AddressType {
	var atyp protocol.AddressType
	switch atype {
	case socks5ATypeIPv4:
		atyp = protocol.TCPIPv4
	case socks5ATypeIPv6:
		atyp = protocol.TCPIPv6
	case socks5ATypeDomain:
		fallthrough
	default:
		atyp = protocol.TCPHost
	}
	return atyp
}

func socks5AtypeToProtocolUDPAtype(atype byte) protocol.AddressType {
	var atyp protocol.AddressType
	switch atype {
	case socks5ATypeIPv4:
		atyp = protocol.UDPIPv4
	case socks5ATypeIPv6:
		atyp = protocol.UDPIPv6
	case socks5ATypeDomain:
		fallthrough
	default:
		atyp = protocol.UDPHost
	}
	return atyp
}

func socks5Addr(atype byte, r io.Reader) ([]byte, uint16, error) {
	var addr []byte
	var err error
	switch atype {
	case socks5ATypeIPv4:
		addr = make([]byte, 4+2)
		_, err = io.ReadFull(r, addr)
	case socks5ATypeDomain:
		b := [1]byte{}
		_, err = io.ReadFull(r, b[:1])
		if err != nil {
			return nil, 0, err
		}
		addr = make([]byte, b[0]+2)
		_, err = io.ReadFull(r, addr)
	case socks5ATypeIPv6:
		addr = make([]byte, 16+2)
		_, err = io.ReadFull(r, addr)
	default:
		return nil, 0, ErrNoSocks5BadAddressType
	}
	if err != nil {
		return nil, 0, err
	}
	alen := len(addr)
	return addr, uint16(addr[alen-2])<<8 | uint16(addr[alen-1]), err
}

func socks5(lg log.Log, d *dial, bb *buffer.Buffer, b []byte, laddr *net.TCPAddr, r net.Conn, auth socks5Auth, tcpExec socks5Exec, udpExec socks5Exec) error {
	_, err := io.ReadFull(r, b[:2])
	if err != nil {
		return err
	}
	l, err := io.ReadFull(r, b[:b[1]])
	if err != nil {
		return err
	}
	authSelected := false
	isNoAuth := false
	for i := 0; i < l; i++ {
		switch b[i] {
		case socks5MethodNoAuth:
			if !authSelected && auth == nil {
				authSelected = true
				isNoAuth = true
			}
		case socks5MethodUsernamePassword:
			if !authSelected {
				authSelected = true
				isNoAuth = false
			}
		default:
		}
	}
	if !authSelected {
		return ErrNoSocks5AuthMethod
	}
	if isNoAuth {
		_, err = r.Write([]byte{0x05, 00})
	} else {
		_, err = r.Write([]byte{0x05, 02})
		if err != nil {
			return err
		}
		_, err = io.ReadFull(r, b[:2])
		if err != nil {
			return err
		}
		username := make([]byte, b[1])
		_, err = io.ReadFull(r, username)
		if err != nil {
			return err
		}
		_, err = io.ReadFull(r, b[:1])
		if err != nil {
			return err
		}
		password := make([]byte, b[0])
		_, err = io.ReadFull(r, password)
		if err != nil {
			return err
		}
		if auth(string(username), string(password)) {
			return ErrNoSocks5AuthFailed
		}
		_, err = r.Write([]byte{0x05, 00})
	}
	if err != nil {
		return err
	}
	_, err = io.ReadFull(r, b[:4])
	cmd := b[1]
	atype := b[3]
	var addr []byte
	var port uint16
	addr, port, err = socks5Addr(atype, r)
	if err != nil {
		return err
	}
	switch cmd {
	case socks5CmdConnect:
		return tcpExec(lg, d, bb, laddr, b, atype, addr[:len(addr)-2], port, r)
	case socks5CmdUDP:
		return udpExec(lg, d, bb, laddr, b, atype, addr[:len(addr)-2], port, r)
	default:
		return ErrUnsupportedSocks5Request
	}
}
