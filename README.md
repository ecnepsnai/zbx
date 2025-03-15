# zbx

[![Go Report Card](https://goreportcard.com/badge/github.com/ecnepsnai/zbx?style=flat-square)](https://goreportcard.com/report/github.com/ecnepsnai/zbx)
[![Godoc](https://img.shields.io/badge/go-documentation-blue.svg?style=flat-square)](https://pkg.go.dev/github.com/ecnepsnai/zbx)
[![Releases](https://img.shields.io/github/release/ecnepsnai/zbx/all.svg?style=flat-square)](https://github.com/ecnepsnai/zbx/releases)
[![LICENSE](https://img.shields.io/github/license/ecnepsnai/zbx.svg?style=flat-square)](https://github.com/ecnepsnai/zbx/blob/main/LICENSE)

Package zbx is a Zabbix Agent implementation in golang that allows your application
to act as a zabbix passive agent and respond to simple requests, or a zabbix active agent and send data to the server.

It is compatible with Zabbix version 4 and newer and supports plain-text and certificate-based TLS.

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

### Active Agent

An active agent works by pushing item data directly to the Zabbix server, rather than passive agents where the Zabbix
server pulls data from the agent.

The zbx package supports active agents using plain-text and certificate-based TLS. PSK-based TLS is not supported.

```go
session, items, err := zbx.StartActive(
    "myserver.example.com",     // The name of this agent as configured on the zabbix server
    "zabbix.example.com:10051", // The address to the zabbix server's listener - this is different from the web interface
)
if err != nil {
    panic(err)
}

// items contains a slice of what items the zabbix server expects you to send, and how frequently
if len(items) == 0 {
    // If items is empty, then the zabbix server either does not recognize this host (myserver.example.com)
    // or there are no items with the "Zabbix Agent (Active)" type.
    panic("no items")
}

// Active checks work using internal item IDs, which you must map to the item key
// For example, here we'll map the 'agent.ping' item to its id.
var pingItemId int
for _, item := range items {
    if item.Key == "agent.ping" {
        pingItemId = item.ItemId
        break
    }
}
if pingItemId == 0 {
    panic("item not found")
}

// You can send multiple item values at once, but in this example we'll only send one
if err := session.Send(map[int]string{
    pingItemId: "ok",
}); err != nil {
    panic(err)
}
```
