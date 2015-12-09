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

import "log"

const staticAuthenticationName = "static"

type staticAuthentication struct {
	config  staticAuthenticationConfiguration
	enabled bool
}

type staticAuthenticationConfiguration struct {
	Username string `toml:"username"`
}

func (a *staticAuthentication) Init(c *configuration, e EchoMiddlewareUser) (err error) {
	log.Println("staticAuthentication initializing")
	a.config = staticAuthenticationConfiguration{
		Username: "super.developer",
	}

	if err = c.UnifyAuthenticationConfiguration(staticAuthenticationName, &a.config); err != nil {
		return
	}

	log.Println("\tVerifying configuration")
	configured := a.config.Username != ""

	if !configured {
		log.Println("staticAuthentication not configured")
		return
	}

	a.enabled = true
	log.Println("staticAuthentication ready")
	return
}

func (a *staticAuthentication) Authenticated(c EchoStasher) (bool, string) {
	if !a.enabled {
		return true, ""
	}
	return true, a.config.Username
}

func init() {
	RegisterAuthenticationInterface(staticAuthenticationName, func() AuthenticationInterface {
		return &staticAuthentication{}
	})
}
