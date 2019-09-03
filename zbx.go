/*
Package zbx is a Zabbix Agent implementation in golang that allows your application
to act as a zabbix agent and respond to simple requests.

It is compatible with Zabbix version 4.0 and 4.2 only. It does not support
TLS or PSK encryption at this time.

An example of how to implement an agent can be seen in cmd/zbx/main.go
*/
package zbx

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"

	"github.com/ecnepsnai/logtic"
)

var log *logtic.Source

// Agent describes the interface for a Zabbix Agent
type Agent interface {
	// GetItem is called for each individual request from the zabbix server/proxy for an item
	// Your value will be encoded as a string and returned to the server
	// If error is not nil, it will be sent back to the server
	// If (nil, nil) is returned then it is assumed the key is unknown.
	GetItem(key string) (interface{}, error)
}

// Start start the zabbix agent. Will block and return on fatal error
func Start(agent Agent, address string) error {
	log = logtic.Connect("zbx")
	l, err := net.Listen("tcp", address)
	if err != nil {
		return err
	}
	for {
		conn, err := l.Accept()
		if err != nil {
			return err
		}
		go newConnection(agent, conn)
	}
}

func newConnection(agent Agent, conn net.Conn) {
	who := conn.RemoteAddr().String()
	log.Debug("New connection from '%s'", who)

	reply := consumeReader(agent, conn)
	if reply != nil {
		conn.Write(reply)
	}

	conn.Close()
	log.Debug("Closing connection")
}

func consumeReader(agent Agent, r io.Reader) []byte {
	// Read 9 bytes for the content length
	buf := make([]byte, 13)
	if _, err := r.Read(buf); err != nil && err != io.EOF {
		fmt.Printf("Error: %s\n", err.Error())
		return nil
	}

	dataLengthBuf := buf[5:9]
	dataLength := binary.LittleEndian.Uint32(dataLengthBuf)
	keyBuf := make([]byte, dataLength)
	if _, err := r.Read(keyBuf); err != nil && err != io.EOF {
		fmt.Printf("Error: %s\n", err.Error())
		return nil
	}
	key := string(keyBuf)

	log.Debug("Server requesting key '%s'", key)
	respObj, err := agent.GetItem(key)

	var data []byte
	if err != nil {
		log.Error("Error getting key '%s' from agent: %s", key, err.Error())
		data = []byte("ZBX_NOTSUPPORTED\x00" + err.Error())
	} else if respObj == nil {
		log.Warn("Nil response for key '%s'", key)
		data = []byte("ZBX_NOTSUPPORTED\x00Item key unknown")
	} else {
		data = []byte(fmt.Sprintf("%v", respObj))
		log.Debug("Response '%s' (%d) for key '%s'", data, len(data), key)
	}

	length := len(data)
	header := []byte("ZBXD\x01")
	lenBuf := make([]byte, 8)
	binary.LittleEndian.PutUint64(lenBuf, uint64(length))

	reply := make([]byte, 13+length)

	i := 0
	for _, b := range header {
		reply[i] = b
		i++
	}
	for _, b := range lenBuf {
		reply[i] = b
		i++
	}
	for _, b := range data {
		reply[i] = b
		i++
	}

	return reply
}
