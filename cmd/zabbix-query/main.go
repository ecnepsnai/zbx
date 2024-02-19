// Command zabbix-query provides a simply utility to return a item value from a running zabbix
// agent.
package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"os"
)

func main() {
	if len(os.Args) != 3 {
		fmt.Printf(`Usage: %s <Host> <Key>

Where <Host> is the address and port of the zabbix agent and <Key> is the name of the item key
to request from the agent.
`, os.Args[0])
		os.Exit(1)
	}

	host := os.Args[1]
	key := os.Args[2]

	conn, err := net.Dial("tcp", host)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error dialing zabbix agent: %s", err.Error())
		os.Exit(1)
	}
	defer conn.Close()

	if err := sendRequest(conn, key); err != nil {
		fmt.Fprintf(os.Stderr, "Error sending request: %s", err.Error())
		os.Exit(1)
	}
	reply, err := readReply(conn)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading reply: %s", err.Error())
		os.Exit(1)
	}

	fmt.Printf("%s\n", reply)
}

func sendRequest(conn net.Conn, key string) error {
	header := []byte("ZBXD\x01")

	keyLength := len(key)
	keyLenBuf := make([]byte, 4)
	binary.LittleEndian.PutUint32(keyLenBuf, uint32(keyLength))

	reserved := make([]byte, 4)

	if _, err := conn.Write(header); err != nil {
		return err
	}
	if _, err := conn.Write(keyLenBuf); err != nil {
		return err
	}
	if _, err := conn.Write(reserved); err != nil {
		return err
	}
	if _, err := conn.Write([]byte(key)); err != nil {
		return err
	}
	return nil
}

func readReply(conn net.Conn) ([]byte, error) {
	header := make([]byte, 5)

	if _, err := conn.Read(header); err != nil {
		return nil, err
	}

	if !bytes.Equal([]byte("ZBXD\x01"), header) {
		return nil, fmt.Errorf("bad reply header")
	}

	dataLenBuf := make([]byte, 8)
	if _, err := conn.Read(dataLenBuf); err != nil && err != io.EOF {
		return nil, err
	}
	dataLength := binary.LittleEndian.Uint64(dataLenBuf)

	if dataLength >= 134217728 {
		return nil, fmt.Errorf("reply too large")
	}

	data := make([]byte, dataLength)
	if _, err := conn.Read(data); err != nil && err != io.EOF {
		return nil, err
	}

	return data, nil
}
