package zbx_test

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"os"
	"testing"
	"time"

	"github.com/ecnepsnai/logtic"
	"github.com/ecnepsnai/zbx"
)

const socketAddr = "127.0.0.1:8765"

func TestMain(m *testing.M) {
	for _, arg := range os.Args {
		if arg == "-test.v=true" {
			logtic.Log.Level = logtic.LevelDebug
			logtic.Log.Open()
		}
	}

	items := map[string]func() (interface{}, error){
		"agent.ping": func() (interface{}, error) {
			return 1, nil
		},
		"generate.error": func() (interface{}, error) {
			return nil, fmt.Errorf("this is an error")
		},
		"agent.hostname": func() (interface{}, error) {
			return os.Hostname()
		},
		"agent.version": func() (interface{}, error) {
			return "4.0.0", nil
		},
	}
	go zbx.Start(func(key string) (interface{}, error) {
		f, ok := items[key]
		if !ok {
			return nil, nil
		}
		return f()
	}, socketAddr)

	c, err := retryDial(socketAddr)
	if err != nil {
		panic("unable to connect to socket")
	}
	c.Close()

	os.Exit(m.Run())
}

func retryDial(addr string) (net.Conn, error) {
	tries := 0
	for tries < 5 {
		c, err := net.Dial("tcp", addr)
		if err == nil {
			return c, nil
		}
		tries++
		time.Sleep(5 * time.Millisecond)
	}
	return nil, fmt.Errorf("cant connect after 5 attempts")
}

func requestForKey(key string) []byte {
	length := make([]byte, 8)
	binary.LittleEndian.PutUint64(length, uint64(len(key)))
	request := make([]byte, 13+len(key))
	header := []byte("ZBXD\x01")
	i := 0
	for _, b := range header {
		request[i] = b
		i++
	}
	for _, b := range length {
		request[i] = b
		i++
	}
	for _, b := range []byte(key) {
		request[i] = b
		i++
	}
	return request
}

func TestAgentPing(t *testing.T) {
	t.Parallel()

	c, err := retryDial(socketAddr)
	if err != nil {
		t.Fatalf("Error connecting to zabbix agent: %s", err.Error())
	}
	if _, err := c.Write(requestForKey("agent.ping")); err != nil {
		t.Fatalf("Error writing request: %s", err.Error())
	}
	reply, err := io.ReadAll(c)
	if err != nil {
		t.Fatalf("Error reading reply: %s", err.Error())
	}
	expectedResponse := []byte("\x5A\x42\x58\x44\x01\x01\x00\x00\x00\x00\x00\x00\x00\x31")
	if !bytes.Equal(reply, expectedResponse) {
		t.Errorf("Unexpected reply from server. Expected:\n%x\nGot:\n%x", expectedResponse, reply)
	}
}
func TestAgentError(t *testing.T) {
	t.Parallel()

	c, err := retryDial(socketAddr)
	if err != nil {
		t.Fatalf("Error connecting to zabbix agent: %s", err.Error())
	}
	if _, err := c.Write(requestForKey("generate.error")); err != nil {
		t.Fatalf("Error writing request: %s", err.Error())
	}
	reply, err := io.ReadAll(c)
	if err != nil {
		t.Fatalf("Error reading reply: %s", err.Error())
	}
	expectedResponse := []byte("\x5A\x42\x58\x44\x01\x21\x00\x00\x00\x00\x00\x00\x00\x5A\x42\x58\x5F\x4E\x4F\x54\x53\x55\x50\x50\x4F\x52\x54\x45\x44\x00\x74\x68\x69\x73\x20\x69\x73\x20\x61\x6E\x20\x65\x72\x72\x6F\x72")
	if !bytes.Equal(reply, expectedResponse) {
		t.Errorf("Unexpected reply from server. Expected:\n%x\nGot:\n%x", expectedResponse, reply)
	}
}

func TestUnknownKey(t *testing.T) {
	t.Parallel()

	c, err := retryDial(socketAddr)
	if err != nil {
		t.Fatalf("Error connecting to zabbix agent: %s", err.Error())
	}
	if _, err := c.Write(requestForKey("not.a.key")); err != nil {
		t.Fatalf("Error writing request: %s", err.Error())
	}
	reply, err := io.ReadAll(c)
	if err != nil {
		t.Fatalf("Error reading reply: %s", err.Error())
	}
	expectedResponse := []byte("\x5A\x42\x58\x44\x01\x21\x00\x00\x00\x00\x00\x00\x00\x5A\x42\x58\x5F\x4E\x4F\x54\x53\x55\x50\x50\x4F\x52\x54\x45\x44\x00\x49\x74\x65\x6D\x20\x6B\x65\x79\x20\x75\x6E\x6B\x6E\x6F\x77\x6E")
	if !bytes.Equal(reply, expectedResponse) {
		t.Errorf("Unexpected reply from server. Expected:\n%x\nGot:\n%x", expectedResponse, reply)
	}
}

func TestBadHeader(t *testing.T) {
	t.Parallel()

	c, err := retryDial(socketAddr)
	if err != nil {
		t.Fatalf("Error connecting to zabbix agent: %s", err.Error())
	}
	if _, err := c.Write([]byte("Hack the planet!")); err != nil {
		t.Fatalf("Error writing request: %s", err.Error())
	}
	if _, err := io.ReadAll(c); err == nil {
		t.Fatalf("No error seen when one expected")
	}
}

// Ensure that the agent does not attempt to reply to a request that reports its size as being over 128MiB
func TestOversizedRequest(t *testing.T) {
	t.Parallel()

	key := "agent.ping"
	length := make([]byte, 8)
	binary.LittleEndian.PutUint64(length, uint64(134217729))
	request := make([]byte, 13+len(key))
	header := []byte("ZBXD\x01")
	i := 0
	for _, b := range header {
		request[i] = b
		i++
	}
	for _, b := range length {
		request[i] = b
		i++
	}
	for _, b := range []byte(key) {
		request[i] = b
		i++
	}

	c, err := retryDial(socketAddr)
	if err != nil {
		t.Fatalf("Error connecting to zabbix agent: %s", err.Error())
	}
	if _, err := c.Write(request); err != nil {
		t.Fatalf("Error writing request: %s", err.Error())
	}
	if _, err := io.ReadAll(c); err == nil {
		t.Fatalf("No error seen when one expected")
	}
}

// Ensure that the agent does not attempt to reply to a request that falsified its data length
func TestFalseDataLength(t *testing.T) {
	t.Parallel()

	key := "agent.ping"
	length := make([]byte, 8)
	binary.LittleEndian.PutUint64(length, uint64(128))
	request := make([]byte, 13+len(key))
	header := []byte("ZBXD\x01")
	i := 0
	for _, b := range header {
		request[i] = b
		i++
	}
	for _, b := range length {
		request[i] = b
		i++
	}
	for _, b := range []byte(key) {
		request[i] = b
		i++
	}

	c, err := retryDial(socketAddr)
	if err != nil {
		t.Fatalf("Error connecting to zabbix agent: %s", err.Error())
	}
	if _, err := c.Write(request); err != nil {
		t.Fatalf("Error writing request: %s", err.Error())
	}
	reply, err := io.ReadAll(c)
	if err != nil {
		t.Fatalf("Error reading reply: %s", err.Error())
	}
	if len(reply) > 0 {
		t.Fatalf("Unexpected reply when none expected")
	}
}
