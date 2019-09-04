# zbx

[![Go Report Card](https://goreportcard.com/badge/github.com/ecnepsnai/zbx?style=flat-square)](https://goreportcard.com/report/github.com/ecnepsnai/zbx)
[![Godoc](http://img.shields.io/badge/go-documentation-blue.svg?style=flat-square)](https://godoc.org/github.com/ecnepsnai/zbx)
[![Releases](https://img.shields.io/github/release/ecnepsnai/zbx/all.svg?style=flat-square)](https://github.com/ecnepsnai/zbx/releases)
[![LICENSE](https://img.shields.io/github/license/ecnepsnai/zbx.svg?style=flat-square)](https://github.com/ecnepsnai/zbx/blob/master/LICENSE)

Package zbx is a Zabbix Agent implementation in golang that allows your application
to act as a zabbix agent and respond to simple requests.

It is compatible with Zabbix version 4.0 and 4.2 only. It does not support
TLS or PSK encryption at this time.

An example of how to implement an agent can be seen in cmd/zbx/main.go