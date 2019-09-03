package zbx

import (
	"bytes"
	"os"
	"testing"

	"github.com/ecnepsnai/logtic"
)

func TestMain(m *testing.M) {
	logtic.Log.FilePath = "/dev/null"
	logtic.Log.Level = logtic.LevelDebug
	logtic.Open()
	os.Exit(m.Run())
}

type testAgentType struct{}

func (a testAgentType) GetItem(key string) (interface{}, error) {
	if key == "agent.ping" {
		return uint(1), nil
	}
	return nil, nil
}

func TestReader(t *testing.T) {
	reader := bytes.NewReader([]byte("\x5a\x42\x58\x44\x01\x0a\x00\x00\x00\x00\x00\x00\x00\x61\x67\x65\x6e\x74\x2e\x70\x69\x6e\x67"))
	expectedResponse := []byte("\x5A\x42\x58\x44\x01\x01\x00\x00\x00\x00\x00\x00\x00\x31")
	response := consumeReader(testAgentType{}, reader)
	if bytes.Compare(response, expectedResponse) != 0 {
		t.Errorf("Unexpected response\nExpected: %v\nGot:      %v", expectedResponse, response)
	}
}
