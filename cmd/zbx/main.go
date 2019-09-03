package main

import (
	"fmt"

	"github.com/ecnepsnai/logtic"
	"github.com/ecnepsnai/zbx"
)

type agent struct{}

func main() {
	logtic.Log.FilePath = "/dev/null"
	logtic.Log.Level = logtic.LevelDebug
	logtic.Open()
	a := agent{}
	zbx.Start(a, "0.0.0.0:10050")
}

func (a agent) GetItem(key string) (interface{}, error) {
	fmt.Printf("%#v\n", key)
	if key == "agent.ping" {
		return uint(1), nil
	}
	return nil, nil
}
