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
	"errors"
	"net"
	"time"
)

var (
	ErrCancelled = errors.New("Relay: Retrieve has been cancelled")
)

type Error struct {
	E         error
	IsTimeout bool
}

func (e Error) IsError() bool {
	return e.E != nil
}

func (e Error) Error() string {
	return e.E.Error()
}

func newError(e error) Error {
	ee, eee := e.(net.Error)

	return Error{
		E:         e,
		IsTimeout: eee && ee.Timeout(),
	}
}

type Connector func(c net.Conn, err error)
type Retriever func(r []byte, err Error)

type Relay interface {
	Serve(rbuf []byte, c Config, connected Connector) Error
	Retrieve(r Retriever, t time.Duration)
	Send(b []byte) (int, Error)
	Close()
}

type retrieverReq struct {
	r Retriever
	t time.Duration
}

func tempErr(e error) bool {
	if nerr, isnerr := e.(net.Error); isnerr && nerr.Temporary() {
		return true
	}

	return false
}

func buildDialer(laddr net.Addr, timeout time.Duration) net.Dialer {
	return net.Dialer{
		Timeout:   timeout,
		Deadline:  time.Now().Add(timeout),
		LocalAddr: laddr,
		KeepAlive: 30 * time.Second,
		Resolver:  nil,
		Control:   nil,
	}
}
