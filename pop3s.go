// pop3s.go

package main

import (
	"fmt"
	"io"
	"net"
	"net/textproto"
	"strings"
)

type POP3Conn struct {
	conn *textproto.Conn
}

type Dialer interface {
	Dial(network, address string) (net.Conn, error)
}

func ParseResponseLine(input string) (ok bool, msg string, err error) {
	s := strings.SplitN(input, " ", 2)
	switch s[0] {
	case "+OK":
		ok = true
	case "-ERR":
		ok = false
	default:
		return false, "", fmt.Errorf("Malformed response status: %s", s[0])
	}

	if len(s) == 2 {
		msg = s[1]
	}
	return ok, msg, nil
}

func (pc *POP3Conn) Cmd(format string, args ...interface{}) (string, error) {
	c := pc.conn
	id, err := c.Cmd(format, args...)
	if err != nil {
		return "", err
	}

	c.StartResponse(id)
	defer c.EndResponse(id)
	line, err := c.ReadLine()
	if err != nil {
		return "", nil
	}

	ok, rmsg, err := ParseResponseLine(line)
	if err != nil {
		return "", err
	}

	if !ok {
		return "", fmt.Errorf("%s", rmsg)
	}

	return rmsg, nil
}

func (pc *POP3Conn) CmdMulti(format string, args ...interface{}) (string, []byte, error) {
	c := pc.conn
	id, err := c.Cmd(format, args...)
	if err != nil {
		return "", nil, err
	}

	c.StartResponse(id)
	defer c.EndResponse(id)
	line, err := c.ReadLine()
	if err != nil {
		return "", nil, err
	}

	ok, rmsg, err := ParseResponseLine(line)
	if err != nil {
		return "", nil, err
	}

	if !ok {
		return "", nil, fmt.Errorf("%s", rmsg)
	}

	data, err := c.ReadDotBytes()
	if err != nil {
		return "", nil, err
	}

	return rmsg, data, nil
}

func NewPOP3Conn(rwc io.ReadWriteCloser) *POP3Conn {
	pc := &POP3Conn{}
	pc.conn = textproto.NewConn(rwc)
	return pc
}
