[![Go Report Card](https://goreportcard.com/badge/github.com/pepa65/m2m)](https://goreportcard.com/report/github.com/pepa65/m2m)
[![GoDoc](https://godoc.org/github.com/pepa65/m2m?status.svg)](https://godoc.org/github.com/pepa65/m2m)
[![GitHub](https://img.shields.io/github/license/pepa65/m2m.svg)](LICENSE)
# m2m - Move from POP3 to Maildir

* **v1.17.2**
* License: GPLv3+
* Just pull mails from POP3 servers (TLS can be disabled) and put them in local Maildirs.
* Proxies and Onion entry servers are supported.
* Multiple accounts supported, which are accessed concurrently.
* RFC6856 compliant (UTF8 before anything) so works with Courier as well.
* It can keep mails on the server, but does not remember/store which mails have been seen.
* Expanded from github.com/unkaktus/mm

## Install
* `go install github.com/pepa65/m2m@latest`
* The directory `~/.m2m.conf` contains all the account config files which are checked concurrently.
  The file name is taken as the account name, must not start with a `.`! (See the `Example` file in the repo).
* The (YAML) config files have the POP3 server config details and the Maildir location with parameters:
  - `active`: `true`/`false` - Account is active [default] or not
  - `username`: POP3 username [mandatory]
  - `password`: POP3 password [mandatory]
  - `tlsdomain`: Server domainname (as in its certificate) [mandatory]
  - `port`: Port [default: 995]
  - `entryserver`: Initial IP/Domainname for the server [default: not used]
  - `proxyport`: Proxy server (`server:port`) [default: empty, not used]
  - `tls`: `true`/`false` - Use TLS [default] or not
  - `keep`: `true`/`false` - Keep mails on the POP3 server, or delete them [default]
  - `maildir`: Path under $HOME to Maildir [default: `Maildir`]
  - Default options are taken when the parameter is not specified.
  - Comments are lines starting with '#'.

## Run
* Usage: `m2m [-s|--serial] [-q|--quiet] | [-h|--help]`
* Normally, a minimal report is sent to `stdout` (nothing on no mails),
  and any additional verbose output is logged to `stderr`. Route this output as desired!
* Flag `-q`/`--quiet` outputs only fatal errors to `stderr`.
* Flag `-h`/`--help` outputs just a help text.
* While running, each account gets locked by creating a file `.ACCOUNT_locked` in directory
  `~/.m2m.conf` when it gets checked online.

## Help
```
m2m v1.17.2 - Move from POP3 to Maildir
* Downloading emails from POP3 servers and moving them into Maildir folders.
* Repo:   github.com/pepa65/m2m
* Usage:  m2m [-s|--serial] [-q|--quiet] | [-h|--help]
    -s/--serial:  Check the accounts in order, do not check concurrently.
    -q/--quiet:   Output only on critical errors (on 'stderr').
    -h/--help:    Output this help text.
    If mails are found, a minimal report goes to 'stdout'; errors to 'stderr'.
* The directory '~/.m2m.conf' contains all account config files, which are
        checked concurrently by default (each filename is taken as the account name).
  Lockfiles '.ACCOUNT_locked' get placed here when an account gets checked.
* Parameter names (lowercase!) in the configuration files:
    active: true/false  Account is active [default] or not
    username:           POP3 username [mandatory]
    password:           POP3 password [mandatory]
    tlsdomain:          Server domainname (as in its certificate) [mandatory]
    port:               Port [default: 995]
    entryserver:        Initial server IP/Domainname [default: not used]
    proxyport:          Proxy server (server:port) [default: not used]
    tls: true/false     Use TLS [default], or not
    keep: true/false    Keep mails on POP3 server, or delete them [default]
    maildir:            Path under $HOME to Maildir [default: 'Maildir']
```
