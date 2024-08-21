package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"log"
	"net"
	"os"
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

		receivedData := buf[:size]
		fmt.Println("Получен буфер: ", string(receivedData))

		// parse request
		//id, qrOpCodeAaTcRd, rcode, _, _, _, questionLength := parseRequest(receivedData)
		packet := parseRequest(receivedData)

		// response packet
		// set header
		header := setHeader(packet.header)

		// set question section
		headerLength := 12
		questionSection := receivedData[headerLength:(headerLength + questionLength)]

		// set answer section
		answerSection := setAnswerSection(questionSection)

		// set packet
		response := append(header, questionSection...)

		response = append(response, answerSection...)

		_, err = udpConn.WriteToUDP(response, source)
		if err != nil {
			fmt.Println("Failed to send response:", err)
		}
	}
}
func parseRequest(requestInBytes []byte) Request {
	fmt.Println("Длина переданного буфера в parseRequest", len(requestInBytes))

	// parse header
	headerLength := 12
	header := requestInBytes[:headerLength]
	parsedHeader := parseHeader(header)
	//id, qrOpCodeAaTcRd, rcode := parsedHeader.id, parsedHeader.qrOpCodeAaTcRd, parsedHeader.rcode

	//parse question section
	parsedQuestionSection := parseQuestionSection(requestInBytes[headerLength:])
	//label, qType, qClass, questionLength := parsedQuestionSection.label, parsedQuestionSection.qType, parsedQuestionSection.qClass, parsedQuestionSection.questionLength

	return Request{header: parsedHeader, questionSection: parsedQuestionSection}
}

func parseHeader(requestInBytes []byte) HeaderOptions {
	fmt.Println("Длина переданного буфера в parseHeader", len(requestInBytes))

	// extract id
	id := requestInBytes[:2]

	// allocate a byte for the next 5 options: qr, opCode, aa, tc, rd
	qrOpCodeAaTcRd := requestInBytes[2]

	// extract opCode
	opCodeMask := byte(0b01111000)
	opCode := qrOpCodeAaTcRd & opCodeMask

	// set qr to 1
	qrMask := byte(0b10000000)
	qrOpCodeAaTcRd |= qrMask

	// set rcode based on opCode value
	// 0 - standard query
	var rcode byte
	if opCode == 0 {
		rcode = byte(0) // no error
	} else {
		rcode = byte(4) //else
	}
	return HeaderOptions{id, qrOpCodeAaTcRd, rcode}
}

func parseQuestionSection(requestInBytes []byte) QuestionSection {
	fmt.Println("Длина переданного буфера в parseQuestionSection", len(requestInBytes))
	nullByteIndex := bytes.Index(requestInBytes, []byte{0})
	fmt.Println("nullByteIndex", nullByteIndex)

	labelLength := nullByteIndex + 1
	typeLength := 2
	classLength := 2
	questionLength := labelLength + typeLength + classLength

	label := requestInBytes[:labelLength]
	// parse type
	qType := requestInBytes[labelLength:(labelLength + typeLength)]

	// parse class
	qClass := requestInBytes[(labelLength + typeLength):(labelLength + typeLength + classLength)]

	return QuestionSection{label, qType, qClass, questionLength}
}

func setHeader(headerOptions HeaderOptions) []byte {
	id, qrOpCodeAaTcRd, rcode := headerOptions.id, headerOptions.qrOpCodeAaTcRd, rcode
	// hardcode values
	ra, z, qdcount, ancount, nscount, arcount := false, byte(0), uint16(1), uint16(1), uint16(0), uint16(0)

	// allocate array of 12 bytes
	header := make([]byte, 12)

	// white id into header
	copy(header[0:2], id)

	// write 5 options: qr, opCode, aa, tc, rd into header
	header[2] = qrOpCodeAaTcRd

	// allocate a byte for the next 3 options: ra, z, rcode
	raZRcode := byte(0)

	raMask := byte(0b10000000)

	if ra {
		raZRcode |= raMask
	}

	//zMask := (byte(1<<6) | byte(1<<5) | byte(1<<4)) & (z << 3)
	zMask := byte(0b01110000) & (z << 3)

	if z > 0 {
		raZRcode |= zMask
	}

	//rcodeMask := byte(1<<3) | byte(1<<2) | byte(1<<1) | byte(1)
	rcodeMask := byte(0b00001111)

	if rcode > 0 {
		raZRcode |= rcodeMask & rcode
	}

	// write 5 options: qr, opCode, aa, tc, rd into header
	header[3] = raZRcode

	// white the last 4 options: qdcount, ancount, nscount, arcount into header
	binary.BigEndian.PutUint16(header[4:], qdcount)
	binary.BigEndian.PutUint16(header[6:], ancount)
	binary.BigEndian.PutUint16(header[8:], nscount)
	binary.BigEndian.PutUint16(header[10:], arcount)

	return header
}

func setAnswerSection(questionsSection []byte) []byte {
	answerSection := questionsSection

	// add TTL info
	TTLArray := make([]byte, 4)
	binary.BigEndian.PutUint32(TTLArray, uint32(60))
	answerSection = append(answerSection, TTLArray...)

	// add length of IP info
	lengthArray := make([]byte, 2)
	IPVersion := 4
	binary.BigEndian.PutUint16(lengthArray, uint16(IPVersion))
	answerSection = append(answerSection, lengthArray...)

	// add IP info
	data := setIPIntoBytes()
	answerSection = append(answerSection, data...)

	return answerSection
}

func setIPIntoBytes() []byte {

	// hardcode ip address
	ip := "80.156.23.56"

	// convert ip into []bytes
	ipInBytes := net.ParseIP(ip)

	if ipInBytes == nil {
		os.Exit(1)
	}
	return ipInBytes
}

type Request struct {
	header          HeaderOptions
	questionSection QuestionSection
}

type HeaderOptions struct {
	id             []byte
	qrOpCodeAaTcRd byte
	rcode          byte
}

type QuestionSection struct {
	label          []byte
	qType          []byte
	qClass         []byte
	questionLength int
}

//  response[0]=1234 >> 8; response[1]= 1234 & 0xFF;
// 1234 - это число, наверно подефолту на 32 бита, нам нужно его записать в 16 бит, 2 байта, пишем сначала один байт (>>8), затем второй (& 0xFF - отбрасываем лишние байты)
//  потому что 1234 - это 16 бит и ты делишь на 256 ( сдвиг на 8 битов)
