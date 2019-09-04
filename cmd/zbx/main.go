package main

import (
	"github.com/ecnepsnai/zbx"
)

type agent struct{}

func main() {
	a := agent{}
	zbx.Start(a, "0.0.0.0:10050")
}

func (a agent) GetItem(key string) (interface{}, error) {
	if key == "agent.ping" {
		return uint(1), nil
	}
	return nil, nil
}
