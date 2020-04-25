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

package config

import (
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

func LoadString(name string) string {
	v := os.Getenv("WWF" + name)
	if strings.HasPrefix(v, "$") {
		return os.Getenv(v[1:])
	}
	return v
}

func LoadStringDefault(name string, def string) string {
	v := LoadString(name)
	if len(v) > 0 {
		return v
	}
	return def
}

func HostPortDefault(name string, def string) string {
	v := LoadString(name)
	if len(v) == 0 {
		return def
	}
	host, port, err := net.SplitHostPort(v)
	if err != nil {
		_, err = strconv.ParseInt(v, 10, 64)
		if err == nil {
			port = v
		} else {
			return def
		}
	}
	return net.JoinHostPort(host, port)
}

func LoadUint16(name string) uint16 {
	v := LoadString(name)
	if len(v) == 0 {
		return 0
	}
	vv, e := strconv.ParseUint(v, 10, 16)
	if e != nil {
		return 0
	}
	return uint16(vv)
}

func LoadUint16Default(name string, def uint16) uint16 {
	v := LoadUint16(name)
	if v != 0 {
		return v
	}
	return def
}

func LoadTimeDuration(name string) time.Duration {
	v := LoadString(name)
	if len(v) == 0 {
		return 0
	}
	vv, e := strconv.ParseInt(v, 10, 64)
	if e != nil {
		return 0
	}
	return time.Duration(vv) * time.Second
}

func LoadTimeDurationDefault(name string, def time.Duration) time.Duration {
	v := LoadTimeDuration(name)
	if v != 0 {
		return v
	}
	return def
}
