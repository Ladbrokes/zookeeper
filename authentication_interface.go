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

var authenticationInterfaces = map[string]func() AuthenticationInterface{}

type AuthenticationInterface interface {
	Init(*configuration, EchoMiddlewareUser) error
	Authenticated(EchoStasher) (bool, string)
}

func RegisterAuthenticationInterface(name string, f func() AuthenticationInterface) {
	authenticationInterfaces[name] = f
}

func GetAuthenticationInterface(name string) (authenticationInterface AuthenticationInterface) {
	if authenticationInterfacef, ok := authenticationInterfaces[name]; ok {
		authenticationInterface = authenticationInterfacef()
	}
	return
}
