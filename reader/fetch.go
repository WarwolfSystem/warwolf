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
	"io"
)

var (
	ErrAllFetched      = errors.New("Fetcher: All fetched")
	ErrNotEnoughBuffer = errors.New("Fetcher: Not enough buffer to be fetched")
	ErrNotEnoughData   = errors.New("Fetcher: Not enough data to be fetched")
	ErrBufferNotEmpty  = errors.New("Fetcher: There are still some data left to fetch in the buffer")
)

const (
	MaxFetchSize = int((^uint(0)) >> 1)
)

type Fetch func(min int, oldbuf []byte, oldbufused int) ([]byte, error)

func ReaderFetch(b []byte, r io.Reader, allFetchedErr error) Fetch {
	blen := len(b)
	return func(min int, oldbuf []byte, oldbufused int) ([]byte, error) {
		oldbuflen := len(oldbuf)
		oldbufremain := oldbuflen - oldbufused
		toread := min - oldbufremain
		if toread > blen {
			return nil, ErrNotEnoughBuffer
		}
		m := copy(b, oldbuf[oldbufused:])
		l, e := io.ReadAtLeast(r, b[m:], toread)
		if e != nil {
			return nil, e
		}
		return b[:l+m], nil
	}
}

func ByteFetch(b []byte, allFetchedErr error) Fetch {
	fetched := false
	return func(min int, oldbuf []byte, oldbufused int) ([]byte, error) {
		if fetched {
			return nil, allFetchedErr
		}
		if len(oldbuf) != oldbufused {
			return nil, ErrBufferNotEmpty
		}
		fetched = true
		return b, nil
	}
}

func SizeLimitedFetch(size int, f *Fetcher) Fetch {
	remain := size
	return func(min int, oldbuf []byte, oldbufused int) ([]byte, error) {
		if remain == 0 {
			return nil, ErrAllFetched
		}
		if len(oldbuf) != oldbufused {
			return nil, ErrBufferNotEmpty
		}
		b, err := f.FetchMax(remain)
		remain -= len(b)
		if err != nil {
			return b, err
		}
		return b, nil
	}
}

func FetchAll(size int, f *Fetcher, fetched func(b []byte)) error {
	remain := size - f.Total()
	ff := NewFetcher(SizeLimitedFetch(remain, f))
	for {
		b, e := ff.FetchMax(remain)
		if e == ErrAllFetched {
			return nil
		}
		if e != nil {
			return e
		}
		fetched(b)
	}
}

type Fetcher struct {
	f    Fetch
	s    []byte
	used int
	read int
}

func NewFetcher(f Fetch) Fetcher {
	return Fetcher{
		f:    f,
		s:    nil,
		used: 0,
		read: 0,
	}
}

func (f *Fetcher) fetch(min int) ([]byte, error) {
	if len(f.s)-f.used >= min {
		return f.s[f.used:], nil
	}
	ff, err := f.f(min, f.s, f.used)
	if err != nil {
		return nil, err
	}
	f.s = ff
	f.used = 0
	return f.s, nil
}

func (f *Fetcher) FetchMax(max int) ([]byte, error) {
	ff, err := f.fetch(1)
	if err != nil {
		return nil, err
	}
	fLen := len(ff)
	if fLen > max {
		fLen = max
	}
	f.used += fLen
	f.read += fLen
	return ff[:fLen], nil
}

func (f *Fetcher) Fetch(n int) ([]byte, error) {
	if n == 0 {
		return nil, nil
	}
	ff, err := f.fetch(n)
	if err != nil {
		return nil, err
	}
	if len(ff) < n {
		return nil, ErrNotEnoughBuffer
	}
	f.used += n
	f.read += n
	return ff[:n], nil
}

func (f *Fetcher) Read(b []byte) (int, error) {
	if len(b) == 0 {
		return 0, io.EOF
	}
	ff, err := f.FetchMax(len(b))
	if err != nil {
		return 0, err
	}
	return copy(b, ff), nil
}

func (f *Fetcher) Total() int {
	return f.read
}
