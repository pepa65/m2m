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

const version = "1.11.0"

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
* Usage:  m2m [ -h|--help | -q|--quiet ]
    -h/--help:   Output this help text.
    -q/--quiet:  Output only on critical errors (on 'stderr').
    No flag:     A minimal report is sent to 'stdout' (nothing on no mails),
                 and any additional verbose output is logged to 'stderr'.
* The directory '~/.m2m.conf' contains all the account config files, which
  are checked concurrently. The filename is taken as the account name.
* Parameters in the configuration files:
    active: true/false  Account is active [default] or not
    username:           POP3 username [mandatory]
    password:           POP3 password [mandatory]
    tlsdomain:          Server domainname (as in its certificate) [mandatory]
    port:               Port [default: 995]
    entryserver:        Initial IP/Domainname for the server [default: not used]
    proxyport:          Proxy server (server:port) [default: not used]
    tls: true/false     Use TLS [default], or not
    keep: true/false    Keep mails on POP3 server, or delete them [default]
    maildir:            Path to the Maildir directory [default: '~/Maildir']
`)

	if msg != "" { // Critical message
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
	if len(os.Args) > 2 { // Critical message
		usage("Only 1 (optional) argument allowed: -h/--help / -q/--quiet")
	}

	quiet := false
	if len(os.Args) == 2 {
		if os.Args[1] == "-h" || os.Args[1] == "--help" {
			usage("")
		} else if os.Args[1] == "-q" || os.Args[1] == "--quiet" {
			quiet = true
		} else { // Critical message
			usage("The only argument allowed is: -h/--help / -q/--quiet")
		}
	}

	log := log.New(new(writer), "- ", log.Lmsgprefix)
	var err error
	home, err = os.UserHomeDir()
	if err != nil { // Critical message
		log.Fatal(err)
	}

	cfgpath := filepath.Join(home, ".m2m.conf")
	dir, err := os.Open(cfgpath)
	if err != nil { // Critical message
		log.Fatal(err)
	}

	files, err := dir.Readdirnames(0)
	if err != nil { // Critical message
		log.Fatal(err)
	}

	start := time.Now()
	sort.Strings(files)
	for _, file := range files {
		wg.Add(1)
		go check(file, filepath.Join(cfgpath, file), quiet)
	}
	wg.Wait()
	duration := time.Since(start).Seconds()
	if !quiet {
		mails := false
		logline := time.Now().Format("2006-01-02_15:04:05 ")
		for _, file := range files {
			if n := accounts[file]; n != "" {
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

func check(account string, filename string, quiet bool) { // I:home O:accounts IO:wg
	defer unpanic()
	defer wg.Done()
	log := log.New(new(writer), account+": ", log.Lmsgprefix)
	file, err := os.Open(filename + "_blocked")
	if err == nil { // Account blocked: skip
		file.Close()
		return
	}
	// Block account
	file, err = os.OpenFile(filename + "_blocked", os.O_CREATE, 0400)
	if err != nil {
		log.Panic("Cannot write lock file '" + filename + "_blocked'")
	}
	defer os.Remove(filename + "_blocked")
	defer file.Close()

	cfgdata, err := ioutil.ReadFile(filename)
	if err != nil { // Critical message
		log.Panic(err)
	}

	var cfg Config
	// Default values
	cfg.Port = "995"
	cfg.TLS = true
	cfg.Maildir = filepath.Join(home, "Maildir")
	cfg.Active = true
	err = yaml.UnmarshalStrict(cfgdata, &cfg)
	if err != nil { // Critical message
		log.Panic("Error in config file '" + filename + "'\n" + err.Error())
	}

	if !cfg.Active && !quiet {
		log.Panic("Inactive")
	}
	if cfg.Username == "" { // Critical message
		log.Panic("Missing 'username' in configfile '" + filename + "'")
	}

	if cfg.TLSDomain == "" && cfg.TLS == true { // Critical message
		log.Panic("Missing 'tlsdomain' in configfile '" + filename + "' while TLS required")
	}

	var dialer Dialer
	dialer = &net.Dialer{}
	if cfg.ProxyPort != "" {
		dialer, err = proxy.SOCKS5("tcp", cfg.ProxyPort, nil, proxy.Direct)
		if err != nil { // Critical message
			log.Panic(err)
		}
	}

	var conn net.Conn
	if cfg.EntryServer != "" {
		conn, err = dialer.Dial("tcp", cfg.EntryServer+":"+cfg.Port)
	} else {
		conn, err = dialer.Dial("tcp", cfg.TLSDomain+":"+cfg.Port)
	}
	if err != nil { // Critical message
		log.Panic(err)
	}

	defer conn.Close()
	if cfg.TLS {
		tlsConfig := &tls.Config{ServerName: cfg.TLSDomain}
		tlsConn := tls.Client(conn, tlsConfig)
		if err != nil { // Critical message
			log.Panic(err)
		}

		conn = tlsConn
	}

	buf := make([]byte, 255)
	n, err := conn.Read(buf)
	if err != nil { // Critical message
		log.Panic(err)
	}

	ok, msg, err := ParseResponseLine(string(buf[:n]))
	if err != nil { // Critical message
		log.Panic(err)
	}

	if !ok { // Critical message
		log.Panic("Server error: " + msg)
	}

	popConn := NewPOP3Conn(conn)
	popConn.Cmd("UTF8")
	line, err := popConn.Cmd("USER %s", cfg.Username)
	if err != nil { // Critical message
		log.Panic(err)
	}

	line, err = popConn.Cmd("PASS %s", cfg.Password)
	if err != nil { // Critical message
		log.Panic(err)
	}

	line, err = popConn.Cmd("STAT")
	if err != nil { // Critical message
		log.Panic(err)
	}

	stat := strings.Split(line, " ")
	if len(stat) != 2 { // Critical message
		log.Panic("STAT response malformed: " + line)
	}

	nmsg, err := strconv.Atoi(stat[0])
	if err == nil {
		accounts[account] = stat[0]
	} else { // Critical message
		log.Panic("Malformed number of messages: " + stat[0])
	}

	boxsize, err := strconv.Atoi(stat[1])
	if err != nil { // Critical message
		log.Panic("Malformed mailbox size: " + stat[1])
	}

	if !quiet {
		log.Printf("%d messages %d bytes", nmsg, boxsize)
	}
	delerrs := 0
	for i := 1; i <= nmsg; i++ {
		line, data, err := popConn.CmdMulti("RETR %d", i)
		if err != nil { // Critical message
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
		if err != nil { // Critical message
			log.Printf("Error saving mesage %d/%d to maildir %s: %s", i, nmsg, cfg.Maildir, err.Error())
			continue
		}

		if !cfg.Keep {
			line, err = popConn.Cmd("DELE %d", i)
			if err != nil { // Critical message
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
}
