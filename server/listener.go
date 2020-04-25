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
	"crypto/tls"
	"log"
	"net/http"
	"sync"
	"time"
	"warwolf/buffer"
	"warwolf/cipher"
	"warwolf/dispatch"
	"warwolf/relay"
	"warwolf/session"
)

const (
	rwBufferSize          = (1024 * 64) - 1 // -1 to avoid overflow
	defaultNonceStoreSize = 512
	MaxRequestBodySize    = rwBufferSize
)

type Listener struct{}

func (l Listener) Listen(config Config) error {
	log.Printf("Warwolf System is starting up as backend server")
	log.Printf("(C) 2020 The Warwolf Authors. All rights reserved")
	log.Printf("The right to communicate freely, privately and securely is fundamental part of a safe society")
	c, err := config.Load().Verify()
	if err != nil {
		log.Fatalf("Configuration error: %s", err)
		return err
	}
	wg := sync.WaitGroup{}
	defer wg.Wait()
	closeChan := make(chan struct{})
	defer close(closeChan)
	sess := session.New(c.MaxOutgoingConnections, c.IdleTimeout)
	defer sess.CloseAll()
	wg.Add(1)
	go func() {
		defer wg.Done()
		timer := time.NewTimer(c.IdleTimeout / 2)
		for {
			select {
			case <-timer.C:
				sess.Recycle()
			case <-closeChan:
				return
			}
		}
	}()
	buf := buffer.New(rwBufferSize, c.MaxOutgoingConnections*2)
	rsp := dispatch.NewResponder(&sess, nil, relay.Config{
		DialTimeout:     c.DialTimeout,
		RetrieveTimeout: c.RetrieveTimeout,
	}, &buf)
	lgg := func(format string, v ...interface{}) {
		log.Printf(format, v...)
	}
	if !c.Logging {
		lgg = func(format string, v ...interface{}) {}
	}
	nonces := cipher.NewNonces(defaultNonceStoreSize, &sync.Mutex{})
	handler := handler{
		lg:       lgg,
		dispatch: &rsp,
		buffer:   &buf,
		key:      cipher.KeyGen{Key: c.Key},
		nv:       nonces.Verify,
	}
	var tlsConfig *tls.Config
	if len(c.TLSPublicKeyBlock) > 0 && len(c.TLSPrivateKeyBlock) > 0 {
		cert, err := tls.X509KeyPair(
			c.TLSPublicKeyBlock,
			c.TLSPrivateKeyBlock,
		)
		if err != nil {
			log.Printf("Invalid X509 key pair: %s", err)
			return err
		}
		tlsConfig = &tls.Config{
			Certificates: []tls.Certificate{cert},
		}
		log.Printf("TLS enabled")
	}
	server := http.Server{
		Addr:              c.Listen,
		Handler:           http.HandlerFunc(handler.Serve),
		TLSConfig:         tlsConfig,
		ReadTimeout:       c.IdleTimeout,
		ReadHeaderTimeout: c.RetrieveTimeout,
		WriteTimeout:      c.IdleTimeout,
		IdleTimeout:       c.IdleTimeout,
	}
	log.Printf("Start listening on %s", c.Listen)
	var e error
	defer func() {
		if e == nil {
			log.Printf("Shutting down")
			return
		}
		log.Printf("Shutting down: %s", e)
	}()
	e = server.ListenAndServe()
	return e
}

func New() Listener {
	return Listener{}
}
