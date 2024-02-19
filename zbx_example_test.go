package zbx_test

import (
	"crypto/tls"
	"runtime"

	"github.com/ecnepsnai/zbx"
)

func Example() {
	// This function is called for each incoming request from the Zabbix server
	getItem := func(itemKey string) (interface{}, error) {
		if itemKey == "agent.ping" {
			return "1", nil
		} else if itemKey == "runtime.version" {
			return runtime.Version, nil
		}

		// Returning nil, nil means the itemKey was unknown
		return nil, nil
	}

	// This will block
	zbx.Start(getItem, "0.0.0.0:10050")
}

func ExampleStart() {
	// This function is called for each incoming request from the Zabbix server
	getItem := func(itemKey string) (interface{}, error) {
		if itemKey == "agent.ping" {
			return "1", nil
		} else if itemKey == "runtime.version" {
			return runtime.Version, nil
		}

		// Returning nil, nil means the itemKey was unknown
		return nil, nil
	}

	// This will block
	zbx.Start(getItem, "0.0.0.0:10050")
}

func ExampleStartTLS() {
	// This function is called for each incoming request from the Zabbix server
	getItem := func(itemKey string) (interface{}, error) {
		if itemKey == "agent.ping" {
			return "1", nil
		} else if itemKey == "runtime.version" {
			return runtime.Version, nil
		}

		// Returning nil, nil means the itemKey was unknown
		return nil, nil
	}

	// Load the certificate and key that the zabbix agent will use for incoming connections
	// from the Zabbix server
	cert, err := tls.LoadX509KeyPair("zabbix.crt", "zabbix.key")
	if err != nil {
		panic(err)
	}

	// This will block
	zbx.StartTLS(getItem, "0.0.0.0:10050", cert)
}
