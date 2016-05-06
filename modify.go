package milter

// A Modifier provides methods for modifying the current message.
type Modifier interface {
	// AddRecipient adds a recipient to the message.
	AddRecipient(r string)

	// DeleteRecipient removes a recipient from the message.
	DeleteRecipient(r string)

	// ReplaceBody replaces the message body.
	ReplaceBody(newBody []byte)

	// AddHeader adds a header.
	AddHeader(name, value string)

	// ChangeHeader replaces an existing header. Since there can be multiple
	// headers with the same name, index specifies which one to change;
	// the first header with that name is numbered 1 (not 0). To delete a
	// header, replace it with the empty string.
	ChangeHeader(name string, index int, value string)
}

// writeModification is like writeResponse, but it stores any error encountered
// instead of returning it.
func (c *conn) writeModification(code byte, data []byte) {
	err := c.writeResponse(code, data)
	if err != nil && c.err == nil {
		c.err = err
	}
}

func (c *conn) AddRecipient(r string) {
	c.writeModification('+', encode("<"+r+">"))
}

func (c *conn) DeleteRecipient(r string) {
	c.writeModification('-', encode("<"+r+">"))
}

func (c *conn) ReplaceBody(newBody []byte) {
	c.writeModification('b', newBody)
}

func (c *conn) AddHeader(name, value string) {
	var data struct {
		Name  string
		Value string
	}
	data.Name = name
	data.Value = value
	c.writeModification('h', encode(data))
}

func (c *conn) ChangeHeader(name string, index int, value string) {
	var data struct {
		Index uint32
		Name  string
		Value string
	}
	data.Index = uint32(index)
	data.Name = name
	data.Value = value
	c.writeModification('m', encode(data))
}
