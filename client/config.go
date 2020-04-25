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
	"fmt"
	"strings"
	"time"
	"warwolf/config"
)

type Config struct {
	Backend               string
	Key                   []byte
	Listen                string
	Username              string
	Password              string
	BackendHostEnforce    string
	MaxClientConnections  int
	MaxBackendConnections int
	MaxRetrieveLength     uint16
	RequestTimeout        time.Duration
	IdleTimeout           time.Duration
	MaxRetries            int
}

func (c Config) Load() Config {
	return Config{
		Backend:               strings.TrimSpace(config.LoadString("Backend")),
		Key:                   []byte(strings.TrimSpace(config.LoadStringDefault("Key", "TheRightToCommunicateFreelyPrivatelySecretlyAndSecurelyIsEssentialForASafeSociety"))),
		Listen:                strings.TrimSpace(config.HostPortDefault("Listen", "127.0.0.1:1080")),
		Username:              strings.TrimSpace(config.LoadString("Username")),
		Password:              strings.TrimSpace(config.LoadString("Password")),
		BackendHostEnforce:    strings.TrimSpace(config.HostPortDefault("BackendHostEnforce", "")),
		MaxClientConnections:  int(config.LoadUint16Default("MaxClientConnections", 128)),
		MaxBackendConnections: int(config.LoadUint16Default("MaxBackendConnections", 5)),
		MaxRetrieveLength:     config.LoadUint16Default("MaxRetrieveLength", requestMaxReqPayloadSize),
		RequestTimeout:        config.LoadTimeDurationDefault("RequestTimeout", 32*time.Second),
		IdleTimeout:           config.LoadTimeDurationDefault("IdleTimeout", 128*time.Second),
		MaxRetries:            int(config.LoadUint16Default("MaxRetries", 6)),
	}
}

func (c Config) Verify() (Config, error) {
	if len(c.Backend) == 0 {
		return c, fmt.Errorf("Option \"Backend\" is required")
	}
	if len(c.Key) == 0 {
		return c, fmt.Errorf("Option \"Key\" is required")
	}
	if len(c.Listen) == 0 {
		return c, fmt.Errorf("Option \"Listen\" is required")
	}
	if c.MaxClientConnections < 1 {
		return c, fmt.Errorf("Option \"MaxClientConnections\" is required and must be greater than 0")
	}
	if c.MaxBackendConnections < 1 {
		return c, fmt.Errorf("Option \"MaxBackendConnections\" is required and must be greater than 0")
	}
	if c.MaxRetrieveLength < 1 {
		return c, fmt.Errorf("Option \"MaxRetrieveLength\" is required and must be greater than 0")
	}
	if c.RequestTimeout < 1*time.Second {
		return c, fmt.Errorf("Option \"RequestTimeout\" is required and must be greater than %s", 1*time.Second)
	}
	if c.IdleTimeout < c.RequestTimeout {
		return c, fmt.Errorf("Option \"IdleTimeout\" is required and must be greater than the \"RequestTimeout\" which currently is %s", c.RequestTimeout)
	}
	if c.MaxRetries < 1 {
		return c, fmt.Errorf("Option \"MaxRetries\" is required and must be greater than 0")
	}
	return c, nil
}
