# zbx

[![Go Report Card](https://goreportcard.com/badge/github.com/ecnepsnai/zbx?style=flat-square)](https://goreportcard.com/report/github.com/ecnepsnai/zbx)
[![Godoc](https://img.shields.io/badge/go-documentation-blue.svg?style=flat-square)](https://pkg.go.dev/github.com/ecnepsnai/zbx)
[![Releases](https://img.shields.io/github/release/ecnepsnai/zbx/all.svg?style=flat-square)](https://github.com/ecnepsnai/zbx/releases)
[![LICENSE](https://img.shields.io/github/license/ecnepsnai/zbx.svg?style=flat-square)](https://github.com/ecnepsnai/zbx/blob/master/LICENSE)

Package zbx is a Zabbix Agent implementation in golang that allows your application
to act as a zabbix agent and respond to simple requests.

It is compatible with Zabbix version 4 and newer, however it does not support compression and only
supports certificate based authentication.

## Usage

### Basic Agent

This sets up a basic agent with no encryption.

```go
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
```

### Agent with TLS

This sets up a certificate-based TLS agent. This package doesn't support PSK-based TLS, as crypto/tls
does not support this feature, [yet](https://github.com/golang/go/issues/6379).

```go
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
```
