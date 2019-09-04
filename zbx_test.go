package zbx

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
	"testing"

	"github.com/ecnepsnai/logtic"
)

func TestMain(m *testing.M) {
	verbose := false
	for _, arg := range os.Args {
		if arg == "-v" {
			verbose = true
		}
	}
	if verbose {
		logtic.Log.FilePath = "/dev/null"
		logtic.Log.Level = logtic.LevelDebug
		if err := logtic.Open(); err != nil {
			panic(err)
		}
		log = logtic.Connect("zbx")
	}
	os.Exit(m.Run())
}

type testAgentType struct{}

func (a testAgentType) GetItem(key string) (interface{}, error) {
	if key == "agent.ping" {
		return uint(1), nil
	} else if key == "generate.error" {
		return nil, fmt.Errorf("this is an error")
	}
	return nil, nil
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
	// Ensure that the agent responds with '1' for a valid key
	reader := bytes.NewReader(requestForKey("agent.ping"))
	expectedResponse := []byte("\x5A\x42\x58\x44\x01\x01\x00\x00\x00\x00\x00\x00\x00\x31")
	response := consumeReader(testAgentType{}, reader)
	if bytes.Compare(response, expectedResponse) != 0 {
		t.Errorf("Unexpected response\nExpected: %v\nGot:      %v", expectedResponse, response)
	}
}

func TestAgentError(t *testing.T) {
	// Ensure that the agent responds with the expected error for a key that generates an error
	reader := bytes.NewReader(requestForKey("generate.error"))
	expectedResponse := []byte("\x5A\x42\x58\x44\x01\x21\x00\x00\x00\x00\x00\x00\x00\x5A\x42\x58\x5F\x4E\x4F\x54\x53\x55\x50\x50\x4F\x52\x54\x45\x44\x00\x74\x68\x69\x73\x20\x69\x73\x20\x61\x6E\x20\x65\x72\x72\x6F\x72")
	response := consumeReader(testAgentType{}, reader)
	if bytes.Compare(response, expectedResponse) != 0 {
		t.Errorf("Unexpected response\nExpected: %v\nGot:      %x", expectedResponse, response)
	}
}

func TestUnknownKey(t *testing.T) {
	// Ensure that the agent returns with the expected error for an unknown key
	reader := bytes.NewReader(requestForKey("not.a.key"))
	expectedResponse := []byte("\x5A\x42\x58\x44\x01\x21\x00\x00\x00\x00\x00\x00\x00\x5A\x42\x58\x5F\x4E\x4F\x54\x53\x55\x50\x50\x4F\x52\x54\x45\x44\x00\x49\x74\x65\x6D\x20\x6B\x65\x79\x20\x75\x6E\x6B\x6E\x6F\x77\x6E")
	response := consumeReader(testAgentType{}, reader)
	if bytes.Compare(response, expectedResponse) != 0 {
		t.Errorf("Unexpected response\nExpected: %v\nGot:      %v", expectedResponse, response)
	}
}

func TestBadHeader(t *testing.T) {
	// Ensure that the agent does not attempt to reply to a request with an invalid header
	reader := bytes.NewReader([]byte("Hack the planet!"))
	response := consumeReader(testAgentType{}, reader)
	if len(response) > 0 {
		t.Errorf("Unexpected response\nExpected no response.\nGot: %v", response)
	}
}

func TestOversizedRequest(t *testing.T) {
	// Ensure that the agent does not attempt to reply to a request that reports its size as being over 128MiB
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

	reader := bytes.NewReader(request)
	response := consumeReader(testAgentType{}, reader)
	if len(response) > 0 {
		t.Errorf("Unexpected response\nExpected no response.\nGot: %v", response)
	}
}

func TestFalseDataLength(t *testing.T) {
	// Ensure that the agent does not attempt to reply to a request that falsified its data length
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

	reader := bytes.NewReader(request)
	response := consumeReader(testAgentType{}, reader)
	if len(response) > 0 {
		t.Errorf("Unexpected response\nExpected no response.\nGot: %v", response)
	}
}
