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
	"sort"
	"strconv"
	"strings"
	"time"

	"golang.org/x/net/proxy"
	"gopkg.in/yaml.v2"
)

const version = "1.7.1"

type Config struct {
	Username    string
	Password    string
	TLSDomain   string
	Port        string
	EntryServer string
	ProxyPort   string
	TLS         bool
	Keep        bool
	Maildir     string
}

var (
	self = ""
	home = ""
	accounts = make(map[string]string)
)

func usage(msg string) { // I:self,version
	fmt.Print(self+" v"+version+` - Move from POP3 to Maildir
* Downloading emails from POP3 servers and moving them into Maildir folders.
* Repo:   github.com/pepa65/m2m
* Usage:  m2m [ -h|--help | -q|--quiet ]
    -h/--help:   Output this help text.
    -q/--quiet:  Output only on critical errors (on 'stderr').
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
		usage("Only 1 (optional) argument allowed: -h/--help / -q/--quiet")
	}

	quiet := false
	if len(os.Args) == 2 {
		if os.Args[1] == "-h" || os.Args[1] == "--help" {
			usage("")
		} else if os.Args[1] == "-q" || os.Args[1] == "--quiet" {
			quiet = true
		} else {
			usage("The only argument allowed is: -h/--help / -q/--quiet")
		}
	}

	var err error
	home, err = os.UserHomeDir()
	if err != nil {
		log.Fatal(err)
	}

	cfgpath := filepath.Join(home, ".m2m.conf")
	//files, err := ioutil.ReadDir(cfgpath)
	dir, err := os.Open(cfgpath)
	if err != nil {
		log.Fatal(err)
	}

	files, err := dir.Readdirnames(0)
	if err != nil {
		log.Fatal(err)
	}

	mails := false
	start := time.Now()
	sort.Strings(files)
	for _, file := range files {
		n, errormsg := check(file, filepath.Join(cfgpath, file), quiet)
		if n > 0 {
			mails = true
		}
		if errormsg != "" && !quiet {
			log.Print(errormsg)
		}
	}
	duration := time.Since(start).Seconds()
	if !quiet && mails {
		logline := time.Now().Format("2006-01-02_15:04:05 ")
		for _, account := range files {
			logline += account+": "+accounts[account]+" "
		}
		fmt.Printf("%s(%.3fs) ", logline, duration)
	}
	if !quiet {
		log.Printf("Running time: %fs", duration)
	}
}

func check(account string, filename string, quiet bool) (int, string) {
	cfgdata, err := ioutil.ReadFile(filename)
	if err != nil {
		return 0, account+": "+err.Error()
	}

	var cfg Config
	// Default values
	cfg.Port = "995"
	cfg.TLS = true
	cfg.Maildir = filepath.Join(home, "Maildir")
	err = yaml.UnmarshalStrict(cfgdata, &cfg)
	if err != nil {
		return 0, account+": Error in config file '"+filename+"'\n"+err.Error()
	}

	if cfg.Username == "" {
		return 0, account+": Missing 'username' in configfile '"+filename+"'"
	}

	if cfg.TLSDomain == "" && cfg.TLS == true {
		return 0, account+": Missing 'tlsdomain' in configfile '"+filename+"' while TLS required"
	}

	var dialer Dialer
	dialer = &net.Dialer{}
	if cfg.ProxyPort != "" {
		dialer, err = proxy.SOCKS5("tcp", cfg.ProxyPort, nil, proxy.Direct)
		if err != nil {
			return 0, account+": "+err.Error()
		}
	}

	var conn net.Conn
	if cfg.EntryServer != "" {
		conn, err = dialer.Dial("tcp", cfg.EntryServer+":"+cfg.Port)
	} else {
		conn, err = dialer.Dial("tcp", cfg.TLSDomain+":"+cfg.Port)
	}
	if err != nil {
		return 0, account+": "+err.Error()
	}

	if cfg.TLS {
		tlsConfig := &tls.Config{ServerName: cfg.TLSDomain}
		tlsConn := tls.Client(conn, tlsConfig)
		if err != nil {
			return 0, account+": "+err.Error()
		}

		conn = tlsConn
	}

	buf := make([]byte, 255)
	n, err := conn.Read(buf)
	if err != nil {
		return 0, account+": "+err.Error()
	}

	ok, msg, err := ParseResponseLine(string(buf[:n]))
	if err != nil {
		return 0, account+": "+err.Error()
	}

	if !ok {
		return 0, account+": Server error: "+msg
	}

	popConn := NewPOP3Conn(conn)
	line, _ := popConn.Cmd("UTF8")  // Ignore any server error
	line, err = popConn.Cmd("USER %s", cfg.Username)
	if err != nil {
		return 0, account+": "+err.Error()
	}

	line, err = popConn.Cmd("PASS %s", cfg.Password)
	if err != nil {
		return 0, account+": "+err.Error()
	}

	line, err = popConn.Cmd("STAT")
	if err != nil {
		return 0, account+": "+err.Error()
	}

	stat := strings.Split(line, " ")
	if len(stat) != 2 {
		return 0, account+": "+"STAT response malformed: "+line
	}

	nmsg, err := strconv.Atoi(stat[0])
	if err != nil {
		return 0, account+": "+"Malformed number of messages: "+stat[0]
	}

	boxsize, err := strconv.Atoi(stat[1])
	if err != nil {
		return 0, account+": "+"Malformed mailbox size: "+stat[1]
	}

	if !quiet {
		log.Printf("%s: Found %d messages of total size %d bytes", account, nmsg, boxsize)
		accounts[account] = stat[0]
	}
	for i := 1; i <= nmsg; i++ {
		line, data, err := popConn.CmdMulti("RETR %d", i)
		if err != nil {
			log.Printf("%s: Error retrieving message %d/%d: %s", account, i, nmsg, err.Error())
			continue
		}

		size, _, ok := strings.Cut(line, " ")
		if !ok && !quiet {
			log.Printf("%s: RETR response malformed for message %d/%d: %s", account, i, nmsg, line)
		}
		_, err = strconv.Atoi(size)
		if err != nil && !quiet {
			log.Printf("%s: Malformed size for message %d/%d: %s", account, i, nmsg, size)
			size = "?"
		}
		if !quiet {
			log.Printf("%s: Fetched message %d/%d (%s bytes)", account, i, nmsg, size)
		}
		err = SaveToMaildir(cfg.Maildir, data)
		if err != nil {
			log.Printf("%s: Error saving mesage %d/%d to maildir %s: %s", account, i, nmsg, cfg.Maildir, err.Error())
			continue
		}

		if !cfg.Keep {
			line, err = popConn.Cmd("DELE %d", i)
			if err != nil {
				log.Printf("%s: Error deleting mesage %d/%d from server: %s", account, i, nmsg, err.Error())
			} else if !quiet {
				log.Print(account+": Deleted message %d/%d from server: "+line)
			}
		}
	}

	if !quiet && nmsg > 0 {
		if cfg.Keep {
			log.Print(account+": Not deleting messages from the server")
		}
	}
	popConn.Cmd("QUIT")
	conn.Close()
	return nmsg, ""
}
