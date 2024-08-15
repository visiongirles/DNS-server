package main

import (
	"encoding/binary"
	"fmt"
	"log"
	"net"
)

func main() {
	fmt.Println("Logs from your program will appear here!")

	udpAddr, err := net.ResolveUDPAddr("udp", "127.0.0.1:2053")
	if err != nil {
		fmt.Println("Failed to resolve UDP address:", err)
		return
	}

	udpConn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		fmt.Println("Failed to bind to address:", err)
		return
	}
	defer func(udpConn *net.UDPConn) {
		err := udpConn.Close()
		if err != nil {
			//fmt.Fprint(os.Stderr, "Failed to close to UDP connection:", err)
			log.Fatalf("Failed to close to UDP connection: %s", err)

		}
	}(udpConn)

	buf := make([]byte, 512)

	for {
		size, source, err := udpConn.ReadFromUDP(buf)
		if err != nil {
			fmt.Println("Error receiving data:", err)
			break
		}

		receivedData := string(buf[:size])
		fmt.Printf("Received %d bytes from %s: %s\n", size, source, receivedData)

		//  response[0]=1234 >> 8; response[1]= 1234 & 0xFF;
		// 1234 - это число, наверно подефолту на 32 бита, нам нужно его записать в 16 бит, 2 байта, пишем сначала один байт (>>8), затем второй (& 0xFF - отбрасываем лишние байты)
		//  потому что 1234 - это 16 бит и ты делишь на 256 ( сдвиг на 8 битов)

		header := setHeader(HeaderOptions{uint16(1234), true, byte(0), false, false, false, false, byte(0), byte(0), uint16(0), uint16(0), uint16(0), uint16(0)})
		_, err = udpConn.WriteToUDP(header, source)
		if err != nil {
			fmt.Println("Failed to send response:", err)
		}
	}
}

type HeaderOptions struct {
	id      uint16
	qr      bool
	opCode  byte
	aa      bool
	tc      bool
	rd      bool
	ra      bool
	z       byte
	rcode   byte
	qdcount uint16
	abcount uint16
	nscount uint16
	arcount uint16
}

func setHeader(headerOptions HeaderOptions) []byte {
	// deconstruct
	id, qr, opCode, aa, tc, rd, ra, z, rcode, qdcount, abcount, nscount, arcount := headerOptions.id, headerOptions.qr, headerOptions.opCode, headerOptions.aa, headerOptions.tc, headerOptions.rd, headerOptions.ra, headerOptions.z, headerOptions.rcode, headerOptions.qdcount, headerOptions.abcount, headerOptions.nscount, headerOptions.arcount

	// masks
	qrMask := byte(1 << 7)
	opCodeMask := byte(1<<6) | byte(1<<5) | byte(1<<4) | byte(1<<3)
	aaMask := byte(1 << 2)
	tcMask := byte(1 << 1)
	rdMask := byte(1)

	raMask := byte(1 << 7)
	zMask := byte(1<<6) | byte(1<<5) | byte(1<<4)
	rcodeMask := byte(1<<3) | byte(1<<2) | byte(1<<1) | byte(1)

	// allocate array of 12 bytes
	header := make([]byte, 12)

	// put first option
	binary.BigEndian.PutUint16(header[0:], id)

	// allocate a byte for the next 5 options: qr, opCode, aa, tc, rd
	qrOpCodeAaTcRd := byte(0)

	if qr {
		qrOpCodeAaTcRd = qrOpCodeAaTcRd | qrMask //128 // (1 << 7) == 128
	}

	if opCode > 0 {
		qrOpCodeAaTcRd = qrOpCodeAaTcRd | (opCodeMask & (opCode << 3)) //(15 << 3) == 120
	}

	if aa {
		qrOpCodeAaTcRd = qrOpCodeAaTcRd | aaMask
	}

	if tc {
		qrOpCodeAaTcRd = qrOpCodeAaTcRd | tcMask
	}

	if rd {
		qrOpCodeAaTcRd = qrOpCodeAaTcRd | rdMask
	}

	// write 5 options: qr, opCode, aa, tc, rd into header
	header[2] = qrOpCodeAaTcRd

	// allocate a byte for the next 3 options: ra, z, rcode
	raZRcode := byte(0)

	if ra {
		raZRcode = raZRcode | raMask
	}

	if z > 0 {
		raZRcode = raZRcode | (zMask & (z << 3))
	}

	if rcode > 0 {
		raZRcode = raZRcode | (rcodeMask & rcode)
	}

	// write 5 options: qr, opCode, aa, tc, rd into header
	header[3] = raZRcode

	// white the last 4 options: qdcount, abcount, nscount, arcount into header
	binary.BigEndian.PutUint16(header[4:], qdcount)
	binary.BigEndian.PutUint16(header[6:], abcount)
	binary.BigEndian.PutUint16(header[8:], nscount)
	binary.BigEndian.PutUint16(header[10:], arcount)
	return header
}
