# m2m - Move from POP3S to Maildir

## Install
* `go install github.com/pepa65/m2m@latest`
* `cp -i .m2m.conf ~/` - If the default path is to be used
* Edit `~/.m2m.conf` for the POP3S server details and the Maildir location:
  - username: POP3S username
  - password: POP3S password
  - tlsservername: Server domainname according to the certificate
  - serveraddress: IP/Domainname address of the server
  - proxyaddress: Address of the proxy server (empty when not used)
  - disabletls: `true`/`false` - Disable TLS or not (default)
  - keep: `true`/`false` - Keep messages on the POP3S server or not (default)

## Run
* By default, `~/.m2m.conf` is used as the configuration file: `m2m`
* Multiple configuration files can be used by running with each one like: `m2m <conf_file>`
