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
	"log"
	"net/http"
	"time"

	"github.com/GeertJohan/go.rice"
	"github.com/labstack/echo"
	mw "github.com/labstack/echo/middleware"
	"github.com/thoas/stats"
)

func adminInterface() (e *echo.Echo) {
	log.Println("Administration interface starting")

	assetHandler := http.FileServer(rice.MustFindBox("public").HTTPBox())

	e = echo.New()
	s := stats.New()

	e.Use(mw.Logger())
	e.Use(mw.Recover())
	e.Use(s.Handler)

	log.Println("Configuring authentication interface")
	var authInterface AuthenticationInterface
	if config.AuthenticationMethod != "" {
		authInterface = GetAuthenticationInterface(config.AuthenticationMethod)
		if authInterface != nil {
			err := authInterface.Init(config, e)
			if err != nil {
				log.Println("Unable to initialize authentication module")
				log.Fatal(err)
			}
		}
	}

	log.Println("Configuring accesscontrol interface")
	var accessControlInterface AccessControlInterface
	if config.AccessControlMethod != "" {
		accessControlInterface = GetAccessControlInterface(config.AccessControlMethod)
		if accessControlInterface != nil {
			err := accessControlInterface.Init(config, authInterface, e)
			if err != nil {
				log.Println("Unable to initialize access control module")
				log.Fatal(err)
			}
		}
	}

	e.SetHTTPErrorHandler(func(err error, c *echo.Context) {
		code := http.StatusInternalServerError
		msg := http.StatusText(code)
		if he, ok := err.(*echo.HTTPError); ok {
			code = he.Code()
			msg = he.Error()
		}
		switch code {
		case 404:
			msg = "404 page not found"
		}

		if e.Debug() {
			msg = err.Error()
		}

		if !c.Response().Committed() {
			http.Error(c.Response(), msg, code)
		}
	})

	e.Get("/*", func(c *echo.Context) error {
		assetHandler.ServeHTTP(c.Response().Writer(), c.Request())
		return nil
	})

	e.Get("/interfaces", func(c *echo.Context) error {
		return c.JSON(http.StatusOK, config.Addresses)
	})

	e.Get("/stats", func(c *echo.Context) error {
		return c.JSON(http.StatusOK, s.Data())
	})

	g := e.Group("/proxy")
	g.Use(func(c *echo.Context) error {
		if _, ok := metaData[c.Param("ip")]; !ok {
			return echo.NewHTTPError(http.StatusNotFound)
		}
		return nil
	})

	g.Get("/:ip", func(c *echo.Context) error {
		data := getData(c.Param("ip"))
		return c.JSON(http.StatusOK, data)
	})

	g.Post("/:ip", func(c *echo.Context) error {
		data := getData(c.Param("ip"))
		data.SetHeader = http.Header{}
		c.Bind(data)
		return c.JSON(http.StatusOK, data)
	})

	/* - Simplified the api a bit... might revisit

	g.Post("/:ip/setheader", func(c *echo.Context) error {
		data := getData(c.Param("ip"))
		c.Bind(data.SetHeader)
		return c.JSON(http.StatusOK, data)
	})

	g.Post("/:ip/targeturl", func(c *echo.Context) error {
		data := getData(c.Param("ip"))
		c.Bind(data.TargetURL)
		return c.JSON(http.StatusOK, data)
	})

	g.Post("/:ip/maintainhost", func(c *echo.Context) error {
		data := getData(c.Param("ip"))
		c.Bind(data.MaintainHost)
		return c.JSON(http.StatusOK, data)
	})

	g.Post("/:ip/comment", func(c *echo.Context) error {
		data := getData(c.Param("ip"))
		c.Bind(data.Comment)
		return c.JSON(http.StatusOK, data)
	}) */

	g.Post("/:ip/enable", func(c *echo.Context) error {
		ip := c.Param("ip")
		data := getData(ip)
		previous := data.Enabled
		c.Bind(&data.Enabled)

		updateProxy := func() {
			data.Expire = time.Now().Add(config.MaxTTL.Duration)
			if authInterface != nil {
				_, user := authInterface.Authenticated(c)
				data.Who = user
			}
		}

		if previous != data.Enabled {
			if data.Enabled {
				proxies[ip].Handler = proxyUpInterface(ip)
				updateProxy()
			} else {
				proxies[ip].Handler = proxyDownInterface(ip)
			}
		} else if data.Enabled {
			updateProxy()
		}
		return c.JSON(http.StatusOK, data)
	})
	return
}
