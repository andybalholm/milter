package milter

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/textproto"
	"strconv"
	"strings"
)

// Serve accepts connections received on l, and processes them with milters
// returned by newMilter.
func Serve(l net.Listener, newMilter func() Milter) error {
	for {
		c, err := l.Accept()
		if err != nil {
			return err
		}

		go func() {
			mc := newConn(c)
			err := mc.run(newMilter())
			if err != nil {
				log.Println(err)
			}
		}()
	}
}

func (c *conn) run(milter Milter) error {
	defer c.conn.Close()

	for {
		data, err := c.readPacket()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		if len(data) == 0 {
			return errors.New("zero-length command packet")
		}

		command := data[0]
		data = data[1:]
		var resp Response = Continue

		if command != c.macrosFor {
			c.macros = nil
		}

		switch command {
		case 'O':
			// Negotiate connection options.
			var optNeg struct {
				Version  uint32
				Actions  uint32
				Protocol uint32
			}
			if err := decode(data, &optNeg); err != nil {
				return fmt.Errorf("error decoding options from server: %v", err)
			}
			optNeg.Protocol = 0
			if err := c.writeResponse('O', encode(optNeg)); err != nil {
				return err
			}
			continue // Writing the 'O' response was all the response that was needed.

		case 'D':
			// Define macros.
			if len(data) == 0 {
				return errors.New("macro-definition packet with no data")
			}
			c.macrosFor = data[0]
			c.macros = map[string]string{}

			kv := splitCStrings(data[1:])
			for i := 0; i < len(kv)-1; i += 2 {
				c.macros[stripBrackets(kv[i], "{}")] = kv[i+1]
			}
			continue // A macro packet doesn't need a response.

		case 'A':
			// Abort (cancel current message and get ready to process a new one).
			c.headers = nil
			c.body = nil
			continue // An abort packet doesn't need a response.

		case 'Q':
			// Quit.
			return nil

		case 'C':
			// Connect.
			var connInfo struct {
				Hostname       string
				ProtocolFamily byte
				Port           uint16
				Address        string
			}
			if err := decode(data, &connInfo); err != nil {
				return fmt.Errorf("error decoding connection info: %v", err)
			}
			var network, address string
			switch connInfo.ProtocolFamily {
			case 'L':
				network = "unix"
				address = connInfo.Address
			case '4':
				network = "tcp4"
				address = net.JoinHostPort(connInfo.Address, strconv.Itoa(int(connInfo.Port)))
			case '6':
				network = "tcp6"
				address = net.JoinHostPort(connInfo.Address, strconv.Itoa(int(connInfo.Port)))
			}
			resp = milter.Connect(connInfo.Hostname, network, address, c.macros)

		case 'H':
			// HELO.
			name := strings.TrimSuffix(string(data), "\x00")
			resp = milter.Helo(name, c.macros)

		case 'M':
			// MAIL FROM.
			args := splitCStrings(data)
			if len(args) == 0 {
				return errors.New("MAIL FROM with no address")
			}
			from := stripBrackets(args[0], "<>")
			resp = milter.From(from, c.macros)

		case 'R':
			// RCPT TO.
			args := splitCStrings(data)
			if len(args) == 0 {
				return errors.New("RCPT TO with no address")
			}
			to := stripBrackets(args[0], "<>")
			resp = milter.To(to, c.macros)

		case 'T':
			// DATA.
			// Just ignore it, to avoid complicating the milter interface further.

		case 'L':
			// a header
			if c.headers == nil {
				c.headers = make(textproto.MIMEHeader)
			}
			keyVal := splitCStrings(data)
			if len(keyVal) != 2 {
				return fmt.Errorf("header key/value pair with %d items (should be 2)", len(keyVal))
			}
			c.headers.Add(keyVal[0], keyVal[1])

		case 'N':
			// end of headers
			resp = milter.Headers(c.headers)
			c.headers = nil

		case 'B':
			// a chunk of the body
			c.body = append(c.body, data...)

		case 'E':
			// the end of the body
			resp = milter.Body(c.body, c)
			if c.err != nil {
				return c.err
			}
			c.body = nil

		default:
			log.Printf("Unrecognized command code: %c", command)
		}

		if err := c.writeResponse(resp.Response()); err != nil {
			return err
		}
	}
}
