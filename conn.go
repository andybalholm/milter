package milter

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"net/textproto"
	"strings"
)

// A conn is a network connection from the mail transfer agent.
type conn struct {
	conn io.ReadWriteCloser
	bw   *bufio.Writer
	err  error

	macros    map[string]string
	macrosFor byte

	headers textproto.MIMEHeader
	body    []byte
}

// newConn returns a new Conn wrapping c.
func newConn(c io.ReadWriteCloser) *conn {
	return &conn{
		conn: c,
		bw:   bufio.NewWriter(c),
	}
}

// readPacket reads a command packet and returns the packet data (excluding the
// length prefix).
func (c *conn) readPacket() ([]byte, error) {
	var length uint32
	if err := binary.Read(c.conn, binary.BigEndian, &length); err != nil {
		return nil, err
	}

	data := make([]byte, length)
	if _, err := io.ReadFull(c.conn, data); err != nil {
		return nil, err
	}
	return data, nil
}

// writeResponse sends a response packet.
func (c *conn) writeResponse(code byte, data []byte) error {
	if err := binary.Write(c.bw, binary.BigEndian, uint32(len(data)+1)); err != nil {
		return err
	}
	if err := c.bw.WriteByte(code); err != nil {
		return err
	}
	if _, err := c.bw.Write(data); err != nil {
		return err
	}
	return c.bw.Flush()
}

type unexpectedCommandError byte

func (e unexpectedCommandError) Error() string {
	return fmt.Sprintf("unexpected command code: %q", byte(e))
}

// splitCStrings takes a byte slice full of null-terminated strings, and
// returns them as a slice of Go strings.
func splitCStrings(data []byte) []string {
	if len(data) == 0 {
		return nil
	}
	if data[len(data)-1] == 0 {
		data = data[:len(data)-1]
	}

	return strings.Split(string(data), "\x00")
}

// stripBrackets returns s without its surrounding brackets, if it is enclosed
// in the pair of brackets specified. brackets must be a two-character string
// containing the opening and closing brackets.
func stripBrackets(s, brackets string) string {
	if len(s) > 2 && s[0] == brackets[0] && s[len(s)-1] == brackets[1] {
		return s[1 : len(s)-1]
	}
	return s
}
