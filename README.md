# m2m - Move from POP3S to Maildir

# Install
* `go install github.com/pepa65/m2m@latest`
* `cp -i .m2m.conf ~/`
* Edit `~/.m2m.conf` for the POP3S server details and the Maildir location:
  - Username: the POP3S username
  - Password: the POP3S password
  - TLSServerName: the server domainname according to the certificate
  - ServerAddress: the IP/Domainname address of the server
  - ProxyAddress: the address of the proxy server (empty when not used)
  - DisableTLS: `false`/`true` - disable TLS or not (default)
  - Keep: `false`/`true` keep messages on the POP3s server or not (default)

# Run
* `m2m`
