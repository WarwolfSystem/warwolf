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
	"net"
	"warwolf/protocol"
	"warwolf/relay"
)

type RetrieverError struct {
	E        error
	TryAgain bool
}

func (r RetrieverError) IsError() bool {
	return r.E != nil
}

func (r RetrieverError) Error() string {
	return r.E.Error()
}

func newRetrieverError(e error, tryAgain bool) RetrieverError {
	return RetrieverError{
		E:        e,
		TryAgain: tryAgain,
	}
}

func isErrorRecoverable(e error) bool {
	if e == nil {
		return true
	}
	ee, eee := e.(RetrieverError)
	if eee && (!ee.IsError() || ee.TryAgain) {
		return true
	}
	return false
}

func intIsIn(n int, in ...int) bool {
	for i := range in {
		if in[i] != n {
			continue
		}

		return true
	}

	return false
}

func buildRelay(r *protocol.DialRequest, laddr net.Addr, raddr net.Addr) (relay.Relay, relay.Error) {
	switch r.ATyp {
	case protocol.TCPIPv4:
		fallthrough
	case protocol.TCPIPv6:
		fallthrough
	case protocol.TCPHost:
		return relay.NewTCP(raddr, laddr)

	case protocol.UDPIPv4:
		fallthrough
	case protocol.UDPIPv6:
		fallthrough
	case protocol.UDPHost:
		return relay.NewUDP(raddr, laddr)

	default:
		return nil, relay.Error{E: ErrInvalidProtocol, IsTimeout: false}
	}
}
