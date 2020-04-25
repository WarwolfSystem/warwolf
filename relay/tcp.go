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
	"io"
	"net"
	"time"
	"warwolf/reader"
)

type TCP struct {
	raddr   net.Addr
	laddr   net.Addr
	connReq chan *conn
	conn    *conn
	retReq  chan retrieverReq
}

func NewTCP(
	raddr net.Addr,
	laddr net.Addr,
) (*TCP, Error) {
	return &TCP{
		raddr:   raddr,
		laddr:   laddr,
		connReq: make(chan *conn, 1),
		conn:    nil,
		retReq:  make(chan retrieverReq, 1),
	}, Error{}
}

func (u *TCP) getConn() (*conn, error) {
	if u.conn != nil {
		return u.conn, nil
	}
	var ok bool
	u.conn, ok = <-u.connReq
	if !ok {
		return nil, io.EOF
	}
	return u.conn, nil
}

func (u *TCP) Serve(rbuf []byte, cc Config, connected Connector) Error {
	defer close(u.connReq)
	d := buildDialer(u.laddr, cc.DialTimeout)
	ccc, err := d.Dial("tcp", u.raddr.String())
	if err != nil {
		connected(nil, err)
		return newError(err)
	}
	defer func() {
		if ccc == nil {
			return
		}
		ccc.Close()
	}()
	connected(ccc, nil)
	c := ccc.(*net.TCPConn)
	c.SetDeadline(time.Now().Add(cc.RetrieveTimeout))
	cconn := &conn{
		Conn:    reader.NewNetConn(ccc),
		timeout: cc.RetrieveTimeout,
	}
	ccc = nil
	defer cconn.Close()
	u.connReq <- cconn
	for {
		ret, retok := <-u.retReq
		if !retok {
			return Error{}
		}
		var rlen int
		var rerr error
		if ret.t > 0 {
			rlen, rerr = cconn.ReadTimeout(rbuf, ret.t)
		} else {
			rlen, rerr = cconn.Read(rbuf)
		}
		if rerr != nil {
			ret.r(nil, newError(rerr))
			continue
		}
		ret.r(rbuf[:rlen], Error{})
	}
}

func (u *TCP) Retrieve(r Retriever, t time.Duration) {
	_, err := u.getConn()
	if err != nil {
		r(nil, newError(err))
		return
	}
	u.retReq <- retrieverReq{
		r: r,
		t: t,
	}
}

func (u *TCP) Send(b []byte) (int, Error) {
	conn, err := u.getConn()
	if err != nil {
		return 0, newError(err)
	}
	l, err := conn.Write(b)
	return l, newError(err)
}

func (u *TCP) Close() {
	conn, err := u.getConn()
	if err != nil {
		return
	}
	conn.Close()
	close(u.retReq)
}
