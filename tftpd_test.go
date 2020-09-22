package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"net"
	"testing"
	"time"
)

const ServerAddr = "127.0.0.1:69"

const (
	RRQ   = 1
	WRQ   = 2
	DATA  = 3
	ACK   = 4
	ERROR = 5
)

type RequestPacket struct {
	opcode   uint16
	filename string
	mode     string
}

func (rrq RequestPacket) MarshalBinary() (data []byte, err error) {
	var packet bytes.Buffer
	var opcode [2]byte
	binary.BigEndian.PutUint16(opcode[:], rrq.opcode)
	packet.Write(opcode[:])
	packet.Write([]byte(rrq.filename))
	packet.Write([]byte{0})
	packet.Write([]byte(rrq.mode))
	packet.Write([]byte{0})
	return packet.Bytes(), nil
}

func TestReadRequest(t *testing.T) {
	conn, err := net.ListenPacket("udp", ":0")
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	dst, err := net.ResolveUDPAddr("udp", ServerAddr)
	if err != nil {
		t.Fatal(err)
	}

	rrq := RequestPacket{RRQ, "video.avi", "octet"}
	packet, _ := rrq.MarshalBinary()

	_, err = conn.WriteTo(packet, dst)
	if err != nil {
		t.Fatal(err)
	}

	conn.SetReadDeadline(time.Now().Add(1 * time.Second))
	buf := make([]byte, 1024)
	n, addrFrom, err := conn.ReadFrom(buf)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Print(n, addrFrom, buf)
}
