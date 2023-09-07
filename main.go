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

const version = "1.6.0"

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
	fmt.Fprint(os.Stderr, self+" v"+version+` - Move from POP3 to Maildir
* Downloading emails from POP3 servers and moving them into Maildir folders.
* Repo:   github.com/pepa65/m2m
* Usage:  m2m [ -v|--verbose | -q|--quiet | -h|--help ]
    No flags:      Output a minimal report (no output on no mails)
    -v/--verbose:  Output a more detailed log of actions
    -q/--quiet:    Output only fatal errors.
    -h/--help:     Output this help text
* The directory '~/.m2m.conf' contains all the account config files, which
  are checked in lexical order. The filename is the account name.
* Parameters in the configuration files:
    username:         POP3 username
    password:         POP3 password
    tlsdomain:        Server domainname (according to its certificate)
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

	mails := false
	start := time.Now()
	for _, file := range files {
		n, errormsg := check(file.Name(), filepath.Join(cfgpath, file.Name()), verbose)
		if n > 0 {
			mails = true
		}
		if errormsg != "" {
			log.Print(errormsg)
		}
	}
	duration := time.Since(start).Seconds()
	if verbose == 1 && mails {
		logline := time.Now().Format("2006-01-02_15:04:05 ")
		for account, n := range accounts {
			logline += account+": "+n+" "
		}
		fmt.Fprintf(os.Stderr, "%s(%.3fs) ", logline, duration)
	} else if verbose == 2 {
		log.Printf("Running time: %fs", duration)
	}
}

func check(account string, filename string, verbose int) (int, string) {
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

	if verbose == 2 {
		log.Printf("%s: Found %d messages of total size %d bytes", account, nmsg, boxsize)
	} else if verbose == 1 {
		accounts[account] = stat[0]
	}
	for i := 1; i <= nmsg; i++ {
		line, data, err := popConn.CmdMulti("RETR %d", i)
		if err != nil {
			log.Printf("%s: Error retrieving message %d/%d: %s", account, i, nmsg, err.Error())
			continue
		}

		size, _, ok := strings.Cut(line, " ")
		if !ok && verbose == 2 {
			log.Printf("%s: RETR response malformed for message %d/%d: %s", account, i, nmsg, line)
		}
		_, err = strconv.Atoi(size)
		if err != nil && verbose == 2 {
			log.Printf("%s: Malformed size for message %d/%d: %s", account, i, nmsg, size)
			size = "?"
		}
		if verbose == 2 {
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
			} else if verbose == 2 {
				log.Print(account+": Deleted message %d/%d from server: "+line)
			}
		}
	}

	if verbose == 2 && nmsg > 0 {
		if cfg.Keep {
			log.Print(account+": Not deleting messages from the server")
		}
	}
	line, err = popConn.Cmd("QUIT")
	if err != nil {
		log.Print(account+": Error quitting from server: "+err.Error())
	} else if verbose == 2 {
		log.Print(account+": Quit from server: "+line)
	}

	conn.Close()
	return nmsg, ""
}
