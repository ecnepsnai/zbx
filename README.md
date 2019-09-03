# zbx

Package zbx is a Zabbix Agent implementation in golang that allows your application
to act as a zabbix agent and respond to simple requests.

It is compatible with Zabbix version 4.0 and 4.2 only. It does not support
TLS or PSK encryption at this time.

An example of how to implement an agent can be seen in cmd/zbx/main.go