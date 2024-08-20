package main

import (
	"encoding/binary"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
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

		// response packet
		// quesitons section
		questionSection, qdcount := setQuestionSection(questionSectionOptions{"codecrafters.io", A, IN})

		// answer section

		answerSection, ancount := setAnswerSection(questionSection)

		//header
		header := setHeader(HeaderOptions{uint16(1234), true, byte(0), false, false, false, false, byte(0), byte(0), uint16(qdcount), uint16(ancount), uint16(0), uint16(0)})

		response := append(header, questionSection...)
		response = append(response, answerSection...)

		_, err = udpConn.WriteToUDP(response, source)
		if err != nil {
			fmt.Println("Failed to send response:", err)
		}
	}
}

type Packet struct {
	header          []byte
	questionSection []byte
}

type questionSectionOptions struct {
	name  string
	qType qType
	class qClass
}

type answerSectionOptions struct {
	name   string
	qType  qType
	class  qClass
	TTL    int32
	length int16
	data   int32
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
	ancount uint16
	nscount uint16
	arcount uint16
}

func setHeader(headerOptions HeaderOptions) []byte {
	// deconstruct
	id, qr, opCode, aa, tc, rd, ra, z, rcode, qdcount, abcount, nscount, arcount := headerOptions.id, headerOptions.qr, headerOptions.opCode, headerOptions.aa, headerOptions.tc, headerOptions.rd, headerOptions.ra, headerOptions.z, headerOptions.rcode, headerOptions.qdcount, headerOptions.ancount, headerOptions.nscount, headerOptions.arcount

	// masks
	qrMask := byte(1 << 7)
	// TODO: opCodeMask := byte(0b01111000)
	opCodeMask := (byte(1<<6) | byte(1<<5) | byte(1<<4) | byte(1<<3)) & (opCode << 3)
	aaMask := byte(1 << 2)
	tcMask := byte(1 << 1)
	rdMask := byte(1)

	raMask := byte(1 << 7)
	// TODO:  opCodeMask := byte(0b01110000)
	zMask := (byte(1<<6) | byte(1<<5) | byte(1<<4)) & (z << 3)
	rcodeMask := byte(1<<3) | byte(1<<2) | byte(1<<1) | byte(1)

	// allocate array of 12 bytes
	header := make([]byte, 12)

	// put first option
	binary.BigEndian.PutUint16(header[0:], id)

	// allocate a byte for the next 5 options: qr, opCode, aa, tc, rd
	qrOpCodeAaTcRd := byte(0)

	if qr {
		qrOpCodeAaTcRd |= qrMask //128 // (1 << 7) == 128
	}

	if opCode > 0 {
		qrOpCodeAaTcRd |= opCodeMask //(15 << 3) == 120
	}

	if aa {
		qrOpCodeAaTcRd |= aaMask
	}

	if tc {
		qrOpCodeAaTcRd |= tcMask
	}

	if rd {
		qrOpCodeAaTcRd |= rdMask
	}

	// write 5 options: qr, opCode, aa, tc, rd into header
	header[2] = qrOpCodeAaTcRd

	// allocate a byte for the next 3 options: ra, z, rcode
	raZRcode := byte(0)

	if ra {
		raZRcode |= raMask
	}

	if z > 0 {
		raZRcode |= zMask
	}

	if rcode > 0 {
		raZRcode |= rcodeMask & rcode
	}

	// write 5 options: qr, opCode, aa, tc, rd into header
	header[3] = raZRcode

	// white the last 4 options: qdcount, ancount, nscount, arcount into header
	binary.BigEndian.PutUint16(header[4:], qdcount)
	binary.BigEndian.PutUint16(header[6:], abcount)
	binary.BigEndian.PutUint16(header[8:], nscount)
	binary.BigEndian.PutUint16(header[10:], arcount)
	return header
}

// TODO: возможно, нужно принимать массив строк. тогда и оценивать QDCount как больше единицы
// сейчас принимается один name === entry для парсинга, поэтому увеличивает QDCount на единицу
func setQuestionSection(questionSectionOptions questionSectionOptions) ([]byte, int) {

	// deconstruct
	name, qType, class := questionSectionOptions.name, questionSectionOptions.qType, questionSectionOptions.class

	// as we deal with string, not string[] - the number of entries is always 1
	numberOfEntries := 1
	// name := 'codecrafters.io'
	// \x00

	response := setLabelToByteArray(name)

	// add nullByte terminator to the end of entry
	//offset += 1

	// add type info
	qTypeArray := make([]byte, 2)
	binary.BigEndian.PutUint16(qTypeArray, uint16(qType))
	response = append(response, qTypeArray...)

	// add class info
	classArray := make([]byte, 2)
	binary.BigEndian.PutUint16(classArray, uint16(class))
	response = append(response, classArray...)

	return response, numberOfEntries

}

func setLabelToByteArray(name string) []byte {
	nameSplit := strings.Split(name, ".")

	// null terminator
	nullByte := byte('\x00')

	// allocate array of byte for response & offset
	var response []byte
	//offset := 0

	for _, value := range nameSplit {
		lengthInBytes := len(value)
		response = append(response, byte(lengthInBytes))
		//offset += 1
		response = append(response, []byte(value)...)
		//offset += lengthInBytes - 1
	}

	response = append(response, nullByte)

	return response
}

func setAnswerSection(response []byte) ([]byte, int) {

	numberOfEntries := 1

	// add type info
	TTLArray := make([]byte, 4)
	binary.BigEndian.PutUint32(TTLArray, uint32(60))
	response = append(response, TTLArray...)

	// add class info
	lenghtArray := make([]byte, 2)

	//
	IPversion := 4
	binary.BigEndian.PutUint16(lenghtArray, uint16(IPversion))
	response = append(response, lenghtArray...)

	data := setIPdataInBytes()
	response = append(response, data...)

	return response, numberOfEntries

}

type qType uint16

const (
	A qType = iota + 1
	NS
	MD
	MF
	CNAME
	SOA
	MB
	MG
	MR
	NULL
	WKS
	PTR
	HINFO
	MINFO
	MX
	TXT
)

type qClass uint16

const (
	IN qClass = iota + 1
	CS
	CH
	HS
)

func setIPdataInBytes() []byte {

	ip := "80.156.23.56"
	ipInBytes := net.ParseIP(ip)

	if ipInBytes == nil {
		os.Exit(1)
	}
	return ipInBytes
}
