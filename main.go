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
	"crypto/tls"
	"encoding/json"
	"flag"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/labstack/echo"
	"github.com/tylerb/graceful"
)

type URL struct {
	*url.URL
}

func (u *URL) MarshalJSON() ([]byte, error) {
	if u.URL == nil {
		return json.Marshal(nil)
	}
	return json.Marshal(u.String())
}

func (u *URL) UnmarshalJSON(b []byte) error {
	s := ""
	err := json.Unmarshal(b, &s)
	if err != nil {
		return err
	}
	u.URL, err = url.Parse(s)
	return err
}

type EchoMiddlewareUser interface {
	Use(m ...echo.Middleware)
}

type EchoStasher interface {
	Set(string, interface{})
	Get(string) interface{}
}

type proxyData struct {
	TargetURL    *URL
	SetHeader    http.Header
	Enabled      bool
	Comment      string
	Who          string
	MaintainHost bool
	Expire       time.Time
	stop         chan bool
}

var config *configuration

var proxies = map[string]*http.Server{}
var metaData = map[string]*proxyData{}

func getData(ip string) *proxyData {
	data, ok := metaData[ip]
	if !ok {
		data = &proxyData{
			TargetURL: &URL{},
			SetHeader: make(http.Header),
		}
		metaData[ip] = data
	}
	return data
}

func clientIP(r *http.Request) string {
	rawClientIP, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		log.Printf("Unable to parse IP %q: %s", r.RemoteAddr, err)
		return "0.0.0.0"
	}
	clientIP := net.ParseIP(rawClientIP)
	return clientIP.String()
}

func proxyDownInterface(ip string) (e *echo.Echo) {
	data := getData(ip)
	data.Enabled = false
	if data.stop != nil {
		close(data.stop)
	}
	data.stop = nil

	e = echo.New()
	e.Any("/*", func(c *echo.Context) error {
		r := c.Request()
		clientIP := clientIP(r)
		log.Printf("[%s] %s %s %s disabled", ip, clientIP, r.Host, r.URL.String())

		return c.String(http.StatusOK, "Proxy disabled")
	})
	return
}

func singleJoiningSlash(a, b string) string {
	aslash := strings.HasSuffix(a, "/")
	bslash := strings.HasPrefix(b, "/")
	switch {
	case aslash && bslash:
		return a + b[1:]
	case !aslash && !bslash:
		return a + "/" + b
	}
	return a + b
}

func proxyUpInterface(ip string) *httputil.ReverseProxy {
	data := getData(ip)
	data.Enabled = true
	if data.stop != nil {
		close(data.stop)
	}
	data.stop = make(chan bool)
	go func() {
		select {
		case <-time.After(data.Expire.Sub(time.Now())):
			log.Println("Shutting down proxy interface on", ip)
			proxies[ip].Handler = proxyDownInterface(ip)
		case <-data.stop:
			data.stop = nil
		}
	}()

	return &httputil.ReverseProxy{
		Director: func(r *http.Request) {
			data := getData(ip)
			clientIP := clientIP(r)
			originalRequest := r.URL
			targetQuery := data.TargetURL.RawQuery

			r.URL.Scheme = data.TargetURL.Scheme
			r.URL.Host = data.TargetURL.Host
			r.URL.Path = singleJoiningSlash(data.TargetURL.Path, r.URL.Path)
			if targetQuery == "" || r.URL.RawQuery == "" {
				r.URL.RawQuery = targetQuery + r.URL.RawQuery
			} else {
				r.URL.RawQuery = targetQuery + "&" + r.URL.RawQuery
			}

			log.Printf("[%s] %s %s %s > %s", ip, clientIP, r.Host, originalRequest.String(), r.URL.String())

			forwardedFor := r.Header.Get(echo.XForwardedFor)
			if forwardedFor != "" {
				forwardedFor += ", " + clientIP
			} else {
				forwardedFor = clientIP
			}

			r.Header.Add("X-Remote-Addr", r.RemoteAddr)
			r.Header.Add("X-Real-IP", clientIP)
			r.Header.Add("X-Forwarded-For", forwardedFor)

			for name, val := range data.SetHeader {
				r.Header[name] = val
			}

			if !data.MaintainHost {
				r.Host = r.URL.Host
			}
		},
	}
}

func saveState() error {
	writer, err := os.Create(config.StateSaver.File)
	if err != nil {
		log.Println("Unable to save state:", err)
		return err
	}
	defer writer.Close()
	encoder := json.NewEncoder(writer)
	return encoder.Encode(metaData)
}

func loadState() error {
	reader, err := os.Open(config.StateSaver.File)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer reader.Close()
	decoder := json.NewDecoder(reader)
	if err = decoder.Decode(&metaData); err != nil {
		return err
	}

	for ip, data := range metaData {
		proxy, ok := proxies[ip]
		if !ok || proxy == nil {
			log.Printf("\tInterface %s doesn't exist, ignoring state", ip)
			continue
		}
		log.Println("\tRestoring state for", ip)

		if data.Enabled && data.Expire.After(time.Now()) {
			proxy.Handler = proxyUpInterface(ip)
		}
	}

	return nil
}

func main() {
	configFile := flag.String("config", "config.toml", "Configuration file")
	flag.Parse()

	log.Println("Loading configuration file")

	var err error
	config, err = loadConfiguration(*configFile)
	if err != nil {
		log.Printf("Problem parsing configuration file %s: %s\n", *configFile, err)
		return
	}

	log.Println("Initializing TLS configuration")
	cer, err := tls.X509KeyPair(config.TLS.Certificate, config.TLS.Key)
	tlsConfig := &tls.Config{Certificates: []tls.Certificate{cer}}

	// Todo, make proxy interfaces an object, add mutex

	log.Println("Binding proxy interfaces")
	errors := false
	for ip := range config.Addresses {
		listener, err := tls.Listen("tcp", ip+":443", tlsConfig)
		log.Println("\tBinding", ip, ":443")
		if err != nil {
			log.Println(err)
			errors = true
		}
		proxies[ip] = &http.Server{
			Handler: proxyDownInterface(ip),
		}

		go proxies[ip].Serve(listener)
	}

	if errors {
		log.Fatal("Please fix the above errors")
	}

	if config.StateSaver.Enabled && config.StateSaver.File != "" && config.StateSaver.Interval != nil {
		log.Println("Enabling statesaver")
		if err := loadState(); err != nil {
			log.Println("Unable to load state")
			log.Fatal(err)
		}
		go func() {
			for range time.Tick(config.StateSaver.Interval.Duration) {
				saveState()
			}
		}()
	} else {
		config.StateSaver.Enabled = false
	}

	graceful.ListenAndServe(adminInterface().Server(config.Listen), 1*time.Second)

	if config.StateSaver.Enabled {
		saveState()
	}
}
