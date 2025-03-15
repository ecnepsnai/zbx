package zbx

import (
	"encoding/json"
	"net"
	"testing"
	"time"
)

func TestStartActive(t *testing.T) {
	t.Parallel()

	var agentPort string
	go func() {
		l, err := net.Listen("tcp", "127.0.0.1:")
		if err != nil {
			panic(err)
		}

		_, port, err := net.SplitHostPort(l.Addr().String())
		if err != nil {
			panic(err)
		}
		agentPort = port

		for {
			c, err := l.Accept()
			if err != nil {
				return
			}

			msg, err := readZabbixMessage(c)
			if err != nil {
				panic(err)
			}

			var acRequest activeCheckRequest
			if json.Unmarshal(msg, &acRequest) == nil {
				reply, _ := json.Marshal(activeChecksResponse{
					Response: "success",
					Data: []SupportedItem{
						{
							Key:     "agent.ping",
							ItemId:  1000,
							Delay:   "10s",
							Timeout: "10s",
						},
					},
				})

				sendZabbixMessage(c, reply)
				c.Close()
				continue
			}

			var acData activeDataRequest
			if json.Unmarshal(msg, &acData) == nil {
				reply, _ := json.Marshal(activeDataResponse{
					Response: "success",
					Info:     "foo",
				})

				sendZabbixMessage(c, reply)
				c.Close()
				continue
			}

			panic("unknown message sent to server")
		}
	}()

	// Wait for the server to start up
	time.Sleep(10 * time.Millisecond)

	session, items, err := StartActive("example", "127.0.0.1:"+agentPort)
	if err != nil {
		t.Fatalf("Error starting active session: %s", err.Error())
	}

	if items == nil {
		t.Fatalf("No items returned")
	}

	if err := session.Send(map[int]string{
		1000: "hi",
	}); err != nil {
		t.Fatalf("Error sending data: %s", err.Error())
	}
}
