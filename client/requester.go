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
	"bytes"
	"context"
	cph "crypto/cipher"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"sync"
	"time"
	"warwolf/buffer"
	"warwolf/cipher"
	"warwolf/dispatch"
	"warwolf/log"
	"warwolf/protocol"
	"warwolf/reader"
	"warwolf/server"
	"warwolf/session"
)

const (
	requesterReadWriteBufferSize  = server.MaxRequestBodySize
	requestMaxHTTPReqSize         = requesterReadWriteBufferSize
	requestReqOverheadSize        = cipher.OverheadSize
	requestReqPadSize             = cipher.HeaderSize
	requestMaxReqPayloadSize      = requestMaxHTTPReqSize - requestReqOverheadSize
	requestReqSendDelay           = 128 * time.Millisecond
	requestReqSendShortDelay      = 8 * time.Millisecond
	requestReqSendSwitchThreshold = 128 * time.Millisecond
)

var (
	ErrRequestBodyTooLarge = session.RetrieverError{
		E:        errors.New("Requester: Request body too large"),
		TryAgain: true,
	}

	ErrRequestCipherFailed = session.RetrieverError{
		E:        errors.New("Requester: Cipher failed"),
		TryAgain: true,
	}

	ErrRequestUnresponded = session.RetrieverError{
		E:        errors.New("Requester: Unresponded"),
		TryAgain: true,
	}

	ErrRequestHTTPNoRespondBody = session.RetrieverError{
		E:        errors.New("Requester: HTTP responded with no data"),
		TryAgain: true,
	}
)

type requestBodyReadCloser struct {
	*bytes.Buffer
}

func (r requestBodyReadCloser) Close() error {
	return nil
}

func newClient(c Config) http.Client {
	dl := net.Dialer{
		Timeout:   c.RequestTimeout,
		KeepAlive: c.IdleTimeout,
	}
	dial := func(ctx context.Context, network, addr string) (net.Conn, error) {
		return dl.DialContext(ctx, network, addr)
	}
	if len(c.BackendHostEnforce) > 0 {
		dial = func(ctx context.Context, network, addr string) (net.Conn, error) {
			return dl.DialContext(ctx, network, c.BackendHostEnforce)
		}
	}
	return http.Client{
		Transport: &http.Transport{
			DialContext:           dial,
			IdleConnTimeout:       c.IdleTimeout,
			ResponseHeaderTimeout: c.RequestTimeout,
			WriteBufferSize:       requesterReadWriteBufferSize,
			ReadBufferSize:        requesterReadWriteBufferSize,
		},
		CheckRedirect: nil,
		Jar:           nil,
		Timeout:       c.IdleTimeout,
	}
}

func buildRequestCipher(key *cipher.KeyGen) (cph.AEAD, cipher.Time, [12]byte, error) {
	k, t := key.Get()
	cip, err := cipher.AEAD(k)
	if err != nil {
		return cip, t, [12]byte{}, ErrRequestCipherFailed
	}
	n, err := cipher.Nonce()
	if err != nil {
		return cip, t, [12]byte{}, ErrRequestCipherFailed
	}
	return cip, t, n, err
}

func sendRequest(lg log.Log, b *buffer.Buffer, key *cipher.KeyGen, nv cipher.NonceVerifier, dis *dispatch.Requester, address *url.URL, cookies func() map[string]http.Cookie, rspp func(r *http.Response), body []byte, client *http.Client, retrieverCancels *session.RetrieverCancels) error {
	start := time.Now()
	cip, t, n, err := buildRequestCipher(key)
	if err != nil {
		return err
	}
	body = cipher.Encrypt(cip, n, body)
	reqBody := requestBodyReadCloser{
		Buffer: bytes.NewBuffer(body),
	}
	req := http.Request{
		Header:        http.Header{"Connection": []string{"keep-alive"}},
		Method:        "POST",
		URL:           address,
		Body:          reqBody,
		ContentLength: int64(reqBody.Len()),
	}
	for _, c := range cookies() {
		req.AddCookie(&c)
	}
	rsp, err := client.Do(&req)
	cost := time.Now().Sub(start)
	if err != nil {
		lg("HTTP request failed after %s: %s, retrying ...", cost, err)
		return err
	}
	if rsp.Body == nil {
		lg("HTTP request failed after %s: %s, retrying ...", cost, err)
		return ErrRequestHTTPNoRespondBody
	}
	lg("HTTP request has been responded after %s, status: %s", cost, rsp.Status)
	rspp(rsp)
	defer rsp.Body.Close()
	bb := b.Request()
	defer b.Return(bb)
	rspfetch := reader.NewFetcher(reader.ReaderFetch(bb, rsp.Body, io.EOF))
	var disErr error
	err = cipher.Decrypt(t, func() (cph.AEAD, error) {
		k, _ := key.Get()
		cip, err := cipher.AEAD(k)
		if err != nil {
			return cip, ErrRequestCipherFailed
		}
		return cip, nil
	}, nv, &rspfetch, io.EOF, func(b []byte) error {
		lg("A segment of %d bytes respond data is received", len(b))
		disErr = dis.Dispatch(func(format string, v ...interface{}) {
			lg("Dispatch: "+format, v...)
		}, b, retrieverCancels)
		return disErr
	})
	if err == disErr {
		return disErr
	}
	if disErr != nil {
		return disErr
	}
	if err != nil {
		return session.RetrieverError{
			E:        errors.New("Requester: Cipher failed: " + err.Error()),
			TryAgain: true,
		}
	}
	return nil
}

type request struct {
	id     protocol.ID
	pusher *reader.Pusher
	cancel session.RetrieverCancel
}

func getRequestRetryDelay(max time.Duration, i, n float64) time.Duration {
	dd := time.Duration(float64(max) / (n - i))
	if dd > max {
		return max
	}
	return dd
}

type requester struct {
	lg                         log.Log
	b                          *buffer.Buffer
	key                        cipher.KeyGen
	nv                         cipher.NonceVerifier
	url                        *url.URL
	requester                  http.Client
	current                    int32
	wait                       sync.WaitGroup
	session                    *session.Retrievers
	dispatch                   *dispatch.Requester
	requests                   chan request
	wrequests                  chan request
	maxHTTPReqBodySize         int
	requestReqPadSize          int
	requestReqOverheadSize     int
	requestMaxReqPayloadSize   int
	maxConcurrentRequests      int
	maxRetries                 int
	maxRetryDelay              time.Duration
	requestSendDelay           time.Duration
	requestSendShortDelay      time.Duration
	requestSendSwitchThreshold time.Duration
}

func newRequester(
	lg log.Log,
	b *buffer.Buffer,
	url *url.URL,
	session *session.Retrievers,
	dispatch *dispatch.Requester,
	nv cipher.NonceVerifier,
	c Config,
) requester {
	return requester{
		lg:                         lg,
		b:                          b,
		key:                        cipher.KeyGen{Key: c.Key},
		nv:                         nv,
		url:                        url,
		requester:                  newClient(c),
		current:                    0,
		wait:                       sync.WaitGroup{},
		session:                    session,
		dispatch:                   dispatch,
		requests:                   make(chan request),
		wrequests:                  make(chan request),
		maxHTTPReqBodySize:         requestMaxHTTPReqSize,
		requestReqPadSize:          requestReqPadSize,
		requestReqOverheadSize:     requestReqOverheadSize,
		requestMaxReqPayloadSize:   requestMaxReqPayloadSize,
		maxConcurrentRequests:      c.MaxBackendConnections,
		maxRetries:                 c.MaxRetries,
		maxRetryDelay:              c.RequestTimeout,
		requestSendDelay:           requestReqSendDelay,
		requestSendShortDelay:      requestReqSendShortDelay,
		requestSendSwitchThreshold: requestReqSendSwitchThreshold,
	}
}

func (r *requester) serve(name string, rchan chan request, wg *sync.WaitGroup) {
	defer wg.Done()
	var timerChan <-chan time.Time = nil
	timer := time.NewTimer(r.requestSendDelay)
	defer timer.Stop()

	fullbuf := make([]byte, r.maxHTTPReqBodySize)
	paddedbuf := fullbuf[r.requestReqPadSize:r.requestReqPadSize]
	cancels := make(session.RetrieverCancels, 256)
	requests := rchan
	runlgs := func(format string, v ...interface{}) { r.lg(name+": "+format, v...) }
	lastReq := time.Now()
	cookies := make(map[string]http.Cookie, 32)
	reqcookies := func() map[string]http.Cookie {
		n := time.Now()
		for k, c := range cookies {
			if c.Expires.IsZero() || c.Expires.After(n) {
				continue
			}
			delete(cookies, k)
		}
		return cookies
	}
	rspparse := func(rsp *http.Response) {
		for _, c := range rsp.Cookies() {
			cookies[c.Name] = *c
		}
	}

	for {
		select {
		case rr, ok := <-requests:
			if !ok {
				return
			}
			if len(paddedbuf)+rr.pusher.Size() > r.requestMaxReqPayloadSize {
				runlgs("Sending %d requests (buffer full)", len(cancels))
				res := sendRequest(runlgs, r.b, &r.key, r.nv, r.dispatch, r.url, reqcookies, rspparse, fullbuf[:r.requestReqOverheadSize+len(paddedbuf)], &r.requester, &cancels)
				if res != nil {
					runlgs("Request failed: %s", res)
				} else {
					runlgs("Request successful")
				}
				if res != nil {
					runlgs(res.Error())
				}
				cancels.SettleAll(ErrRequestUnresponded)
				paddedbuf = paddedbuf[:0]
			}
			if len(paddedbuf)+rr.pusher.Size() > r.requestMaxReqPayloadSize {
				rr.cancel(ErrRequestBodyTooLarge)
				continue
			}
			paddedbuf = append(paddedbuf, rr.pusher.Data()...)
			cancels.Append(rr.id, rr.cancel)
			curtime := time.Now()
			if curtime.Sub(lastReq) < r.requestSendSwitchThreshold {
				timer.Reset(r.requestSendShortDelay)
			} else {
				timer.Reset(r.requestSendDelay)
			}
			timerChan = timer.C
			lastReq = curtime

		case <-timerChan:
			timerChan = nil
			requests = rchan
			if len(paddedbuf) == 0 {
				continue
			}
			runlgs("Sending %d requests (flush timer)", len(cancels))
			res := sendRequest(runlgs, r.b, &r.key, r.nv, r.dispatch, r.url, reqcookies, rspparse, fullbuf[:r.requestReqOverheadSize+len(paddedbuf)], &r.requester, &cancels)
			if res != nil {
				runlgs("Request failed: %s", res)
			} else {
				runlgs("Request successful")
			}
			if res != nil {
				runlgs(res.Error())
			}
			cancels.SettleAll(ErrRequestUnresponded)
			paddedbuf = paddedbuf[:0]
		}
	}
}

func (r *requester) init() {
	r.wait.Add(r.maxConcurrentRequests + 1)
	go r.serve("Requester 0", r.wrequests, &r.wait)
	for i := 0; i < r.maxConcurrentRequests; i++ {
		go r.serve(fmt.Sprintf("Requester %d", i+1), r.requests, &r.wait)
	}
}

func (r *requester) kill() {
	close(r.requests)
	close(r.wrequests)
	r.wait.Wait()
}

func (r *requester) run(c func() error) error {
	result := session.RetrieverError{}
	for i := 0; i < r.maxRetries; i++ {
		sessErr := c()
		if sessErr == nil {
			return nil
		}
		err, ok := sessErr.(session.RetrieverError)
		if !ok {
			return sessErr
		}
		result = err
		if !result.IsError() {
			return nil
		}
		if !result.TryAgain {
			return err.E
		}
		time.Sleep(getRequestRetryDelay(
			r.maxRetryDelay,
			float64(i),
			float64(r.maxRetries),
		))
	}
	return result.E
}

func (r *requester) dial(
	rr protocol.DialRequest,
	p *reader.Pusher,
	resp session.RetrieverBuilder,
	after func(),
) (session.Retriever, error) {
	defer after()
	pt := p.Size()
	defer p.Truncate(pt)
	id, ret, err := r.session.Retriever(resp)
	if err != nil {
		return nil, err
	}
	err = r.run(func() error {
		p.Truncate(pt)
		sErr := make(chan session.RetrieverError, 1)
		cc, err := r.session.Register(id, rr, p, ret, func(e session.RetrieverError) {
			sErr <- e
		})
		if err != nil {
			return err
		}
		r.requests <- request{
			id:     id,
			pusher: p,
			cancel: cc,
		}
		return <-sErr
	})
	if err != nil {
		r.session.Release(id, func(e session.Retriever) error {
			e.Close()
			return nil
		})
		return nil, err
	}
	return ret, nil
}

func (r *requester) retrieve(
	id protocol.ID,
	p *reader.Pusher,
) error {
	pt := p.Size()
	defer p.Truncate(pt)
	return r.run(func() error {
		p.Truncate(pt)
		sErr := make(chan session.RetrieverError, 1)
		cc, err := r.session.Retrieve(id, p, func(e session.RetrieverError) {
			sErr <- e
		})
		if err != nil {
			return err
		}
		r.requests <- request{
			id:     id,
			pusher: p,
			cancel: cc,
		}
		return <-sErr
	})
}

func (r *requester) send(
	id protocol.ID,
	req []byte,
	p *reader.Pusher,
) (int, error) {
	type writeResult struct {
		l int
		e session.RetrieverError
	}
	wres := writeResult{
		l: 0,
		e: session.RetrieverError{E: nil, TryAgain: false},
	}
	rlen := len(req)
	pt := p.Size()
	defer p.Truncate(pt)
	e := r.run(func() error {
		p.Truncate(pt)
		sErr := make(chan writeResult, 1)
		cc, err := r.session.Send(id, req, p, func(size uint16, e session.RetrieverError) {
			sErr <- writeResult{l: int(size), e: e}
		})
		if err != nil {
			return err
		}
		p.Truncate(p.Size() + (rlen - protocol.SendHeaderOverhead))
		select {
		case r.requests <- request{id: id, pusher: p, cancel: cc}:
		case r.wrequests <- request{id: id, pusher: p, cancel: cc}:
		}
		wres = <-sErr
		return wres.e
	})
	if wres.e.IsError() {
		return wres.l, wres.e
	}
	return wres.l, e
}

func (r *requester) close(id protocol.ID, p *reader.Pusher) error {
	pt := p.Size()
	defer p.Truncate(pt)
	return r.run(func() error {
		p.Truncate(pt)
		sErr := make(chan session.RetrieverError, 1)
		cc, err := r.session.Close(id, p, func(e session.RetrieverError) {
			sErr <- e
		})
		if err != nil {
			return err
		}
		select {
		case r.requests <- request{id: id, pusher: p, cancel: cc}:
		case r.wrequests <- request{id: id, pusher: p, cancel: cc}:
		}
		return <-sErr
	})
}
