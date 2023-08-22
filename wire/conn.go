package wire

import (
	"encoding/binary"
	"io"
	"net"
)

type Conn struct {
	conn net.Conn
}

func NewConn(conn net.Conn) *Conn {
	return &Conn{conn: conn}
}

// ReadMessage blocks until a full message is read and decodes it into an RPCMessage
func (c *Conn) ReadMessage() (*Message, error) {
	sizeBuf := make([]byte, 8)
	_, err := io.ReadFull(c.conn, sizeBuf)
	if err != nil {
		return nil, err
	}
	size := binary.LittleEndian.Uint64(sizeBuf)

	readBuf := make([]byte, size)
	_, err = io.ReadFull(c.conn, readBuf)
	if err != nil {
		return nil, err
	}

	msg, err := Decode(readBuf)
	if err != nil {
		return nil, err
	}
	return &msg, nil
}

func (c *Conn) SendMessage(m *Message) error {
	data, err := m.Encode()
	if err != nil {
		return err
	}
	err = binary.Write(c.conn, binary.LittleEndian, uint64(len(data)))
	if err != nil {
		return err
	}
	_, err = c.conn.Write(data)
	return err
}
