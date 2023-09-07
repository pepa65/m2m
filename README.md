[![Go Report Card](https://goreportcard.com/badge/github.com/pepa65/m2m)](https://goreportcard.com/report/github.com/pepa65/m2m)
[![GoDoc](https://godoc.org/github.com/pepa65/m2m?status.svg)](https://godoc.org/github.com/pepa65/m2m)
[![GitHub](https://img.shields.io/github/license/pepa65/m2m.svg)](LICENSE)
# m2m - Move from POP3 to Maildir

* **v1.5.3**
* Just pull mails from POP3 servers (TLS can be disabled) and put them in
  local Maildirs. Proxies can be used, onion can be used.
* RFC6856 compliant (UTF8 before anything) so works with Courier as well.
* It can keep mails on the server, but does not remember/store which have been seen.
* Based on github.com/unkaktus/mm

## Install
* `go install github.com/pepa65/m2m@latest`
* The directory `~/.m2m.conf` contains all the account config files which are checked in lexical order.
  The file name is the account name. (See the `Example` file in the repo).
* The config files have the POP3 server config details and the Maildir location:
  - `username`: POP3 username
  - `password`: POP3 password
  - `tlsdomain`: Server domainname according to the certificate
  - `server`: IP/Domainname of the server
  - `port`: Port (default: 995)
  - `proxyport`: IP/Domainname with port of the proxy server (`server:port`) (default: empty, not used)
  - `tls`: `true`/`false` - Use TLS (default) or not
  - `keep`: `true`/`false` - Keep mails on the POP3 server or delete them (default)
  - `maildir`: Path to the Maildir directory (default: `~/Maildir`)
* The config files (being YAML files) can have comments (starting with '#').

## Run
* Usage: `m2m [ -v|--verbose | -q|--quiet | -h|--help ]`
* Flag `-v`/`--verbose`: Output a more detailed log of actions.
* Flag `-q`/`--quiet`: Output only fatal errors.
* Default (no flag): Output a minimal report (nothing on no mails).
* All output is on `stderr`.

## Help
```
m2m v1.5.3 - Move from POP3 to Maildir
* Downloading emails from POP3 servers and moving them into Maildir folders.
* Repo:   github.com/pepa65/m2m
* Usage:  m2m [ -v|--verbose | -q|--quiet | -h|--help ]
    No flags:      Output a minimal report (no output on no mails)
    -v/--verbose:  Output a more detailed log of actions
    -q/--quiet:    Output only fatal errors.
    -h/--help:     Output this help text
* The directory `~/.m2m.conf` contains all the account config files, which
  are checked in lexical order, where the filename is the account name.
* Parameters in the configuration files:
    username:         POP3 username
    password:         POP3 password
    tlsdomain:        Server domainname according to the certificate
    server:           IP/Domainname of the server
    port:             Port (default: 995)
    proxyport:        Proxy server IP/Domainname:Port (default: not used)
    tls: true/false   Use TLS (default), or not
    keep: true/false  Keep mails on POP3 server, or delete them (default)
    maildir:          Path to the Maildir directory (default: `~/Maildir`)
```
