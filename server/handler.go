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

package server

import (
	cph "crypto/cipher"
	"errors"
	"io"
	"net/http"
	"sync"
	"warwolf/buffer"
	"warwolf/cipher"
	"warwolf/dispatch"
	"warwolf/log"
	"warwolf/protocol"
	"warwolf/reader"
)

var (
	errHTTPSubmitEOF = errors.New("HTTP: All read")
)

const (
	respondHeaderSize  = cipher.OverheadSize
	maxRespondDataSize = rwBufferSize - (respondHeaderSize + protocol.GreatestHeaderSize)
)

type handler struct {
	lg       log.Log
	dispatch *dispatch.Responder
	buffer   *buffer.Buffer
	key      cipher.KeyGen
	nv       cipher.NonceVerifier
}

func (h *handler) Serve(w http.ResponseWriter, r *http.Request) {
	name := r.RemoteAddr
	rbuf := h.buffer.Request()
	defer h.buffer.Return(rbuf)
	if r.ContentLength <= 0 || r.ContentLength > int64(len(rbuf)) {
		h.lg("%s: Invalid request: Invalid request size: %d", name, r.ContentLength)
		w.WriteHeader(http.StatusOK)
		return
	}
	if r.Body == nil {
		h.lg("%s: Invalid request: No request body", name)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	defer r.Body.Close()
	rlen, err := io.ReadFull(r.Body, rbuf[:r.ContentLength])
	if err != nil {
		h.lg("%s: Invalid request: %s", name, err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	key, keyTime := h.key.Get()
	cip, err := cipher.AEAD(key)
	if err != nil {
		h.lg("%s: Unable to create cipher: %s", name, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	h.lg("%s: Has arrived", name)
	defer h.lg("%s: Has left", name)
	w.Header().Add("Transfer-Encoding", "chunked")
	w.Header().Add("Content-Type", "application/octet-stream")
	w.Header().Add("Cache-Control", "no-store")
	w.WriteHeader(http.StatusOK)
	pbuf := h.buffer.Request()
	defer h.buffer.Return(pbuf)
	f := reader.NewFetcher(reader.ByteFetch(rbuf[:rlen], errHTTPSubmitEOF))
	p := reader.NewPusher(pbuf)
	plock := sync.Mutex{}
	err = cipher.Decrypt(keyTime, func() (cph.AEAD, error) {
		return cip, nil
	}, h.nv, &f, errHTTPSubmitEOF, func(b []byte) error {
		return h.dispatch.Dispatch(func(format string, v ...interface{}) {
			h.lg(name+": "+format, v...)
		}, b, func(pp dispatch.PusherExecuter) error {
			vkey, _ := h.key.Get()
			vcip, verr := cipher.AEAD(vkey)
			if verr != nil {
				return verr
			}
			plock.Lock()
			defer plock.Unlock()
			p.Truncate(cipher.HeaderSize)
			defer p.Truncate(0)
			e := pp(&p)
			if e != nil {
				return e
			}
			p.Truncate(p.Size() + cipher.BlockSize)
			nonce, e := cipher.Nonce()
			if e != nil {
				return e
			}
			_, e = w.Write(cipher.Encrypt(vcip, nonce, p.Data()))
			if e != nil {
				return e
			}
			w.(http.Flusher).Flush()
			return nil
		}, dispatch.Config{
			MaxRetrieveLen: maxRespondDataSize,
		})
	})
	if err != nil {
		h.lg("%s: Response failed: %s", name, err)
		return
	}
}
