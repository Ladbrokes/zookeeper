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
	"crypto/rsa"
	"fmt"
	"log"
	"net/http"

	"github.com/dgrijalva/jwt-go"
	"github.com/labstack/echo"
)

const jwtrsAuthenticationName = "jwt-rs"

type jwtrsAuthentication struct {
	config  jwtrsAuthenticationConfiguration
	key     *rsa.PublicKey
	enabled bool
}

type jwtrsAuthenticationConfiguration struct {
	Public        keyFile `toml:"public"`
	Header        string  `toml:"header"`
	UsernameClaim string  `toml:"username_claim"`
	StashKey      string  `toml:"stash_key"`
}

func (a *jwtrsAuthentication) Init(c *configuration, e EchoMiddlewareUser) (err error) {
	log.Println("jwtrsAuthentication initializing")
	a.config = jwtrsAuthenticationConfiguration{
		Header:        "X-User-Authenticate",
		UsernameClaim: "user",
		StashKey:      "claims",
	}

	if err = c.UnifyAuthenticationConfiguration(jwtrsAuthenticationName, &a.config); err != nil {
		return
	}

	configured := a.config.Public != nil
	configured = configured && a.config.Header != ""
	configured = configured && a.config.UsernameClaim != ""
	configured = configured && a.config.StashKey != ""

	if !configured {
		log.Println("jwtrsAuthentication not configured")
		return
	}

	log.Println("\tLoading key")
	a.key, err = jwt.ParseRSAPublicKeyFromPEM(a.config.Public)
	if err != nil {
		log.Println("jwtrsAuthentication Unable to aprse RSA public key from PEM")
		return
	}

	log.Println("\tRegistering middleware")
	e.Use(a.middleware())

	a.enabled = true
	log.Println("jwtrsAuthentication ready")
	return
}

func (a *jwtrsAuthentication) middleware() echo.HandlerFunc {
	he := echo.NewHTTPError(http.StatusUnauthorized)

	return func(c *echo.Context) error {
		if !a.enabled {
			return nil
		}

		// Skip WebSocket requests
		if (c.Request().Header.Get(echo.Upgrade)) == echo.WebSocket {
			return nil
		}

		auth := c.Request().Header.Get(a.config.Header)
		if len(auth) > 0 {
			t, err := jwt.Parse(auth, func(token *jwt.Token) (interface{}, error) {
				if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
					return nil, fmt.Errorf("Unexpected signing method: %s", token.Header["alg"])
				}

				return a.key, nil
			})
			if err == nil && t.Valid {
				c.Set(a.config.StashKey, t.Claims)
				return nil
			}
		}
		return he
	}
}

func (a *jwtrsAuthentication) Authenticated(c EchoStasher) (bool, string) {
	stashed := c.Get(a.config.StashKey)
	if stashed != nil {
		if claims, ok := stashed.(map[string]interface{}); ok {
			if username, ok := claims[a.config.UsernameClaim].(string); ok {
				return ok, username
			}
		}
	}
	return false, ""
}

func init() {
	RegisterAuthenticationInterface(jwtrsAuthenticationName, func() AuthenticationInterface {
		return &jwtrsAuthentication{}
	})
}
