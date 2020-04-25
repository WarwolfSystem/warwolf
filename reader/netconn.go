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

package reader

import (
	"errors"
	"net"
)

var (
	ErrNoBufferToReadTo    = errors.New("NetConn I/O: No buffer space to read to")
	ErrNoBufferToWriteFrom = errors.New("NetConn I/O: No buffer space to write from")
)

type NetConn struct {
	net.Conn
}

func NewNetConn(conn net.Conn) NetConn {
	return NetConn{
		Conn: conn,
	}
}

func (r NetConn) Read(b []byte) (int, error) {
	if len(b) == 0 {
		return 0, ErrNoBufferToReadTo
	}
	return r.Conn.Read(b)
}

func (r NetConn) Write(b []byte) (int, error) {
	if len(b) == 0 {
		return 0, ErrNoBufferToWriteFrom
	}
	return r.Conn.Write(b)
}
