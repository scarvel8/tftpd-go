// Copyright 2017 Steven Carvellas
// Implementation of RFC 1350
// TFTP server written in Go

package main

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"io"
	"log"
	"net"
	"os"
)

// opcodes
const (
	RRQ   = 1
	WRQ   = 2
	DATA  = 3
	ACK   = 4
	ERROR = 5
)

// Logging variables
var (
	Info  *log.Logger
	Error *log.Logger
)

// Initial contains the initial packet from tftp client
type Initial struct {
	Opcode   int
	Filename string
	Mode     string
}

// DataPacket is equivalent to Figure 5-2: DATA packet in RFC 1350
type DataPacket struct {
	Opcode []byte
	Block  []byte
	Data   []byte
}

type ACKPacket struct {
	Opcode   int
	Blocknum int
}

func InitLogging(infoHandle io.Writer, errorHandle io.Writer) {

	Info = log.New(infoHandle, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
	Error = log.New(errorHandle, "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile)
}

func ListenForTransferACK(ser *net.UDPConn, blocknum int) (ackpacket ACKPacket) {
	buf := make([]byte, 512)
	_, _, err := ser.ReadFromUDP(buf)
	CheckError(err)

	opcode := int(binary.BigEndian.Uint16(buf[0:2]))
	blockret := int(binary.BigEndian.Uint16(buf[2:4]))

	a := &ACKPacket{
		Opcode:   opcode,
		Blocknum: blockret,
	}

	return *a
}

func decodeRRQWRQ(buf []byte) *Initial {

	// ipv4/header.go shows some examples of working with []byte's
	opcode := int(binary.BigEndian.Uint16(buf[0:2]))

	readbuffer := bytes.NewBuffer(buf[2:])
	reader := bufio.NewReader(readbuffer)
	s1, _ := reader.ReadString('\x00')

	// length of filename + 2 = starting point of mode packet, which is also a stupid string

	modeindex := len(s1) + 2
	readbuffer = bytes.NewBuffer(buf[modeindex:])
	reader = bufio.NewReader(readbuffer)
	s2, _ := reader.ReadString('\x00')

	i := &Initial{
		Opcode:   opcode,
		Filename: s1,
		Mode:     s2,
	}

	// TODO: if error set, return nil, someerror
	return i
}

func ProcessRRQRequest(ser *net.UDPConn, remote *net.UDPAddr, filename string) {
	var file, fileerr = os.OpenFile(filename[:len(filename)-1], os.O_RDONLY, 0644)
	CheckError(fileerr)
	defer file.Close()

	Info.Printf("Request for %s from %s", filename, remote.IP.String())

	blocknum := 0

	opcode := make([]byte, 2)
	binary.BigEndian.PutUint16(opcode, DATA)

	// RFC 1350 defines reading file in chunks of 512
	for fileerr != io.EOF {
		blocknum = blocknum + 1
		buf := make([]byte, 512)
		numbytes, _ := file.Read(buf)
		// check for non EOF error

		block := make([]byte, 2)

		binary.BigEndian.PutUint16(block, uint16(blocknum))
		r := &DataPacket{
			Opcode: opcode,
			Block:  block,
			Data:   buf,
		}
		payload := append(r.Opcode, r.Block...)
		payload = append(payload, r.Data[:numbytes]...)
		_, err := ser.WriteToUDP(payload, remote)
		CheckError(err)

		// We wait for an ACK, even if its the last packet

		for {
			ack := ListenForTransferACK(ser, blocknum)
			n := ack.Blocknum
			if n != blocknum {
				_, err := ser.WriteToUDP(payload, remote)
				CheckError(err)
			} else {
				break
			}
		}

		if numbytes < 512 {
			return
		}

	}
}

// CheckError all
func CheckError(err error) {
	if err != nil {
		Error.Println("Error: ", err)
		os.Exit(0)
	}
}

func Initialize() {

	ServerAddr, err := net.ResolveUDPAddr("udp", ":69")
	CheckError(err)

	ser, err := net.ListenUDP("udp", ServerAddr)
	CheckError(err)
	defer ser.Close()

	// but, is there a max to the size of the initial TFTP packet??
	buf := make([]byte, 1024)

	// first, we look for a valid RRQ/WRQ packet
	_, remoteaddr, err := ser.ReadFromUDP(buf)
	i1 := decodeRRQWRQ(buf)

	CheckError(err)

	// we should probably verify at this point that it is a "proper" TFTP packet

	switch i1.Opcode {
	case RRQ:
		ProcessRRQRequest(ser, remoteaddr, i1.Filename)
	case WRQ:
		// unimplemented
	default:
		// terminate connection
		Error.Println("Not an RRQ or WRQ packet")
	}

}

func main() {
	InitLogging(os.Stdout, os.Stderr)

	for {
		Initialize()
	}
}
