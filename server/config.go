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
	"fmt"
	"strings"
	"time"
	"warwolf/config"
)

type Config struct {
	Listen                 string
	Key                    []byte
	Logging                bool
	IdleTimeout            time.Duration
	RetrieveTimeout        time.Duration
	DialTimeout            time.Duration
	MaxOutgoingConnections int
	TLSPublicKeyBlock      []byte
	TLSPrivateKeyBlock     []byte
}

func (c Config) Load() Config {
	return Config{
		Listen:                 strings.TrimSpace(config.HostPortDefault("Listen", ":80")),
		Key:                    []byte(strings.TrimSpace(config.LoadStringDefault("Key", "TheRightToCommunicateFreelyPrivatelySecretlyAndSecurelyIsEssentialForASafeSociety"))),
		Logging:                strings.ToLower(strings.TrimSpace(config.LoadStringDefault("Logging", "yes"))) == "yes",
		IdleTimeout:            config.LoadTimeDurationDefault("IdleTimeout", 120*time.Second),
		RetrieveTimeout:        config.LoadTimeDurationDefault("RetrieveTimeout", 3*time.Second),
		DialTimeout:            config.LoadTimeDurationDefault("DialTimeout", 5*time.Second),
		MaxOutgoingConnections: int(config.LoadUint16Default("MaxOutgoingConnections", 128)),
		TLSPublicKeyBlock:      []byte(strings.TrimSpace(config.LoadString("TLSPublicKeyBlock"))),
		TLSPrivateKeyBlock:     []byte(strings.TrimSpace(config.LoadString("TLSPrivateKeyBlock"))),
	}
}

func (c Config) Verify() (Config, error) {
	if len(c.Listen) == 0 {
		return c, fmt.Errorf("Option \"Listen\" is required")
	}
	if len(c.Key) == 0 {
		return c, fmt.Errorf("Option \"Key\" is required")
	}
	if c.IdleTimeout <= c.RetrieveTimeout {
		return c, fmt.Errorf("Option \"IdleTimeout\" is required and must be greater than \"RetrieveTimeout\" which is currently %s", c.RetrieveTimeout)
	}
	if c.RetrieveTimeout < 1*time.Second {
		return c, fmt.Errorf("Option \"RetrieveTimeout\" is required and must not be smaller than %s", 1*time.Second)
	}
	if c.DialTimeout < 1*time.Second {
		return c, fmt.Errorf("Option \"DialTimeout\" is required and must not smaller than %s", 1*time.Second)
	}
	if c.MaxOutgoingConnections < 0 {
		return c, fmt.Errorf("Option \"MaxOutgoingConnections\" is required and must not smaller than 0")
	}
	return c, nil
}
