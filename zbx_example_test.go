package zbx_test

import (
	"runtime"

	"github.com/ecnepsnai/zbx"
)

func Example() {
	getItem := func(itemKey string) (interface{}, error) {
		if itemKey == "agent.ping" {
			return "1", nil
		} else if itemKey == "runtime.version" {
			return runtime.Version, nil
		}

		return nil, nil
	}

	zbx.Start(getItem, "0.0.0.0:10050")
}

func ExampleStart() {
	getItem := func(itemKey string) (interface{}, error) {
		if itemKey == "agent.ping" {
			return "1", nil
		} else if itemKey == "runtime.version" {
			return runtime.Version, nil
		}

		return nil, nil
	}

	zbx.Start(getItem, "0.0.0.0:10050")
}
