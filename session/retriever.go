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
	"errors"
	"warwolf/protocol"
	"warwolf/reader"
)

var (
	ErrResourceNotFound = newRetrieverError(
		errors.New("Resource: Not found"),
		false,
	)

	ErrResourceNotReady = newRetrieverError(
		errors.New("Resource: Not ready"),
		false,
	)

	ErrResourceExpired = newRetrieverError(
		errors.New("Resource: Expired"),
		true,
	)

	ErrResourceBroken = newRetrieverError(
		errors.New("Resource: Broken"),
		false,
	)

	ErrResourceClosed = newRetrieverError(
		errors.New("Resource: Closed"),
		false,
	)

	ErrResourceSendFailure = newRetrieverError(
		errors.New("Resource: Send failure"),
		false,
	)

	ErrResourceUnknown = newRetrieverError(
		errors.New("Resource: Unknown error"),
		false,
	)

	ErrDialFailedInvalidRequest = newRetrieverError(
		errors.New("Dial failure: Invalid request"),
		false,
	)

	ErrDialFailedUnreachable = newRetrieverError(
		errors.New("Dial failure: Unreachable"),
		true,
	)

	ErrDialFailedOvercapacity = newRetrieverError(
		errors.New("Dial failure: Overcapacity"),
		true,
	)

	ErrDialFailedAlreadyDialed = newRetrieverError(
		errors.New("Dial failure: Already dialed"),
		false,
	)

	ErrDialFailedInternalFailure = newRetrieverError(
		errors.New("Dial failure: Internal failure"),
		false,
	)

	ErrDialFailedUnknown = newRetrieverError(
		errors.New("Dial failure: Unknown"),
		false,
	)

	ErrResumeUnexpectedRespondRetry = newRetrieverError(
		errors.New("Resume failure: Unexpected respond, retry"),
		true,
	)

	ErrRetrieveUnexpectedRespondRetry = newRetrieverError(
		errors.New("Retrieve failure: Unexpected respond, retry"),
		true,
	)

	ErrSendUnexpectedRespondRetry = newRetrieverError(
		errors.New("Send failure: Unexpected respond, retry"),
		true,
	)
)

func getRetrieverResourceError(n byte) RetrieverError {
	switch n {
	case protocol.ResourceErrorSuccess:
		return RetrieverError{}
	case protocol.ResourceErrorNotFound:
		return ErrResourceNotFound
	case protocol.ResourceErrorNotReady:
		return ErrResourceNotReady
	case protocol.ResourceErrorExpired:
		return ErrResourceExpired
	case protocol.ResourceErrorBroken:
		return ErrResourceBroken
	case protocol.ResourceErrorClosed:
		return ErrResourceClosed
	case protocol.ResourceErrorSendFailure:
		return ErrResourceSendFailure
	case protocol.ResourceErrorUnknown:
		return ErrResourceUnknown
	default:
		return ErrResourceUnknown
	}
}

func getRetrieverDialError(n byte) RetrieverError {
	switch n {
	case protocol.DialErrorInvalidRequest:
		return ErrDialFailedInvalidRequest
	case protocol.DialErrorUnreachable:
		return ErrDialFailedUnreachable
	case protocol.DialErrorOverCapacity:
		return ErrDialFailedOvercapacity
	case protocol.DialErrorAlreadyDialed:
		return ErrDialFailedAlreadyDialed
	case protocol.DialErrorInternalFailure:
		return ErrDialFailedInternalFailure
	default:
		return ErrDialFailedUnknown
	}
}

type retrieverErrorReact func(e error)
type RetrieverDialResult func(e RetrieverError)
type RetrieverRetrieveResult func(e RetrieverError)
type RetrieverSendResult func(size uint16, e RetrieverError)
type RetrieverCloseResult func(e RetrieverError)

type Retriever interface {
	Dialed()
	Retrieved(respond []byte)
	Retrieving(bool)
	Serve(b []byte) error
	Close()
}

type retriever struct {
	rec     Retriever
	dialcb  RetrieverDialResult
	rid     uint64
	roffset uint16
	rtotal  uint16
	rcb     RetrieverRetrieveResult
	wid     uint64
	wcb     RetrieverSendResult
	ccb     RetrieverCloseResult
}

func newRetriever(rec Retriever) *retriever {
	return &retriever{
		rec:     rec,
		dialcb:  nil,
		rid:     0,
		roffset: 0,
		rtotal:  0,
		rcb:     nil,
		wid:     0,
		wcb:     nil,
		ccb:     nil,
	}
}

func (r *retriever) dialed(e byte, d *protocol.DialRespond, rr *reader.Fetcher, l *lock, er retrieverErrorReact, c *RetrieverCancels) error {
	if r.dialcb == nil {
		er(ErrNotReady)
		return ErrNotReady
	}
	c.clear(d.ID)
	dialcb := r.dialcb
	r.dialcb = nil
	if e > 0 {
		err := getRetrieverDialError(e)
		er(err)
		l.unlock()
		dialcb(err)
		return err
	}
	r.rid = d.RID
	r.roffset = d.RespondLength
	r.rtotal = d.Total
	roffset := r.roffset
	er(nil)
	l.unlock()
	dialcb(RetrieverError{})
	r.rec.Dialed()
	return reader.FetchAll(int(roffset), rr, func(b []byte) {
		r.rec.Retrieved(b)
	})
}

func (r *retriever) retrieved(e byte, d *protocol.RetrieveRespond, rr *reader.Fetcher, l *lock, er retrieverErrorReact, c *RetrieverCancels) error {
	if r.rcb == nil {
		er(ErrNotReady)
		return ErrNotReady
	}
	c.clear(d.ID)
	rcb := r.rcb
	r.rcb = nil
	if e > 0 {
		err := getRetrieverResourceError(e)
		er(err)
		l.unlock()
		rcb(err)
		return err
	}
	if r.rid > d.RID || d.Offset < r.roffset {
		er(ErrRetrieveUnexpectedRespondRetry)
		l.unlock()
		rcb(ErrRetrieveUnexpectedRespondRetry)
		return nil
	}
	r.roffset = d.Offset + d.PayloadLength
	r.rtotal = d.Total
	roffset := r.roffset
	r.rec.Retrieving(true)
	defer r.rec.Retrieving(false)
	er(nil)
	l.unlock()
	rcb(RetrieverError{})
	return reader.FetchAll(int(roffset), rr, func(b []byte) {
		r.rec.Retrieved(b)
	})
}

func (r *retriever) resumed(e byte, d *protocol.ResumeRespond, rr *reader.Fetcher, l *lock, er retrieverErrorReact, c *RetrieverCancels) error {
	if r.rcb == nil {
		er(ErrNotReady)
		return ErrNotReady
	}
	c.clear(d.ID)
	rcb := r.rcb
	r.rcb = nil
	if e > 0 && (e != protocol.ResourceErrorExpired || d.NewRID != r.rid+1) {
		err := getRetrieverResourceError(e)
		er(err)
		l.unlock()
		rcb(err)
		return err
	}
	if r.rid >= d.NewRID {
		er(ErrResumeUnexpectedRespondRetry)
		l.unlock()
		rcb(ErrResumeUnexpectedRespondRetry)
		return ErrResumeUnexpectedRespondRetry
	}
	r.rid = d.NewRID
	r.roffset = d.PayloadLength
	r.rtotal = d.Total
	roffset := r.roffset
	r.rec.Retrieving(true)
	defer r.rec.Retrieving(false)
	er(nil)
	l.unlock()
	rcb(RetrieverError{})
	return reader.FetchAll(int(roffset), rr, func(b []byte) {
		r.rec.Retrieved(b)
	})
}

func (r *retriever) sent(e byte, d *protocol.SendRespond, l *lock, er retrieverErrorReact, c *RetrieverCancels) error {
	if r.wcb == nil {
		er(ErrNotReady)
		return ErrNotReady
	}
	c.clear(d.ID)
	wcb := r.wcb
	r.wcb = nil
	if e > 0 && (e != protocol.ResourceErrorExpired || d.NewWID != r.wid+1) {
		err := getRetrieverResourceError(e)
		er(err)
		l.unlock()
		wcb(0, err)
		return err
	}
	if r.wid >= d.NewWID {
		er(ErrSendUnexpectedRespondRetry)
		l.unlock()
		wcb(0, ErrSendUnexpectedRespondRetry)
		return ErrSendUnexpectedRespondRetry
	}
	r.wid = d.NewWID
	er(nil)
	l.unlock()
	wcb(d.Sent, RetrieverError{})
	return nil
}

func (r *retriever) release() func() {
	dialcb := r.dialcb
	rcb := r.rcb
	wcb := r.wcb
	ccb := r.ccb
	r.dialcb = nil
	r.rcb = nil
	r.wcb = nil
	r.ccb = nil
	return func() {
		if dialcb != nil {
			dialcb(ErrResourceClosed)
		}
		if rcb != nil {
			rcb(ErrResourceClosed)
		}
		if wcb != nil {
			wcb(0, ErrResourceClosed)
		}
		if ccb != nil {
			ccb(ErrResourceClosed)
		}
		r.rec.Close()
	}
}

func (r *retriever) closed(e byte, d *protocol.CloseRespond, l *lock, er retrieverErrorReact, c *RetrieverCancels) error {
	if r.ccb != nil {
		er(ErrNotReady)
		return ErrNotReady
	}
	c.clear(d.ID)
	ccb := r.ccb
	r.ccb = nil
	u := r.release()
	er(nil)
	l.unlock()
	if ccb != nil {
		ccb(RetrieverError{})
	}
	u()
	return nil
}
