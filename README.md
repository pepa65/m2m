# m2m - Move from POP3S to Maildir

## Install
* `go install github.com/pepa65/m2m@latest`
* An example config file is `.m2m.conf`. If it is put in `~/` it is used as
  the sole "Default" account to be used. Alternatively, `~/.m2m.conf` can be
  a directory with account config files (whose filename is taken to be the
  'account name') which are all used when run.
* The config files have the POP3S server config details and the Maildir location:
  - `username`: POP3S username
  - `password`: POP3S password
  - `tlsservername`: Server domainname according to the certificate
  - `serveraddress`: IP/Domainname address of the server
  - `proxyaddress`: Address of the proxy server (default: not used)
  - `disabletls`: `true`/`false` - Disable TLS or not (default)
  - `keep`: `true`/`false` - Keep messages on the POP3S server or not (default)
  - `maildirpath`: Path to the Maildir directory (default: `~/Maildir`)

## Run
* Usage: `m2m [ -v|--verbose | -q|--quiet ]`
* `-v`/`--verbose` gives more detailed output.
* `-q`/`--quiet` only outputs on fatal errors.
* All output is on `stderr`.
