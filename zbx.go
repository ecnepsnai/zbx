/*
Package zbx is a Zabbix Agent implementation in golang that allows your application
to act as a zabbix agent and respond to simple requests.

It is compatible with Zabbix version 4 and newer, however it does not support compression or TLS PSK
authentication.
*/
package zbx

import (
	"bytes"
	"crypto/tls"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"os"
	"runtime/debug"
)

// ErrorLog is the writer that error messages are written to. By default this is stderr.
var ErrorLog io.Writer = os.Stderr

// ItemFunc describes the method invoked when the Zabbix Server (or proxy) is requesting
// an item from this agent. The returned interface be encoded as a string and returned to the
// server.
//
// If error is not nil, it will be sent back to the server. If (nil, nil) is returned then it is
// assumed the key is unknown.
//
// Any calls to `panic()` will be recovered from and written to ErrorLog and the server will act as
// if the key was unknown.
type ItemFunc func(key string) (interface{}, error)

// StartTLS will start the Zabbix agent on the specified address with TLS. The agent will present
// the given certificate to the server when connected.
// Will panic if itemFunc is nil.
func StartTLS(itemFunc ItemFunc, address string, certificate tls.Certificate) error {
	if itemFunc == nil {
		panic("itemFunc is nil")
	}

	config := &tls.Config{
		Certificates: []tls.Certificate{certificate},
	}

	l, err := tls.Listen("tcp", address, config)
	if err != nil {
		return err
	}
	StartListener(itemFunc, l)
	return nil
}

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
	StartListener(itemFunc, l)
	return nil
}

// Start the Zabbix agent on the specified listener.
func StartListener(itemFunc ItemFunc, l net.Listener) {
	for {
		conn, err := l.Accept()
		if err != nil {
			errorWrite("Error accepting connection: %s", fmt.Sprintf("error='%s'", err.Error()))
			continue
		}
		go newConnection(itemFunc, conn)
	}
}

func newConnection(itemFunc ItemFunc, conn net.Conn) {
	who := conn.RemoteAddr().String()

	reply := consumeReader(itemFunc, conn)
	if reply != nil {
		if _, err := conn.Write(reply); err != nil {
			errorWrite("Error writing reply: %s,%s", fmt.Sprintf("remote_addr='%s'", who), fmt.Sprintf("error='%s'", err.Error()))
		}
	}

	conn.Close()
}

func consumeReader(itemFunc ItemFunc, r io.Reader) []byte {
	// Read the first 4 bytes of the header, must be 'ZBXD'
	headerBuf := make([]byte, 4)
	if _, err := r.Read(headerBuf); err != nil && err != io.EOF {
		errorWrite("Error reading request header: %s", fmt.Sprintf("error='%s'", err.Error()))
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
		errorWrite("Error reading request flags: %s", fmt.Sprintf("error='%s'", err.Error()))
		return nil
	}
	if !bytes.Equal(flagsBuf, []byte("\x01")) {
		errorWrite("Unsupported request flags: %s", fmt.Sprintf("flags='%s'", fmt.Sprintf("%x", flagsBuf)))
		return nil
	}

	// Read 4 bytes for the content length
	keyLenBuf := make([]byte, 4)
	if _, err := r.Read(keyLenBuf); err != nil && err != io.EOF {
		errorWrite("Error reading request body: %s", fmt.Sprintf("error='%s'", err.Error()))
		return nil
	}
	dataLength := binary.LittleEndian.Uint32(keyLenBuf)

	// Protocol is limited to 128MiB
	if dataLength >= 134217728 {
		errorWrite("Rejecting oversides request: %s,%s", fmt.Sprintf("max_size=%d", 134217728), fmt.Sprintf("request_size=%d", dataLength))
		return nil
	}

	// Read 4 bytes for the reserved portion of the header, but don't do anything with it
	reservedBuf := make([]byte, 4)
	if _, err := r.Read(reservedBuf); err != nil && err != io.EOF {
		errorWrite("Error reading request header: %s", fmt.Sprintf("error='%s'", err.Error()))
		return nil
	}

	// Read n bytes for the key (n=data length)
	keyBuf := make([]byte, dataLength)
	realLen, err := r.Read(keyBuf)
	if err != nil && err != io.EOF {
		errorWrite("Error reading request key: %s", fmt.Sprintf("error='%s'", err.Error()))
		return nil
	}
	if uint32(realLen) != dataLength {
		errorWrite("Incorrect request size: %s,%s", fmt.Sprintf("reported=%d", dataLength), fmt.Sprintf("reported=%d", realLen))
		return nil
	}

	key := string(keyBuf)

	respObj, err := safeCallItemFunc(itemFunc, key)

	var data []byte
	if err != nil {
		// Error from the agent
		errorWrite("Error reading request key: %s,%s", fmt.Sprintf("key='%s'", key), fmt.Sprintf("error='%s'", err.Error()))
		data = []byte("ZBX_NOTSUPPORTED\x00" + err.Error())
	} else if respObj == nil {
		// No error but no reply, key not found
		data = []byte("ZBX_NOTSUPPORTED\x00Item key unknown")
	} else {
		// Format the reply as a string
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

func safeCallItemFunc(itemFunc ItemFunc, key string) (interface{}, error) {
	defer func() {
		if r := recover(); r != nil {
			errorWrite("Recovered from panic calling function for item %s: %s", key, r)
			ErrorLog.Write(debug.Stack())
		}
	}()

	return itemFunc(key)
}

func errorWrite(format string, a ...interface{}) {
	ErrorLog.Write([]byte(fmt.Sprintf(format, a...)))
	ErrorLog.Write([]byte("\n"))
}
