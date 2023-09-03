// main.go - Move from POP3S to Maildir

package main

import (
	"crypto/tls"
	"io/ioutil"
	"log"
	"net"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"

	"golang.org/x/net/proxy"
	"gopkg.in/yaml.v2"
)

type Config struct {
	Username      string
	Password      string
	TLSServerName string
	ServerAddress string
	ProxyAddress  string
	DisableTLS    bool
	Keep          bool
	MaildirPath   string
}

func main() {
	usr, err := user.Current()
	if err != nil {
		log.Fatal(err)
	}

	cfgpath := filepath.Join(usr.HomeDir, ".m2m.conf")
	if len(os.Args) > 2 {
		log.Fatalf("Only 1 (optional) argument allowed: configfile")
	}
	if len(os.Args) == 2 {
		cfgpath = os.Args[1]
	}
	cfgdata, err := ioutil.ReadFile(cfgpath)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("Configuration: %s", cfgpath)
	var cfg Config
	err = yaml.UnmarshalStrict(cfgdata, &cfg)
	if err != nil {
		log.Fatal(err)
	}
	var dialer Dialer
	dialer = &net.Dialer{}
	if cfg.ProxyAddress != "" {
		var err error
		dialer, err = proxy.SOCKS5("tcp", cfg.ProxyAddress, nil, proxy.Direct)
		if err != nil {
			log.Fatal(err)
		}
	}

	var conn net.Conn
	conn, err = dialer.Dial("tcp", cfg.ServerAddress)
	if err != nil {
		log.Fatal(err)
	}

	if !cfg.DisableTLS {
		tlsConfig := &tls.Config{ServerName: cfg.TLSServerName}
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
		log.Fatalf("Server returned error: %s", msg)
	}

	popConn := NewPOP3Conn(conn)
	line, err := popConn.Cmd("USER %s", cfg.Username)
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
		log.Fatalf("Malformed STAT response: %s", line)
	}

	nmsg, err := strconv.Atoi(s[0])
	if err != nil {
		log.Fatal(err)
	}

	boxsize, err := strconv.Atoi(s[1])
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("There are %d messages of total size %d bytes", nmsg, boxsize)
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
		log.Printf("Fetching message %d/%d (%s bytes)", i, nmsg, msgSize)
		maildir := cfg.MaildirPath
		if maildir == "" {
			maildir = filepath.Join(usr.HomeDir, "Maildir")
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

	if nmsg > 0 {
		if cfg.Keep {
			log.Printf("Not deleting the messages from the server")
		} else {
			log.Printf("Deleted the messages from the server")
		}
	}
	line, err = popConn.Cmd("QUIT")
	if err != nil {
		log.Fatal(err)
	}

	conn.Close()
}
