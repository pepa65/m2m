[![Go Report Card](https://goreportcard.com/badge/github.com/pepa65/m2m)](https://goreportcard.com/report/github.com/pepa65/m2m)
[![GoDoc](https://godoc.org/github.com/pepa65/m2m?status.svg)](https://godoc.org/github.com/pepa65/m2m)
[![GitHub](https://img.shields.io/github/license/pepa65/m2m.svg)](LICENSE)
# m2m - Move from POP3 to Maildir

* **v1.7.1**
* Just pull mails from POP3 servers (TLS can be disabled) and put them in
  local Maildirs. Proxies and Onion entry servers are supported.
* RFC6856 compliant (UTF8 before anything) so works with Courier as well.
* It can keep mails on the server, but does not remember/store which have been seen.
* Expanded from github.com/unkaktus/mm

## Install
* `go install github.com/pepa65/m2m@latest`
* The directory `~/.m2m.conf` contains all the account config files which are checked in lexical order.
  The file name is the account name. (See the `Example` file in the repo).
* The config files have the POP3 server config details and the Maildir location:
  - `username`: POP3 username [mandatory]
  - `password`: POP3 password [mandatory]
  - `tlsdomain`: Server domainname (as in its certificate) [mandatory]
  - `port`: Port [default: 995]
  - `entryserver`: Initial IP/Domainname for the server [default: not used]
  - `proxyport`: Proxy server (`server:port`) [default: empty, not used]
  - `tls`: `true`/`false` - Use TLS [default] or not
  - `keep`: `true`/`false` - Keep mails on the POP3 server, or delete them [default]
  - `maildir`: Path to the Maildir directory [default: `~/Maildir`]
* The config files (being YAML files) can have comments (starting with '#').

## Run
* Usage: `m2m [ -h|--help | -q|--quiet ]`
* Flag `-h`/`--help` outputs just a help text.
* Flag `-q`/`--quiet` outputs only fatal errors to `stderr`.
* Normally, a minimal report is sent to `stdout` (nothing on no mails),
  and any additional verbose output is logged to `stderr`.

## Help
```
m2m v1.7.1 - Move from POP3 to Maildir
* Downloading emails from POP3 servers and moving them into Maildir folders.
* Repo:   github.com/pepa65/m2m
* Usage:  m2m [ -h|--help | -q|--quiet ]
    -h/--help:   Output this help text.
    -q/--quiet:  Output only on critical errors (to 'stderr').
    No flag:     A minimal report is sent to 'stdout' (nothing on no mails),
                 and any additional verbose output is logged to 'stderr'.
* The directory '~/.m2m.conf' contains all the account config files, which
  are checked in lexical order. The filename is the account name.
* Parameters in the configuration files:
    username:         POP3 username [mandatory]
    password:         POP3 password [mandatory]
    tlsdomain:        Server domainname (as in its certificate) [mandatory]
    port:             Port [default: 995]
    entryserver:      Initial IP/Domainname for the server [default: not used]
    proxyport:        Proxy server (server:port) [default: not used]
    tls: true/false   Use TLS [default], or not
    keep: true/false  Keep mails on POP3 server, or delete them [default]
    maildir:          Path to the Maildir directory [default: '~/Maildir']
```
