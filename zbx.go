/*
Package zbx is a Zabbix Agent implementation in golang that allows your application
to act as a zabbix agent and respond to simple requests.

It is compatible with Zabbix version 4 and newer, however it does not support compression or TLS PSK
authentication.
*/
package zbx

import (
	"crypto/tls"
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
		if _, err := sendZabbixMessage(conn, reply); err != nil {
			errorWrite("Error writing reply: %s,%s", fmt.Sprintf("remote_addr='%s'", who), fmt.Sprintf("error='%s'", err.Error()))
		}
	}

	conn.Close()
}

func consumeReader(itemFunc ItemFunc, r io.Reader) []byte {
	keyNameBuf, err := readZabbixMessage(r)
	if err != nil {
		errorWrite("Error reading message: %s", err.Error())
		return nil
	}
	key := string(keyNameBuf)

	respObj, err := safeCallItemFunc(itemFunc, key)
	if err != nil {
		// Error from the agent
		errorWrite("Error reading request key: %s,%s", fmt.Sprintf("key='%s'", key), fmt.Sprintf("error='%s'", err.Error()))
		return []byte("ZBX_NOTSUPPORTED\x00" + err.Error())
	} else if respObj == nil {
		// No error but no reply, key not found
		return []byte("ZBX_NOTSUPPORTED\x00Item key unknown")
	}

	// Format the reply as a string
	return []byte(fmt.Sprintf("%v", respObj))
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
