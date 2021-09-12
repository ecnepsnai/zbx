/*
Package zbx is a Zabbix Agent implementation in golang that allows your application
to act as a zabbix agent and respond to simple requests.

It is compatible with Zabbix version 4 and newer, but it does not support encryption or compression.
*/
package zbx

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"net"

	"github.com/ecnepsnai/logtic"
)

var log = logtic.Log.Connect("zbx")

// ItemFunc describes the method invoked when the Zabbix Server (or proxy) is requesting
// an item from this agent. The returned interface be encoded as a string and returned to the server.
//
// If error is not nil, it will be sent back to the server. If (nil, nil) is returned then it is assumed
// the key is unknown.
type ItemFunc func(key string) (interface{}, error)

// Start the Zabbix agent on the specified address. Will block and always return on error.
// Will panic if itemFunc is nil.
func Start(itemFunc ItemFunc, address string) error {
	if itemFunc == nil {
		panic("itemFunc is nil")
	}

	l, err := net.Listen("tcp", address)
	if err != nil {
		return err
	}
	for {
		conn, err := l.Accept()
		if err != nil {
			return err
		}
		go newConnection(itemFunc, conn)
	}
}

func newConnection(itemFunc ItemFunc, conn net.Conn) {
	who := conn.RemoteAddr().String()
	log.PDebug("New connection", map[string]interface{}{
		"remote_address": who,
	})

	reply := consumeReader(itemFunc, conn)
	if reply != nil {
		if _, err := conn.Write(reply); err != nil {
			log.PError("Error writing reply", map[string]interface{}{
				"remote_addr": who,
				"error":       err.Error(),
			})
		}
	}

	conn.Close()
	log.PDebug("Closing connection", map[string]interface{}{
		"remote_address": who,
	})
}

func consumeReader(itemFunc ItemFunc, r io.Reader) []byte {
	// Read the first 4 bytes of the header, must be 'ZBXD'
	headerBuf := make([]byte, 4)
	if _, err := r.Read(headerBuf); err != nil && err != io.EOF {
		log.PError("Error reading request header", map[string]interface{}{
			"error": err.Error(),
		})
		return nil
	}
	if !bytes.Equal(headerBuf, []byte("ZBXD")) {
		// Don't recognize this header, ignore
		return nil
	}

	// Read 1 byte of the flags
	// Note that this library does not support compression
	flagsBuf := make([]byte, 1)
	if _, err := r.Read(flagsBuf); err != nil && err != io.EOF {
		log.PError("Error reading request flags", map[string]interface{}{
			"error": err.Error(),
		})
		return nil
	}
	if !bytes.Equal(flagsBuf, []byte("\x01")) {
		log.PWarn("Unsupported request flags", map[string]interface{}{
			"flags": flagsBuf,
		})
		return nil
	}

	// Read 4 bytes for the content length
	keyLenBuf := make([]byte, 4)
	if _, err := r.Read(keyLenBuf); err != nil && err != io.EOF {
		log.PError("Error reading request body", map[string]interface{}{
			"error": err.Error(),
		})
		return nil
	}
	dataLength := binary.LittleEndian.Uint32(keyLenBuf)

	log.PDebug("Request length", map[string]interface{}{
		"size_b": dataLength,
		"size":   logtic.FormatBytesB(uint64(dataLength)),
	})
	// Protocol is limited to 128MiB
	if dataLength >= 134217728 {
		log.PError("Oversized request", map[string]interface{}{
			"max_size":     134217728,
			"request_size": dataLength,
		})
		return nil
	}

	// Read 4 bytes for the reserved portion of the header, but don't do anything with it
	reservedBuf := make([]byte, 4)
	if _, err := r.Read(reservedBuf); err != nil && err != io.EOF {
		log.PError("Error reading request header", map[string]interface{}{
			"error": err.Error(),
		})
		return nil
	}

	// Read n bytes for the key (n=data length)
	keyBuf := make([]byte, dataLength)
	realLen, err := r.Read(keyBuf)
	if err != nil && err != io.EOF {
		log.PError("Error reading request key", map[string]interface{}{
			"error": err.Error(),
		})
		return nil
	}
	if uint32(realLen) != dataLength {
		log.PError("Incorrect request size", map[string]interface{}{
			"reported": dataLength,
			"actual":   realLen,
		})
		return nil
	}

	key := string(keyBuf)

	log.PDebug("Server requested key", map[string]interface{}{
		"key": key,
	})
	respObj, err := itemFunc(key)

	var data []byte
	if err != nil {
		// Error from the agent
		log.PError("Error getting key", map[string]interface{}{
			"key":   key,
			"error": err.Error(),
		})
		data = []byte("ZBX_NOTSUPPORTED\x00" + err.Error())
	} else if respObj == nil {
		// No error but no reply, key not found
		log.PWarn("No response for key", map[string]interface{}{
			"key": key,
		})
		data = []byte("ZBX_NOTSUPPORTED\x00Item key unknown")
	} else {
		// Format the reply as a string
		log.PDebug("Response for key", map[string]interface{}{
			"data":     data,
			"length_b": len(data),
			"key":      key,
		})
		data = []byte(fmt.Sprintf("%v", respObj))
	}

	length := len(data)
	header := []byte("ZBXD\x01")
	lenBuf := make([]byte, 8)
	binary.LittleEndian.PutUint64(lenBuf, uint64(length))
	// header + data length is 8 bytes
	reply := make([]byte, 13+length)

	i := 0
	// Add the header
	for _, b := range header {
		reply[i] = b
		i++
	}
	// Add the data length
	for _, b := range lenBuf {
		reply[i] = b
		i++
	}
	// Add the data
	for _, b := range data {
		reply[i] = b
		i++
	}

	return reply
}
