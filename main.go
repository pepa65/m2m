// main.go - Move from POP3 to Maildir

package main

import (
	"crypto/tls"
	"encoding/json"
	"flag"
	"io/ioutil"
	"log"
	"net"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"

	"golang.org/x/net/proxy"
)

type Config struct {
	Username      string
	Password      string
	MaildirPath   string
	TLSServerName string
	ServerAddress string
	ProxyAddress  string
	Keep          bool
	DisableTLS    bool
}

func main() {
	usr, err := user.Current()
	if err != nil {
		log.Fatal(err)
	}

	cfgpath := filepath.Join(usr.HomeDir, ".m2m.conf")
	flag.Parse()
	if len(flag.Args()) == 1 {
		cfgpath = flag.Args()[0]
	}
	var cfg Config
	cfgdata, err := ioutil.ReadFile(cfgpath)
	if err != nil {
		log.Fatal(err)
	}

	err = json.Unmarshal(cfgdata, &cfg)
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
	log.Printf("USER: \"%s\"\n", line)
	if err != nil {
		log.Fatal(err)
	}

	line, err = popConn.Cmd("PASS %s", cfg.Password)
	log.Printf("PASS: \"%s\"\n", line)
	if err != nil {
		log.Fatal(err)
	}

	line, err = popConn.Cmd("STAT")
	log.Printf("STAT: \"%s\"\n", line)
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
		err = SaveToMaildir(cfg.MaildirPath, data)
		if err != nil {
			log.Fatal(err)
		}

		if cfg.Keep {
			log.Printf("Not deleting the messages from the server")
		} else {
			line, err = popConn.Cmd("DELE %d", i)
			log.Printf("DELE: \"%s\"\n", line)
			if err != nil {
				log.Fatal(err)
			}
			log.Printf("Deleted the messages from the server")
		}
	}

	line, err = popConn.Cmd("QUIT")
	log.Printf("QUIT: \"%s\"", line)
	if err != nil {
		log.Fatal(err)
	}

	conn.Close()
}
