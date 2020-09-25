package main

import (
	"bytes"
	"crypto/md5"
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"testing"
	"time"
)

const ServerAddr = "127.0.0.1:69"
const VideoFileMD5Hash = "edda8bd80133569937aeef64ebbf4f0c"

const (
	RRQ   = 1
	WRQ   = 2
	DATA  = 3
	ACK   = 4
	ERROR = 5
)

type RequestPacket struct {
	filename string
	mode     string
}

func (rrq RequestPacket) MarshalBinary() ([]byte, error) {
	var packet bytes.Buffer
	var opcode [2]byte
	binary.BigEndian.PutUint16(opcode[:], RRQ)
	packet.Write(opcode[:])
	packet.Write([]byte(rrq.filename))
	packet.Write([]byte{0})
	packet.Write([]byte(rrq.mode))
	packet.Write([]byte{0})
	return packet.Bytes(), nil
}

type DataPacket struct {
	blockNumber uint16
	data        []byte
	length      int
}

func (pkt *DataPacket) UnmarshalBinary(data []byte) error {
	if len(data) < 4 {
		return errors.New("too small packet")
	}
	opcode := binary.BigEndian.Uint16(data[0:])
	if opcode != DATA {
		return errors.New("invalid packet type")
	}
	pkt.blockNumber = binary.BigEndian.Uint16(data[2:])
	pkt.data = data[4:]
	pkt.length = len(pkt.data)
	return nil
}

type AckPacket struct {
	blockNumber uint16
}

func (ack AckPacket) MarshalBinary() ([]byte, error) {
	var opcode [2]byte
	binary.BigEndian.PutUint16(opcode[:], ACK)
	var blockNumber [2]byte
	binary.BigEndian.PutUint16(blockNumber[:], ack.blockNumber)
	var packet bytes.Buffer
	packet.Write(opcode[:])
	packet.Write(blockNumber[:])
	return packet.Bytes(), nil
}

func TestReadReceiveFirstChunk(t *testing.T) {
	conn, err := sendReadRequest()
	if err != nil {
		t.Fatal(err)
	}

	_, buf, n, err := readPacket(conn)
	if err != nil {
		t.Fatal(err)
	}

	var data DataPacket
	err = data.UnmarshalBinary(buf[:n])
	if err != nil {
		t.Fatal(err)
	}

	if data.blockNumber != 1 {
		t.Fatalf("Invalid block number of %v. It was expected to get the first (1)",
			data.blockNumber)
	}

	desiredLength := 512
	if data.length != desiredLength {
		t.Fatalf("Invalid data length of %v. It was expected %v bytes.",
			data.length, desiredLength)
	}
}

func TestReadReceiveSecondChunkAfterAck(t *testing.T) {
	conn, err := sendReadRequest()
	if err != nil {
		t.Fatal(err)
	}

	addr, buf, n, err := readPacket(conn)
	if err != nil {
		t.Fatal(err)
	}

	var first DataPacket
	err = first.UnmarshalBinary(buf[:n])
	if err != nil {
		t.Fatal(err)
	}

	ack := AckPacket{first.blockNumber}
	_, err = sendAck(ack, addr, conn)
	if err != nil {
		t.Fatal(err)
	}

	_, buf, n, err = readPacket(conn)
	if err != nil {
		t.Fatal(err)
	}

	var second DataPacket
	err = second.UnmarshalBinary(buf[:n])
	if err != nil {
		t.Fatal(err)
	}

	if second.blockNumber != 2 {
		t.Fatalf("Invalid block number of %v. It was expected to get the second one (2)",
			second.blockNumber)
	}
}

func TestReadReceiveFirstChunkAgainIfNotAck(t *testing.T) {
	conn, err := sendReadRequest()
	if err != nil {
		t.Fatal(err)
	}

	_, _, _, err = readPacket(conn)
	if err != nil {
		t.Fatal(err)
	}

	_, buf, n, err := readPacket(conn)
	if err != nil {
		t.Fatal(err)
	}

	var data DataPacket
	err = data.UnmarshalBinary(buf[:n])
	if err != nil {
		t.Fatal(err)
	}

	if data.blockNumber != 1 {
		t.Fatalf("Invalid block number of %v. It was expected to get the first one (1) again.",
			data.blockNumber)
	}
}

func TestReadReceiveEntireFile(t *testing.T) {
	conn, err := sendReadRequest()
	if err != nil {
		t.Fatal(err)
	}

	lastBlockNumber := uint16(0)
	hash := md5.New()

	for {
		addr, buf, n, err := readPacket(conn)
		if err != nil {
			t.Fatal(err)
		}

		var data DataPacket
		err = data.UnmarshalBinary(buf[:n])
		if err != nil {
			t.Fatal(err)
		}

		if data.blockNumber != (lastBlockNumber + 1) {
			t.Fatalf("Invalid block number of %v. It was expected block %v.",
				data.blockNumber, lastBlockNumber+1)
		}

		lastBlockNumber++

		t.Logf("Received block %v of %v bytes.", data.blockNumber, data.length)

		hash.Write(data.data)

		if data.length < 512 {
			break
		}

		ack := AckPacket{data.blockNumber}
		_, err = sendAck(ack, addr, conn)
		if err != nil {
			t.Fatal(err)
		}
	}

	var receivedFileHashString = fmt.Sprintf("%x", hash.Sum(nil))
	if receivedFileHashString != VideoFileMD5Hash {
		t.Fatalf("Invalid file hash: %v. Expected %v.", receivedFileHashString, VideoFileMD5Hash)
	}
}

func sendReadRequest() (conn net.PacketConn, err error) {
	conn, err = net.ListenPacket("udp", ":0")
	if err != nil {
		return nil, err
	}

	dst, err := net.ResolveUDPAddr("udp", ServerAddr)
	if err != nil {
		return conn, err
	}

	rrq := RequestPacket{"video.avi", "octet"}
	packet, err := rrq.MarshalBinary()
	if err != nil {
		return conn, err
	}

	_, err = conn.WriteTo(packet, dst)
	if err != nil {
		return conn, err
	}

	return conn, nil
}

func readPacket(conn net.PacketConn) (addr net.Addr, buf []byte, n int, err error) {
	buf = make([]byte, 516)
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))

	n, addr, err = conn.ReadFrom(buf)
	return addr, buf, n, err
}

func sendAck(ack AckPacket, addr net.Addr, conn net.PacketConn) (n int, err error) {
	packet, err := ack.MarshalBinary()
	if err != nil {
		return n, err
	}

	n, err = conn.WriteTo(packet, addr)
	return n, err
}
