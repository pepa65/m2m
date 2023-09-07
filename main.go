// main.go - Move from POP3 to Maildir

package main

import (
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"golang.org/x/net/proxy"
	"gopkg.in/yaml.v2"
)

const version = "1.4.1"

type Config struct {
	Username         string
	Password         string
	TLSDomain        string
	Server           string
	Port             string
	ProxyAddressPort string
	TLS              bool
	Keep             bool
	MaildirPath      string
}

var (
	self = ""
	home = ""
)

func usage(msg string) { // I:self,version
	fmt.Fprint(os.Stderr, self+" v"+version+` - Move from POP3 to Maildir
* Downloading emails from POP3 servers and moving them into Maildir folders.
* Repo:   github.com/pepa65/m2m
* Usage:  m2m [ -v|--verbose | -q|--quiet | -h|--help ]
    No flags:      Output a minimal report (no output on no mails)
    -v/--verbose:  Output a more detailed log of actions
    -q/--quiet:    Output only fatal errors.
    -h/--help:     Output this help text
* The directory '~/.m2m.conf' contains all the account config files, which
  are checked in lexical order. Yhe filename is the account name.
* Parameters in the configuration files:
    username:          POP3 username
    password:          POP3 password
    tlsdomain:         Server domainname according to the certificate
    server:            IP/Domainname of the server
    port:              Port (default: 995)
    proxyaddressport:  Proxy server IP/Domainname:Port (default: not used)
    tls: true/false    Use TLS (default), or not
    keep: true/false   Keep mails on POP3 server, or delete them (default)
    maildirpath:       Path to the Maildir directory (default: '~/Maildir')
`)

	if msg != "" {
		fmt.Fprintf(os.Stderr, "\n%v\n", msg)
		os.Exit(1)
	}
	os.Exit(0)
}

func main() { // IO:self
	selfparts := strings.Split(os.Args[0], "/")
	self = selfparts[len(selfparts)-1]
	if len(os.Args) > 2 {
		usage("Only 1 (optional) argument allowed: -v/--verbose / -q/--quiet / -h/--help")
	}

	verbose := 1
	if len(os.Args) == 2 {
		if os.Args[1] == "-h" || os.Args[1] == "--help" {
			usage("")
		} else if os.Args[1] == "-v" || os.Args[1] == "--verbose" {
			verbose = 2
		} else if os.Args[1] == "-q" || os.Args[1] == "--quiet" {
			verbose = 0
		} else {
			usage("The only argument allowed is: -v/--verbose / -q/--quiet / -h/--help")
		}
	}

	var err error
	home, err = os.UserHomeDir()
	if err != nil {
		log.Fatal(err)
	}

	cfgpath := filepath.Join(home, ".m2m.conf")
	files, err := ioutil.ReadDir(cfgpath)
	if err != nil {
		log.Fatal(err)
	}

	start := time.Now()
	logline := ""
	nmsg := 0
	for _, file := range files {
		res, n := check(file.Name(), filepath.Join(cfgpath, file.Name()), verbose)
		logline += res
		nmsg += n
	}
	duration := time.Since(start).Seconds()
	if verbose == 1 && nmsg > 0 {
		now := time.Now().Format("2006-01-02_15:04:05")
		fmt.Fprintf(os.Stderr, "%s %s(%.3fs) ", now, logline, duration)
	} else if verbose == 2 {
		log.Printf("Running time: %fs", duration)
	}
}

func check(account string, filename string, verbose int) (string, int) {
	cfgdata, err := ioutil.ReadFile(filename)
	if err != nil {
		log.Fatal(err)
	}

	var cfg Config
	// Default values
	cfg.Port = "995"
	cfg.TLS = true
	err = yaml.UnmarshalStrict(cfgdata, &cfg)
	if err != nil {
		log.Fatalf("Error in config file '%s'\n%s", filename, err.Error())
	}

	var dialer Dialer
	dialer = &net.Dialer{}
	if cfg.ProxyAddressPort != "" {
		dialer, err = proxy.SOCKS5("tcp", cfg.ProxyAddressPort, nil, proxy.Direct)
		if err != nil {
			log.Fatal(err)
		}
	}

	var conn net.Conn
	conn, err = dialer.Dial("tcp", cfg.Server+":"+cfg.Port)
	if err != nil {
		log.Fatal(err)
	}

	if cfg.TLS {
		tlsConfig := &tls.Config{ServerName: cfg.TLSDomain}
		tlsConn := tls.Client(conn, tlsConfig)
		if err != nil {
			log.Fatal(err)
		}

		conn = tlsConn
	}

	buf := make([]byte, 255)
	n, err := conn.Read(buf)
	if err != nil {
		log.Fatal(err)
	}

	ok, msg, err := ParseResponseLine(string(buf[:n]))
	if err != nil {
		log.Fatal(err)
	}

	if !ok {
		log.Fatalf("Server error: %s", msg)
	}

	popConn := NewPOP3Conn(conn)
	line, _ := popConn.Cmd("UTF8")  // Ignore server error
	line, err = popConn.Cmd("USER %s", cfg.Username)
	if err != nil {
		log.Fatal(err)
	}

	line, err = popConn.Cmd("PASS %s", cfg.Password)
	if err != nil {
		log.Fatal(err)
	}

	line, err = popConn.Cmd("STAT")
	if err != nil {
		log.Fatal(err)
	}

	s := strings.Split(line, " ")
	if len(s) != 2 {
		log.Fatalf("STAT malformed: %s", line)
	}

	nmsg, err := strconv.Atoi(s[0])
	if err != nil {
		log.Fatal(err)
	}

	boxsize, err := strconv.Atoi(s[1])
	if err != nil {
		log.Fatal(err)
	}

	var logaccount string
	if verbose == 2 {
		log.Printf("Found %d messages of total size %d bytes", nmsg, boxsize)
	} else if verbose > 0 {
		logaccount = fmt.Sprintf("%s: %d ", account, nmsg)
	}
	for i := 1; i <= nmsg; i++ {
		line, data, err := popConn.CmdMulti("RETR %d", i)
		if err != nil {
			log.Fatal(err)
		}

		s := strings.SplitN(line, " ", 2)
		msgSize := "?"
		if _, err := strconv.Atoi(s[0]); err == nil {
			msgSize = s[0]
		}
		if verbose == 2 {
			log.Printf("Fetching message %d/%d (%s bytes)", i, nmsg, msgSize)
		}
		maildir := cfg.MaildirPath
		if maildir == "" {
			maildir = filepath.Join(home, "Maildir")
		}
		err = SaveToMaildir(cfg.MaildirPath, data)
		if err != nil {
			log.Fatal(err)
		}

		if !cfg.Keep {
			line, err = popConn.Cmd("DELE %d", i)
			if err != nil {
				log.Fatal(err)
			}
		}
	}

	if verbose == 2 && nmsg > 0 {
		if cfg.Keep {
			log.Printf("Not deleting messages from the server")
		} else {
			log.Printf("Deleted all messages from the server")
		}
	}
	line, err = popConn.Cmd("QUIT")
	if err != nil {
		log.Fatal(err)
	}

	conn.Close()
	return logaccount, nmsg
}
