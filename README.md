[![Go Report Card](https://goreportcard.com/badge/github.com/pepa65/m2m)](https://goreportcard.com/report/github.com/pepa65/m2m)
[![GoDoc](https://godoc.org/github.com/pepa65/m2m?status.svg)](https://godoc.org/github.com/pepa65/m2m)
[![GitHub](https://img.shields.io/github/license/pepa65/m2m.svg)](LICENSE)
# m2m - Move from POP3S to Maildir

* v1.1.1
* Just pull mails from POP3S servers (TLS can be disabled) and put them in
  local Maildirs. Proxies can be used, onion can be used.
* RFC6856 compliant (UTF8 before RETR) so works with Courier as well.
* Based on github.com/unkaktus/mm

## Install
* `go install github.com/pepa65/m2m@latest`
* An example config file is `.m2m.conf`. If it is put in `~/` it is used as
  the sole "Default" account to be used. Alternatively, `~/.m2m.conf` can be
  a directory with account config files (whose filename is taken to be the
  'account name') which are all used when run in lexical order.
* The config files have the POP3S server config details and the Maildir location:
  - `username`: POP3S username
  - `password`: POP3S password
  - `tlsservername`: Server domainname according to the certificate
  - `serveraddress`: IP/Domainname address of the server
  - `proxyaddress`: Address of the proxy server (default: empty, not used)
  - `tls`: `true`/`false` - Use TLS (default) or not
  - `keep`: `true`/`false` - Keep messages on the POP3S server or delete them (default)
  - `maildirpath`: Path to the Maildir directory (default: `~/Maildir`)

## Run
* Usage: `m2m [ -v|--verbose | -q|--quiet ]`
* Flag `-v`/`--verbose` gives more detailed output.
* Flag `-q`/`--quiet` only outputs on fatal errors.
* All output is on `stderr`.
