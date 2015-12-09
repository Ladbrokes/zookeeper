/*
 *   Zookeeper - Multi-interface proxy for those times when developers need public IPs
 *   Copyright (c) 2015 Shannon Wynter, Ladbrokes Digital Australia Pty Ltd.
 *
 *   This program is free software: you can redistribute it and/or modify
 *   it under the terms of the GNU General Public License as published by
 *   the Free Software Foundation, either version 3 of the License, or
 *   (at your option) any later version.
 *
 *   This program is distributed in the hope that it will be useful,
 *   but WITHOUT ANY WARRANTY; without even the implied warranty of
 *   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 *   GNU General Public License for more details.
 *
 *   You should have received a copy of the GNU General Public License
 *   along with this program.  If not, see <http://www.gnu.org/licenses/>.
 *
 *   Author: Shannon Wynter <http://fremnet.net/contact>
 */

package main

import (
	"fmt"
	"io/ioutil"
	"time"

	"github.com/BurntSushi/toml"
)

type configuration struct {
	md                   toml.MetaData             `toml:"-"`
	Listen               string                    `toml:"listen"`
	TLS                  tlsConfiguration          `toml:"tls"`
	AuthenticationMethod string                    `toml:"authentication_method"`
	AuthenticationConfig map[string]toml.Primitive `toml:"authentication"`
	AccessControlMethod  string                    `toml:"accesscontrol_method"`
	AccessControlConfig  map[string]toml.Primitive `toml:"accesscontrol"`
	Addresses            ipAddressesConfiguration  `toml:"address"`
	StateSaver           stateSaverConfiguration   `toml:"statesaver"`
	MaxTTL               duration                  `toml:"max_ttl"`
}

func loadConfiguration(file string) (*configuration, error) {
	var err error
	config := configuration{
		Listen: ":8080",
	}
	config.md, err = toml.DecodeFile(file, &config)
	return &config, err
}

type tlsConfiguration struct {
	Certificate keyFile `toml:"certificate"`
	Key         keyFile `toml:"key"`
}

type jwtConfiguration struct {
	Private keyFile `toml:"private"`
	Public  keyFile `toml:"public"`
	Enabled bool    `toml:"enabled"`
}

type keyFile []byte

func (f *keyFile) UnmarshalText(text []byte) error {
	if string(text[0:7]) == "file://" {
		keybytes, err := ioutil.ReadFile(string(text[7:]))
		if err != nil {
			return fmt.Errorf("Unable to read keyfile: %s", err)
		}
		*f = keybytes
	} else {
		*f = text
	}
	return nil
}

type ipAddressesConfiguration map[string]ipAddressConfiguration

type ipAddressConfiguration struct {
	Description string `toml:"description" json:"description"`
}

func (c *configuration) UnifyAuthenticationConfiguration(name string, v interface{}) (err error) {
	if c.md.IsDefined("authentication", name) {
		err = c.md.PrimitiveDecode(c.AuthenticationConfig[name], v)
	}
	return
}

func (c *configuration) UnifyAccessControlConfiguration(name string, v interface{}) (err error) {
	if c.md.IsDefined("accesscontrol", name) {
		err = c.md.PrimitiveDecode(c.AccessControlConfig[name], v)
	}
	return
}

type stateSaverConfiguration struct {
	Enabled  bool      `toml:"enabled"`
	Interval *duration `toml:"interval"`
	File     string    `toml:"file"`
}

type duration struct {
	time.Duration
}

func (d *duration) UnmarshalText(text []byte) error {
	var err error
	d.Duration, err = time.ParseDuration(string(text))
	return err
}
