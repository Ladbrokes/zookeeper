# zookeeper

`zookeeper`  Multi-interface proxy for those times when developers need public IPs

We've all been there right? Working away keeping the infrustructure from collapsing in around us and someone suddenly needs an ip forwarded to their machine for demoing or testing some third party callback system

`zookeeper` will allow port 443 to be reverse proxied, just like [nginx](http://nginx.org/), but with bonus [magic](https://s-media-cache-ak0.pinimg.com/736x/b8/b4/da/b8b4da721decc5b5f6149f4338657dad.jpg)

## Features

 * Reverse proxy
 * Proxy header injection/overwriting
 * Preserve or overwrite the host header
 * Supports admin authentication via a JWT in the header
 * Support static authentication as a set username, useful for testing LDAP config
 * Supports super basic LDAP access control
 * Proxies automatically disable themselves at a set timelimit to prevent them from being forgotten about with potentially buggy code left unattended on the intertubes
 * Only supports https/tls connections (might not be a feature for you)

## Future Features

 * More flexible access control
 * Basic authentication
 * Websocket updating of the UI
 * ~~Button to click to extend timeout without "disable/enabling"~~
 * Web interface for address configuration (Add more ips on the fly)
 * Per interface tls configuration
 * Tests
 * Documentation
 * Prettier disable screen
 * Better management of the proxy metadata and proxies, including mutexing

## Building

	cd $GOPATH
	go get github.com/Ladbrokes/zookeeper

### Building bonus - Use [Rice](https://github.com/GeertJohan/go.rice)!

	go get github.com/GeertJohan/go.rice/rice
	./bin/rice --import-path=github.com/Ladbrokes/zookeeper embed-go
	rm bin/zookeeper
	go install github.com/Ladbrokes/zookeeper

Now it's completely portable, all you need is a config and your PEM files (which you can inline into the config)

## Configuration

Configuration is in [TOML format](https://github.com/toml-lang/toml)

	listen = "10.37.12.203:8080"

	# What method of authentication do you wish to support
	# Can be jwt-rs, static, or empty
	authentication_method = "jwt-rs"

	# What method of access control do you wish to support
	# Can be ldap or empty
	accesscontrol_method = "ldap"

	# Shut off the proxy at this timeout
	max_ttl = "24h"

	# Configuration for the jwt-rs authentication module
	[authentication.jwt-rs]
	# header = "X-User-Authenticate"
	# username_claim = "user"
	# stash_key = claims
	# public = "file://path/to/public.key"
	# or
	public = """
	-----BEGIN PUBLIC KEY-----
	MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEA1YeAEjASE2sFwMd/cpiU
	...
	PwhTMMPp3BnHPcLp8NyTex3kh82u5OIxNm49BXaX2N3YA+m63iqwioWs10+xweEk
	xwIDAQAB
	-----END PUBLIC KEY-----
	"""

	# Configuration for the static authentication module
	[authentication.static]
	username = "shannon.wynter"

	# Configuration for the ldap access control module
	[accesscontrol.ldap]
	address = "ad.example.com:389"
	basedn = "OU=Employees,DC=ad,DC=example,DC=com"
	bind_username = "read-only@ad.example.com"
	bind_password = "my realy real password"
	search_template = """
	(&
	    (sAMAccountName={{.Username}})
	    (objectCategory=CN=Person,CN=Schema,CN=Configuration,DC=ad,DC=example,DC=com)
	    (|
	        (memberOf=CN=team-lead,OU=Atlassian,OU=Groups,DC=ad,DC=example,DC=com)
	        (memberOf=CN=systems,OU=Atlassian,OU=Groups,DC=ad,DC=example,DC=com)
	    )
	)
	"""

	# Save the state periodically
	[statesaver]
	enabled = true
	file = ".state"
	interval = "10m"

	# All incomming requests will be served with this certificate, probably best to make it a wildcard :D
	[tls]
	certificate = "file://magic.crt"
	key = "file://magic.key"

	# Configure the incomming addresses to listen on
	[address."10.37.1.190"]
	Description = "whatever you want really"

	[address."10.37.1.191"]
	Description = "I use the real world ip"


## License

Copyright (c) 2015 Shannon Wynter, Ladbrokes Digital Australia Pty Ltd. Licensed under GPL2. See the [LICENSE.md](LICENSE.md) file for a copy of the license.
