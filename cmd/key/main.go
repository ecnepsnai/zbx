package main

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"os"
)

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintf(os.Stderr, "Requires 2 arguments\n")
		os.Exit(1)
	}

	host := os.Args[1]
	key := os.Args[2]

	request := requestForKey(key)

	conn, err := net.Dial("tcp", host)
	if err != nil {
		panic(err)
	}
	defer conn.Close()
	if _, err := conn.Write(request); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing to host: %s\n", err.Error())
		return
	}

	buf := make([]byte, 128)
	length, err := conn.Read(buf)
	if err != nil && err != io.EOF {
		fmt.Fprintf(os.Stderr, "Error reading from host: %s\n", err.Error())
		return
	}

	fmt.Printf("%x\n%s\n", buf[:length], buf[:length])
	return
}

func requestForKey(key string) []byte {
	length := make([]byte, 8)
	binary.LittleEndian.PutUint64(length, uint64(len(key)))
	request := make([]byte, 13+len(key))
	header := []byte("ZBXD\x01")
	i := 0
	for _, b := range header {
		request[i] = b
		i++
	}
	for _, b := range length {
		request[i] = b
		i++
	}
	for _, b := range []byte(key) {
		request[i] = b
		i++
	}
	return request
}
