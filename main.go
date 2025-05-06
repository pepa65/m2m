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
	"sync"
	"time"

	"golang.org/x/net/proxy"
	"gopkg.in/yaml.v2"
)

const (
	version = "1.17.0"
	confdir = ".m2m.conf"
)

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
	Active      bool
}

type writer struct{}

var (
	self     = ""
	home     = ""
	accounts = make(map[string]string)
	wg       sync.WaitGroup
)

func usage(msg string) { // I:self,version
	fmt.Print(self + " v" + version + ` - Move from POP3 to Maildir
* Downloading emails from POP3 servers and moving them into Maildir folders.
* Repo:   github.com/pepa65/m2m
* Usage:  m2m [-s|--serial] [-q|--quiet] | [-h|--help]
    -s/--serial:  Check the accounts in order, do not check concurrently.
    -q/--quiet:   Output only on critical errors (on 'stderr').
    -h/--help:    Output this help text.
    If mails are found, a minimal report goes to 'stdout'; errors to 'stderr'.
* The directory '~/` + confdir + `' contains all account config files, which are
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
`)

	if msg != "" { // Abort message
		fmt.Fprintf(os.Stderr, "\n%v\n", msg)
		os.Exit(1)
	}
	os.Exit(0)
}

func (w writer) Write(bytes []byte) (int, error) {
	s := fmt.Sprint(time.Now().String()[:23] + " " + string(bytes))
	fmt.Fprint(os.Stderr, s)
	return len(s), nil
}

func main() { // I:accounts O:self,home IO:wg
	selfparts := strings.Split(os.Args[0], "/")
	self = selfparts[len(selfparts)-1]
	quiet, serial := false, false
	for i, arg := range os.Args {
		if i == 0 {
			continue
		}
		switch arg {
		case "-h", "--help":
			usage("")
		case "-q", "--quiet":
			quiet = true
		case "-s", "--serial":
			serial = true
		default:
			// Abort message
			usage("The only arguments allowed are: -s/--serial, -h/--help and -q/--quiet")
		}
	}

	log := log.New(new(writer), "- ", log.Lmsgprefix)
	var err error
	home, err = os.UserHomeDir()
	if err != nil { // Abort
		log.Fatal(err)
	}

	cfgpath := filepath.Join(home, confdir)
	dir, err := os.Open(cfgpath)
	if err != nil {
		log.Fatal(err)
	}

	files, err := dir.Readdirnames(0)
	if err != nil {
		log.Fatal(err)
	}

	start := time.Now()
	sort.Strings(files)
	for _, file := range files {
		if file[0:1] != "." {
			wg.Add(1)
			if serial {
				go check(file, cfgpath, quiet)
			} else {
				check(file, cfgpath, quiet)
			}
		}
	}
	wg.Wait()
	duration := time.Since(start).Seconds()
	if !quiet {
		mails := false
		logline := time.Now().Format("2006-01-02_15:04:05 ")
		for _, file := range files {
			n := accounts[file]
			if n != "" {
				logline += file + ": " + n + " "
				if n != "0" {
					mails = true
				}
			}
		}
		if mails {
			fmt.Printf("%s(%.3fs) ", logline, duration)
		}
	}
	if !quiet {
		log.Printf("Server checking time: %fs", duration)
	}
}

func unpanic() {
	recover()
}

func check(account string, m2mdir string, quiet bool) { // I:home O:accounts IO:wg
	defer unpanic()
	defer wg.Done()
	log := log.New(new(writer), account + ": ", log.Lmsgprefix)
	lockfile := filepath.Join(m2mdir, "." + account + "_locked")
	file, err := os.Open(lockfile)
	if err == nil { // Account locked: skip
		file.Close()
		log.Panic("Locked")
	}

	filename := filepath.Join(m2mdir, account)
	cfgdata, err := ioutil.ReadFile(filename)
	if err != nil { // Abort
		log.Panic(err)
	}

	var cfg Config
	// Default values
	cfg.Port = "995"
	cfg.TLS = true
	cfg.Maildir = filepath.Join(home, "Maildir")
	cfg.Active = true
	err = yaml.UnmarshalStrict(cfgdata, &cfg)
	if err != nil { // Abort
		log.Panic("Error in config file '" + filename + "'\n" + err.Error())
	}

	if !cfg.Active && !quiet {
		log.Panic("Inactive")
	}
	if cfg.Username == "" { // Abort
		log.Panic("Missing 'username' in configfile '" + filename + "'")
	}

	if cfg.TLSDomain == "" && cfg.TLS == true { // Abort
		log.Panic("Missing 'tlsdomain' in configfile '" + filename + "' while TLS required")
	}

	// Lock account before going online
	file, err = os.OpenFile(lockfile, os.O_CREATE, 0400)
	if err != nil {
		log.Panic("Cannot create lock file '" + lockfile + "'")
	}
	defer os.Remove(lockfile)
	defer file.Close()

	var dialer Dialer
	dialer = &net.Dialer{}
	if cfg.ProxyPort != "" {
		dialer, err = proxy.SOCKS5("tcp", cfg.ProxyPort, nil, proxy.Direct)
		if err != nil { // Abort
			log.Panic(err)
		}
	}

	var conn net.Conn
	if cfg.EntryServer != "" {
		conn, err = dialer.Dial("tcp", cfg.EntryServer+":"+cfg.Port)
	} else {
		conn, err = dialer.Dial("tcp", cfg.TLSDomain+":"+cfg.Port)
	}
	if err != nil { // Abort
		log.Panic(err)
	}

	defer conn.Close()
	if cfg.TLS {
		tlsConfig := &tls.Config{ServerName: cfg.TLSDomain}
		tlsConn := tls.Client(conn, tlsConfig)
		if err != nil { // Abort
			log.Panic(err)
		}

		conn = tlsConn
	}

	buf := make([]byte, 255)
	n, err := conn.Read(buf)
	if err != nil { // Abort
		log.Panic(err)
	}

	ok, msg, err := ParseResponseLine(string(buf[:n]))
	if err != nil { // Abort
		log.Panic(err)
	}

	if !ok { // Abort
		log.Panic("Server error: " + msg)
	}

	popConn := NewPOP3Conn(conn)
	popConn.Cmd("UTF8")
	line, err := popConn.Cmd("USER %s", cfg.Username)
	if err != nil { // Abort
		log.Panic(err)
	}

	line, err = popConn.Cmd("PASS %s", cfg.Password)
	if err != nil { // Abort
		log.Panic(err)
	}

	line, err = popConn.Cmd("STAT")
	if err != nil { // Abort
		log.Panic(err)
	}

	stat := strings.Split(line, " ")
	if len(stat) != 2 { // Abort
		log.Panic("STAT response malformed: " + line)
	}

	nmsg, err := strconv.Atoi(stat[0])
	if err == nil {
		accounts[account] = stat[0]
	} else { // Abort
		log.Panic("Malformed number of messages: " + stat[0])
	}

	boxsize, err := strconv.Atoi(stat[1])
	if err != nil { // Abort
		log.Panic("Malformed mailbox size: " + stat[1])
	}

	if !quiet {
		log.Printf("%d messages %d bytes", nmsg, boxsize)
	}
	delerrs := 0
	for i := 1; i <= nmsg; i++ {
		line, data, err := popConn.CmdMulti("RETR %d", i)
		if err != nil {
			log.Printf("Error retrieving message %d/%d: %s", i, nmsg, err.Error())
			continue
		}

		size, _, ok := strings.Cut(line, " ")
		if !ok && !quiet {
			log.Printf("RETR response malformed for message %d/%d: %s", i, nmsg, line)
		}
		_, err = strconv.Atoi(size)
		if err != nil && !quiet {
			log.Printf("Malformed size for message %d/%d: %s", i, nmsg, size)
			size = "?"
		}
		if !quiet {
			log.Printf("Fetched message %d/%d (%s bytes)", i, nmsg, size)
		}
		err = SaveToMaildir(cfg.Maildir, data)
		if err != nil {
			log.Printf("Error saving mesage %d/%d to maildir %s: %s", i, nmsg, cfg.Maildir, err.Error())
			continue
		}

		if !cfg.Keep {
			line, err = popConn.Cmd("DELE %d", i)
			if err != nil {
				delerrs += 1
				log.Printf("Error deleting mesage %d/%d from the server: %s", i, nmsg, err.Error())
			}
		}
	}
	if !quiet && nmsg > 0 {
		if cfg.Keep {
			log.Print("Not deleting messages from the server")
		} else {
			log.Printf("Messages deleted from the server: %d/%d", nmsg-delerrs, nmsg)
		}
	}
	popConn.Cmd("QUIT")
	conn.Close()
}
