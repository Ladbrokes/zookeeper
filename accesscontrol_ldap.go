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
	"bytes"
	"errors"
	"log"
	"net/http"
	"regexp"
	"text/template"

	"github.com/labstack/echo"
	"gopkg.in/ldap.v2"
)

const ldapAccessControlName = "ldap"

type ldapAccessControl struct {
	enabled                bool
	config                 ldapAccessControlConfiguration
	conn                   *ldap.Conn
	compiledSearchTemplate *template.Template
	authenticationProvider AuthenticationInterface
}

type ldapAccessControlConfiguration struct {
	Address        string `toml:"address"`
	UseTLS         bool   `toml:"usetls"`
	UseSSL         bool   `toml:"usessl"`
	BaseDN         string `toml:"basedn"`
	BindUsername   string `toml:"bind_username"`
	BindPassword   string `toml:"bind_password"`
	SearchTemplate string `toml:"search_template"`
}

// It's nice to write pretty filters but something in go/ldap doesn't seem to like all the whitespace/newlines
var ldapAccessControlFilterRegexp = regexp.MustCompile(`\s*\n\s*|\s*(\()\s*(.+?)\s*(\))\s*`)

func (l *ldapAccessControl) Init(c *configuration, a AuthenticationInterface, e EchoMiddlewareUser) (err error) {
	log.Println("ldapAccessControl initializing")

	if a == nil {
		log.Println("\tNo AuthenticationInterface found, disabling")
		return nil
	}

	if err = c.UnifyAccessControlConfiguration(ldapAccessControlName, &l.config); err != nil {
		log.Println("\tUnable to load configuration")
		return
	}

	log.Println("\tVerifying configuration")
	if l.config.UseSSL && l.config.UseTLS {
		return errors.New("usetls or usessl")
	}

	configured := l.config.Address != ""
	configured = configured && l.config.BaseDN != ""
	configured = configured && l.config.SearchTemplate != ""
	configured = configured && l.config.BindUsername != ""
	configured = configured && l.config.BindPassword != ""

	if !configured {
		log.Println("ldapAccessControl not configured")
		return errors.New("insufficient configuration")
	}

	searchFilter := ldapAccessControlFilterRegexp.ReplaceAllString(l.config.SearchTemplate, "$1$2$3")

	log.Println("\tCompiling search template")
	if l.compiledSearchTemplate, err = template.New("filter").Parse(searchFilter); err != nil {
		return
	}

	l.authenticationProvider = a
	l.connect()

	if e != nil {
		log.Println("\tRegistering middleware")
		e.Use(l.middleware())
	}

	l.enabled = true
	log.Println("ldapAccessControl ready")
	return
}

func (l *ldapAccessControl) connect() (err error) {
	l.conn, err = ldap.Dial("tcp", l.config.Address)
	if err != nil {
		log.Println("ldapAccessControl: Unable to connect")
		log.Println(err)
		return
	}
	return l.conn.Bind(l.config.BindUsername, l.config.BindPassword)
}

func (l *ldapAccessControl) middleware() echo.HandlerFunc {

	return func(c *echo.Context) error {
		if !l.enabled {
			return nil
		}

		// Skip WebSocket requests
		if (c.Request().Header.Get(echo.Upgrade)) == echo.WebSocket {
			return nil
		}

		if c.Request().Method == "POST" {
			return l.Can(c)
		}

		return nil
	}
}

func (l *ldapAccessControl) Can(c EchoStasher) error {
	// todo? add individual permissions?
	if ok, user := l.authenticationProvider.Authenticated(c); ok {
		var searchFilterBuffer bytes.Buffer
		if err := l.compiledSearchTemplate.Execute(&searchFilterBuffer, struct{ Username string }{Username: user}); err != nil {
			log.Println("ldapAccessControl: Unable to execute search filter template")
			log.Println(err)
			return echo.NewHTTPError(http.StatusInternalServerError, "Unable to execute search filter template")
		}

		searchRequest := ldap.NewSearchRequest(
			l.config.BaseDN,
			ldap.ScopeWholeSubtree,
			ldap.DerefAlways, 0, 0, false,
			searchFilterBuffer.String(),
			[]string{"sAMAccountName"},
			nil,
		)

		searchResult, err := l.conn.Search(searchRequest)
		if err != nil {
			log.Println("ldapAccessControl: Unable to execute LDAP search")
			log.Println(err)
			return echo.NewHTTPError(http.StatusInternalServerError, "Unable to execute LDAP search")
		}

		if len(searchResult.Entries) == 1 {
			return nil
		}
	}

	return echo.NewHTTPError(http.StatusUnauthorized)
}

func init() {
	RegisterAccessControlInterface(ldapAccessControlName, func() AccessControlInterface {
		return &ldapAccessControl{}
	})
}
