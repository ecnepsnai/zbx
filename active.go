package zbx

import (
	"crypto/rand"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net"
	"time"
)

// Describes a session for a zabbix active check
type ActiveSession struct {
	dialFunc func() (net.Conn, error)
	session  string
	hostname string
	itemIdx  map[int]int
}

// Describes a supported item for zabbix active checks
type SupportedItem struct {
	Key     string `json:"key"`
	ItemId  int    `json:"itemid"`
	Delay   string `json:"delay"`
	Timeout string `json:"timeout"`
}

type activeCheckRequest struct {
	Request string `json:"request"`
	Host    string `json:"host"`
	Version string `json:"version"`
	Variant int    `json:"variant"`
}

type activeChecksResponse struct {
	Response string          `json:"response"`
	Data     []SupportedItem `json:"data"`
}

type activeDataRequest struct {
	Request string            `json:"request"`
	Data    []activeCheckData `json:"data"`
	Session string            `json:"session"`
	Host    string            `json:"host"`
	Version string            `json:"version"`
	Variant int               `json:"variant"`
}

type activeCheckData struct {
	Id     int    `json:"id"`
	ItemId int    `json:"itemid"`
	Value  string `json:"value"`
	Clock  int64  `json:"clock"`
	Ns     int64  `json:"ns"`
}

type activeDataResponse struct {
	Response string `json:"response"`
	Info     string `json:"info"`
}

// StartActiveTls wraps an active connection using TLS authentication. See [StartActive] for more details.
func StartActiveTls(agentHostname, serverAddress string, certificate tls.Certificate) (*ActiveSession, []SupportedItem, error) {
	return startActiveSession(agentHostname, func() (net.Conn, error) {
		config := &tls.Config{
			Certificates: []tls.Certificate{certificate},
		}

		return tls.Dial("tcp", serverAddress, config)
	})
}

// StartActive will prepare a connection for Zabbix active agent checks. With active agent checks, the agent (that is,
// the software that is importing and using this zbx package) is responsible for sending data to the zabbix server,
// rather than the server asking the agent.
//
// When you start an active session, you must identify yourself to the agent using the agentHostname parameter. The
// server will produce a list of items that it expects you to send, and the intervals it expects them to be sent at.
//
// For enhanced security, it's recommended that TLS authentication is used, see [StartActiveTls] for details.
func StartActive(agentHostname, serverAddress string) (*ActiveSession, []SupportedItem, error) {
	return startActiveSession(agentHostname, func() (net.Conn, error) {
		return net.Dial("tcp", serverAddress)
	})
}

func startActiveSession(agentHostname string, dialFunc func() (net.Conn, error)) (*ActiveSession, []SupportedItem, error) {
	body, err := json.Marshal(activeCheckRequest{
		Request: "active checks",
		Host:    agentHostname,
		Version: "7.0.0",
		Variant: 2,
	})
	if err != nil {
		return nil, nil, err
	}

	conn, err := dialFunc()
	if err != nil {
		return nil, nil, err
	}

	session := &ActiveSession{
		dialFunc: dialFunc,
		session:  sessionId(),
		hostname: agentHostname,
	}

	if _, err := sendZabbixMessage(conn, body); err != nil {
		return nil, nil, err
	}

	data, err := readZabbixMessage(conn)
	if err != nil {
		return nil, nil, err
	}

	var reply activeChecksResponse
	if err := json.Unmarshal(data, &reply); err != nil {
		return nil, nil, err
	}

	if reply.Response != "success" {
		return nil, nil, fmt.Errorf("unsuccessful response to active checks query")
	}

	itemIdx := map[int]int{}
	for _, item := range reply.Data {
		itemIdx[item.ItemId] = 1
	}
	session.itemIdx = itemIdx

	return session, reply.Data, nil
}

// Send will send the mapping of itemId to value to the zabbix server. Items should match those
// presented by the zabbix server when this session was started. Each call to [ActiveSession.Send]
// will make a new connection to the Zabbix server, so you may wish to batch item values together.
func (s *ActiveSession) Send(values map[int]string) error {
	conn, err := s.dialFunc()
	if err != nil {
		return err
	}

	request := activeDataRequest{
		Request: "agent data",
		Session: s.session,
		Host:    s.hostname,
		Version: "7.0.0",
		Variant: 2,
	}

	for itemId, value := range values {
		idx := s.itemIdx[itemId]
		s.itemIdx[itemId] = idx + 1
		t := time.Now()
		request.Data = append(request.Data, activeCheckData{
			Id:     idx,
			ItemId: itemId,
			Value:  value,
			Clock:  t.Unix(),
			Ns:     int64(t.Nanosecond()),
		})
	}

	data, err := json.Marshal(request)
	if err != nil {
		return err
	}

	if _, err := sendZabbixMessage(conn, data); err != nil {
		return err
	}

	replyData, err := readZabbixMessage(conn)
	if err != nil {
		return err
	}

	reply := activeDataResponse{}
	if err := json.Unmarshal(replyData, &reply); err != nil {
		return err
	}

	if reply.Response == "success" {
		return nil
	}

	failure := reply.Info
	if failure == "" {
		failure = "unrecognized reply from server"
	}

	return fmt.Errorf("send error: %s", failure)
}

func sessionId() string {
	b := make([]byte, 16)
	rand.Read(b)
	return fmt.Sprintf("%x", b)
}
