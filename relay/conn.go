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

package relay

import (
	"net"
	"time"
)

type conn struct {
	net.Conn
	timeout time.Duration
}

func (t *conn) SetDeadline(tt time.Time) error {
	panic("Unsupported")
}

func (t *conn) SetReadDeadline(tt time.Time) error {
	panic("Unsupported")
}

func (t *conn) SetWriteDeadline(tt time.Time) error {
	panic("Unsupported")
}

func (t *conn) isTimeoutErr(e error) bool {
	if ee, ne := e.(net.Error); !ne || !ee.Timeout() {
		return false
	}
	return true
}

func (t *conn) Read(b []byte) (int, error) {
	l, e := t.Conn.Read(b)
	if e == nil {
		return l, nil
	}
	if !t.isTimeoutErr(e) {
		return l, e
	}
	t.Conn.SetReadDeadline(time.Now().Add(t.timeout))
	return t.Conn.Read(b)
}

func (t *conn) ReadTimeout(b []byte, tt time.Duration) (int, error) {
	t.Conn.SetReadDeadline(time.Now().Add(tt))
	return t.Conn.Read(b)
}

func (t *conn) Write(b []byte) (int, error) {
	l, e := t.Conn.Write(b)
	if e == nil {
		return l, nil
	}
	if !t.isTimeoutErr(e) {
		return l, e
	}
	t.Conn.SetWriteDeadline(time.Now().Add(t.timeout))
	return t.Conn.Write(b)
}
