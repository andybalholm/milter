/*
The milter package is a framework for writing milters (mail filters) for
Sendmail and Postfix.

To implement a milter, make a type that implements the Milter interface,
listen on a Unix or TCP socket, and call Serve with that socket and a factory
function that returns instances of your Milter type.
*/
package milter

import (
	"fmt"
	"net/textproto"
)

// A Milter examines email messages and decides what to do with them. Users of
// this package implement the Milter interface, and the methods are called in
// order as the conversation with the mail
// transfer agent proceeds. A single Milter may be used to process multiple
// messages; in that case the flow will jump back to an earlier point in the
// sequence of methods.
//
// If a method pertains to a stage in the mail workflow that the milter is not
// interested in, it should just return Continue.
//
// The final argument to most of the methods is macros, a map of extra, MTA-
// specific information. (If the MTA sent the macro names enclosed in curly
// braces, they have been removed.)
type Milter interface {
	// Connect is called when a new SMTP connection is received. The values for
	// network and address are in the same format that would be passed to net.Dial.
	Connect(hostname string, network string, address string, macros map[string]string) Response

	// Helo is called when the client sends its HELO or EHLO message.
	Helo(name string, macros map[string]string) Response

	// From is called when the client sends its MAIL FROM message. The sender's
	// address is passed without <> brackets.
	From(sender string, macros map[string]string) Response

	// To is called when the client sends a RCPT TO message. The recipient's
	// address is passed without <> brackets. If it returns a rejection Response,
	// only the one recipient is rejected.
	To(recipient string, macros map[string]string) Response

	// Headers is called when the message headers have been received.
	Headers(h textproto.MIMEHeader) Response

	// Body is called when the message body has been received. It gives an
	// opportunity for the milter to modify the message before it is delivered.
	Body(body []byte, m Modifier) Response
}

// A Response determines what will be done with a message or recipient.
type Response interface {
	Response() (code byte, data []byte)
}

type simpleResponse byte

func (r simpleResponse) Response() (code byte, data []byte) {
	return byte(r), nil
}

const (
	// Accept indicates that the message should be accepted and delivered, with
	// no further processing.
	Accept = simpleResponse('a')

	// Continue indicates that processing of the message should continue. Milter
	// methods should have Continue as their default return value.
	Continue = simpleResponse('c')

	// Discard indicates that the message should be discarded silently (without
	// giving an error to the sender).
	Discard = simpleResponse('d')

	// Reject rejects the message or recipient with a permanent error (5xx).
	Reject = simpleResponse('r')

	// TempFail rejects the message or recipient with a temporary error (4xx).
	TempFail = simpleResponse('t')
)

// A CustomResponse responds with a custom status code and message.
type CustomResponse struct {
	Code    int
	Message string
}

func (r CustomResponse) Response() (code byte, data []byte) {
	return 'y', []byte(fmt.Sprintf("%d %s\x00", r.Code, r.Message))
}
