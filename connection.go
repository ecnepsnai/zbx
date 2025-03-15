package zbx

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
)

// readZabbixMessage will read a zabbix message from reader r, returning the message data or an error
func readZabbixMessage(r io.Reader) ([]byte, error) {
	// Read the first 4 bytes of the header, must be 'ZBXD'
	headerBuf := make([]byte, 4)
	if _, err := r.Read(headerBuf); err != nil && err != io.EOF {
		errorWrite("invalid header: %s", err.Error())
		return nil, err
	}
	if !bytes.Equal(headerBuf, []byte("ZBXD")) {
		// Don't recognize this header, ignore
		errorWrite("invalid header: %x", headerBuf)
		return nil, fmt.Errorf("invalid header: %x", headerBuf)
	}

	// Read 1 byte of the flags
	// Note that this library does not support compression
	flagsBuf := make([]byte, 1)
	if _, err := r.Read(flagsBuf); err != nil && err != io.EOF {
		errorWrite("invalid header: unexpected EOF")
		return nil, err
	}
	flags := flagsBuf[0]

	largePacket := false
	if 0x01&flags != 1 { // 0x01 is zabbix protocol, should always be set
		errorWrite("invalid header: unknown flags")
		return nil, fmt.Errorf("invalid header: unknown flags")
	}
	if 0x04&flags != 0 {
		largePacket = true
	}
	if 0x02&flags != 0 { // Compression
		errorWrite("invalid header: compression is not supported")
		return nil, fmt.Errorf("invalid header: compression is not supported")
	}

	var dataLength uint64
	if largePacket {
		// Read 8 bytes for the content length
		lenBuf := make([]byte, 8)
		if _, err := r.Read(lenBuf); err != nil && err != io.EOF {
			return nil, err
		}
		dlen := binary.LittleEndian.Uint64(lenBuf)
		dataLength = dlen

		// Discard 8 reserved bytes
		reserved := make([]byte, 8)
		if _, err := r.Read(reserved); err != nil && err != io.EOF {
			return nil, err
		}
		if !bytes.Equal(reserved, []byte{0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0}) {
			errorWrite("invalid header: non-zero reserved bytes: %x", reserved)
			return nil, fmt.Errorf("invalid header: non-zero reserved bytes")
		}
	} else {
		// Read 4 bytes for the content length
		lenBuf := make([]byte, 4)
		if _, err := r.Read(lenBuf); err != nil && err != io.EOF {
			return nil, err
		}
		dlen := binary.LittleEndian.Uint32(lenBuf)
		dataLength = uint64(dlen)

		// Discard 4 reserved bytes
		reserved := make([]byte, 4)
		if _, err := r.Read(reserved); err != nil && err != io.EOF {
			return nil, err
		}
		if !bytes.Equal(reserved, []byte{0x0, 0x0, 0x0, 0x0}) {
			errorWrite("invalid header: non-zero reserved bytes: %x", reserved)
			return nil, fmt.Errorf("invalid header: non-zero reserved bytes")
		}
	}

	data := make([]byte, dataLength)
	actualLen, err := r.Read(data)
	if err != nil {
		return nil, err
	}
	if dataLength != uint64(actualLen) {
		return nil, fmt.Errorf("invalid header: incorrect data length")
	}

	return data, nil
}

// sendZabbixMessage will send a zabbix message of data to writer w, returning the total count of
// data written or an error
func sendZabbixMessage(w io.Writer, data []byte) (int, error) {
	length := len(data)
	header := []byte("ZBXD\x01")
	lenBuf := make([]byte, 4)
	binary.LittleEndian.PutUint32(lenBuf, uint32(length))
	reserved := make([]byte, 4)
	// header + flags = 5 bytes
	// dlen + reserved = 8
	out := make([]byte, 5+8+len(data))

	i := 0
	// Add the header
	for _, b := range header {
		out[i] = b
		i++
	}
	// Add the data length
	for _, b := range lenBuf {
		out[i] = b
		i++
	}
	// Add reserved
	for _, b := range reserved {
		out[i] = b
		i++
	}
	// Add the data
	for _, b := range data {
		out[i] = b
		i++
	}

	return w.Write(out)
}
