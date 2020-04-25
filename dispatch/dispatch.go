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
	"errors"
	"sync"
	"warwolf/log"
	"warwolf/protocol"
	"warwolf/reader"
	"warwolf/session"
)

var (
	ErrDispatchCompleted  = errors.New("Dispatch completed")
	ErrUnknownRequestType = errors.New("Unknown request type")
)

type Config struct {
	MaxRetrieveLen int
}

type Handler func(lg log.Log, typ byte, d byte, r *reader.Fetcher, p Pusher, wg *sync.WaitGroup, retrieverCancels *session.RetrieverCancels, c *Config) error

func dispatch(lg log.Log, req []byte, handler Handler, p Pusher, wg *sync.WaitGroup, breakOnHandlerError bool, retrieverCancels *session.RetrieverCancels, c Config) error {
	r := reader.NewFetcher(reader.ByteFetch(req, ErrDispatchCompleted))
	for {
		t, err := r.Fetch(1)
		if err == ErrDispatchCompleted {
			return nil
		} else if err != nil {
			return err
		}
		rType, rData := protocol.ParseRequestType(protocol.RequestType(t[0]))
		err = handler(lg, rType, rData, &r, p, wg, retrieverCancels, &c)
		if err == nil {
			continue
		}
		if !breakOnHandlerError {
			continue
		}
		return err
	}
}

type PusherExecuter func(p *reader.Pusher) error
type Pusher func(PusherExecuter) error

func LockedPusher(l *sync.Mutex, p *reader.Pusher) Pusher {
	return func(pp PusherExecuter) error {
		l.Lock()
		defer l.Unlock()
		return pp(p)
	}
}
