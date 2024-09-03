package main

import (
	"encoding/binary"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
)

//func sendDNSMessage(address: string)

func main() {

	fmt.Println("Logs from your program will appear here!")

	receivingAddress := "127.0.0.1:2053"

	udpAddr, err := net.ResolveUDPAddr("udp", receivingAddress)
	if err != nil {
		fmt.Println("Failed to resolve UDP receivingAddress:", err)
		return
	}

	udpConn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		fmt.Println("Failed to bind to receivingAddress:", err)
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
			fmt.Println("Error receiving rdata:", err)
			break
		}

		receivedData := buf[:size]
		fmt.Println("БУФЕР В БАЙТАХ: ", receivedData)

		// parse request
		parsedRequest := parseRequest(receivedData)

		// 1. Проверка если ли флаг --resolver"
		// 2. парсим буффер и форвард
		if len(os.Args) > 2 {

			command := os.Args[1]

			switch command {
			case "--resolver":
				{
					_, done := forwardDNSPacket(parsedRequest, receivedData, err, udpConn, source)
					if done {
						return
					}
				}
			default:
				{
					fmt.Fprintf(os.Stderr, "Command %s hasn't been implemented", command)
					os.Exit(1)
				}
			}

		} else {
			response := setDNSResponse(parsedRequest)

			_, err = udpConn.WriteToUDP(response, source)
			if err != nil {
				fmt.Println("Failed to send response:", err)
			}
		}

	}
}

var wg sync.WaitGroup

func forwardDNSPacket(
	parsedRequest Request,
	receivedData []byte,
	err error,
	udpConn *net.UDPConn,
	source *net.UDPAddr,
) (error, bool) {

	forwardingAddress := os.Args[2]

	answerBuf := make([]byte, 512)
	headerSize := 12
	bytesFromRead := 0
	offset := 0

	var pairs sync.Map

	//questionSectionSize := []int{}
	//answerSectionSize := []int{}

	headerForward := parsedRequest.header

	//answerSectionInBytes := []byte{}

	for _, questionSection := range parsedRequest.questionSection {
		wg.Add(1)

		//questionSectionSize = append(questionSectionSize, questionSection.length())

		header := setHeader(headerForward, 1, 0)

		headerInBytes := header.setDataToByteArray()
		fmt.Println("[FORWARD] Header for forward in bytes", headerInBytes)
		questionSectionInBytes := questionSection.setDataToByteArray()

		request := append(headerInBytes, questionSectionInBytes...)
		fmt.Println("[FORWARD] Request for forward in bytes", request)

		// Forward DNS Packet to forwarding server
		bytes, answerSection, err2, b, done := forwardQuestionSection(
			forwardingAddress,
			request,
			answerBuf,
			offset,
			answerSectionSize,
			headerSize,
			questionSection,
			answerSectionInBytes,
		)
		bytesFromRead += bytes
		if done {
			return err2, b
		}
	}

	wg.Wait()
	fmt.Println("ОТВЕТ от форвард в байтах: ", answerBuf[:bytesFromRead])
	receivedData = answerBuf[:bytesFromRead]

	newHeader := setHeader(parsedRequest.header, int(parsedRequest.header.qdcount), int(parsedRequest.header.qdcount))
	response := newHeader.setDataToByteArray()

	for _, questionSection := range parsedRequest.questionSection {
		response = append(response, questionSection.setDataToByteArray()...)
	}

	response = append(response, answerSectionInBytes...)
	_, err = udpConn.WriteToUDP(response, source)
	if err != nil {
		fmt.Println("Failed to send response:", err)
	}

	return err, false
}

func forwardQuestionSection(
	forwardingAddress string,
	request []byte,
	answerBuf []byte,
	offset int,
	answerSectionSize []int,
	headerSize int,
	questionSection QuestionSection,
	answerSectionInBytes []byte) (int, []byte, error, bool, bool) {
	defer wg.Done()

	updConnForward, err := net.Dial("udp", forwardingAddress)
	if err != nil {
		fmt.Printf("Some error %v", err)
		return 0, nil, nil, true, true
	}

	_, errWrite := updConnForward.Write(request)

	if errWrite != nil {
		fmt.Printf("Some error %v", err)
		return 0, nil, nil, true, true
	}

	bytesRead, errRead := updConnForward.Read(answerBuf[offset:])
	answerSectionSize = append(answerSectionSize, bytesRead-headerSize-questionSection.length())

	answerSectionInBytes = append(answerSectionInBytes, answerBuf[len(request):]...)
	if errRead != nil {
		fmt.Printf("Some error %v", err)
		return 0, nil, nil, true, true
	}

	offset += len(answerBuf)
	return bytesRead, answerSectionInBytes, nil, false, false
}

func setDNSResponse(parsedRequest Request) []byte {

	// set header
	numberOfQuestionsSections, numberOfAnswerSections := len(parsedRequest.questionSection), len(parsedRequest.questionSection)
	header := setHeader(parsedRequest.header, numberOfQuestionsSections, numberOfAnswerSections)

	response := header.setDataToByteArray()
	fmt.Println("HEADER в байтах: ", response)
	fmt.Println("header size: ", len(response))

	// set question section
	questionSectionArray := parsedRequest.questionSection

	// set answer section
	var answerSectionArray []AnswerSection

	for _, questionSection := range questionSectionArray {
		answerSection := setAnswerSection(questionSection)
		answerSectionArray = append(answerSectionArray, answerSection)
		response = append(response, questionSection.setDataToByteArray()...)
	}
	fmt.Println("questionsection size: ", len(response))

	count := 1
	for _, answerSection := range answerSectionArray {

		response = append(response, answerSection.setDataToByteArray()...)
		fmt.Printf("answersection #%d size: %d \n", count, answerSection.length())
		count++
	}
	fmt.Println("answersection size: ", len(response))
	fmt.Println("РЕЗУЛЬТАТ В БАЙТАХ: ", response)

	return response
}

// TODO: rlength parser questionsection

// func setQuestionSection(receivedData []byte, parsedRequest Request) []byte {
// headerLength := 12
// return receivedData[headerLength:(headerLength + parsedRequest.questionSection.rlength())]
// }
func parseRequest(requestInBytes []byte) Request {
	fmt.Println("Длина переданного буфера в parseRequest", len(requestInBytes))

	// parse header
	parsedHeader := parseHeader(requestInBytes)

	//parse question section
	parsedQuestionSectionArray := parseQuestionSection(requestInBytes)

	fmt.Println("parsedQuestionSectionArray длина", uint16(len(parsedQuestionSectionArray)))
	fmt.Println("parsedHeader", parsedHeader.setDataToByteArray())

	fmt.Println("=======")
	fmt.Println("Отпарсенный массив лейблов")
	for _, qs := range parsedQuestionSectionArray {
		fmt.Println("Лейбл: ", qs.label)
	}
	return Request{header: parsedHeader, questionSection: parsedQuestionSectionArray}
}

func parseHeader(requestInBytes []byte) HeaderOptions {
	fmt.Println("Длина переданного буфера в parseHeader", len(requestInBytes))

	//Packet Identifier (ID)
	// ID assigned to query packets. Response packets must reply with the same ID
	id := requestInBytes[:2]
	fmt.Println("id: ", binary.BigEndian.Uint16(id))
	fmt.Println("id in bytes: ", id)

	// allocate a byte for the next 5 options: qr, opCode, aa, tc, rd
	qrOpCodeAaTcRdRequest := requestInBytes[2]
	qrOpCodeAaTcRdResponse := byte(0)

	// Query/Response Indicator (QR)
	// 1 for a reply packet, 0 for a question packet.
	qrMask := byte(0b10000000)
	qr := qrOpCodeAaTcRdRequest & qrMask
	qrOpCodeAaTcRdResponse |= qr

	// Operation Code (OPCODE)
	// Specifies the kind of query in a message.
	opCodeMask := byte(0b01111000)
	opCode := qrOpCodeAaTcRdRequest & opCodeMask
	qrOpCodeAaTcRdResponse |= opCode

	// Authoritative Answer (AA)
	// 1 if the responding server "owns" the domain queried, i.e., it's authoritative.
	aaMask := byte(0b00000100)
	aa := qrOpCodeAaTcRdRequest & aaMask
	qrOpCodeAaTcRdResponse |= aa

	// Truncation (TC)
	// 1 if the message is larger than 512 bytes. Always 0 in UDP responses.
	tcMask := byte(0b00000010)
	tc := qrOpCodeAaTcRdRequest & tcMask
	qrOpCodeAaTcRdResponse |= tc

	// Recursion Desired (RD)
	// Sender sets this to 1 if the server should recursively resolve this query, 0 otherwise.
	rdMask := byte(0b00000001)
	rd := qrOpCodeAaTcRdRequest & rdMask
	qrOpCodeAaTcRdResponse |= rd

	// extract value for 3 options: ra, z rcode
	raZRcodeRequest := requestInBytes[3]

	// allocate a byte for the next 3 options: ra, z rcode
	raZRcodeResponse := byte(0)

	//Recursion Available (RA)
	// Server sets this to 1 to indicate that recursion is available.
	raMask := byte(0b10000000)
	ra := raZRcodeRequest & raMask
	raZRcodeResponse |= ra

	//Reserved (Z)	3 bits
	// Used by DNSSEC queries. At inception, it was reserved for future use.
	zMask := byte(0b01110000)
	z := raZRcodeRequest & zMask
	raZRcodeResponse |= z

	// Response Code (RCODE)	4 bits
	// Response code indicating the status of the response.
	//	0 (no error).

	// set rcode based on opCode value
	// 0 - standard query
	rcodeMask := byte(0b00001111)
	if opCode != 0 {
		raZRcodeResponse |= rcodeMask // if error
	}

	// Question Count (QDCOUNT)
	// Number of questions in the Question section.
	qdcount := binary.BigEndian.Uint16(requestInBytes[4:6])

	return HeaderOptions{id: id, qrOpCodeAaTcRd: qrOpCodeAaTcRdResponse, raZRcode: raZRcodeResponse, qdcount: qdcount, ancount: 0, nscount: 0, arcount: 0}
}

// TODO: вместо массива сделать dictionary: offset + label
func parseLabels(requestInBytes []byte, offset int) ([]string, int) {
	nullByte := byte(0)
	var labelArray []string

	for requestInBytes[offset] != nullByte {
		//fmt.Println("==============")
		//fmt.Printf("requestInBytes[%d] Размер лейбла: %v\n", offset, requestInBytes[offset])

		pointerMask := byte(0b11000000)

		// check for a pointer
		if (requestInBytes[offset] & pointerMask) == pointerMask {
			offsetMask := byte(0b00111111)
			newOffsetInBytes := []byte{requestInBytes[offset] & offsetMask, requestInBytes[offset+1]}
			newOffset := int(binary.BigEndian.Uint16(newOffsetInBytes))
			result, _ := parseLabels(requestInBytes, newOffset)
			labelArray = append(labelArray, result...)
			offset += 2
			return labelArray, offset
		}

		// check label's length
		labelLength := int(requestInBytes[offset])
		offset += 1

		//fmt.Println("Offset после размера контента", offset)

		// check label's content
		content := requestInBytes[offset : offset+labelLength]
		offset += labelLength
		//fmt.Println(" Label: ", string(content))
		//fmt.Println("Offset после 1 байта с размером контекта и самим контентом", offset)
		labelArray = append(labelArray, string(content))
	}

	// skip of nullByte
	offset += 1

	return labelArray, offset
}

func parseQuestionSection(requestInBytes []byte) []QuestionSection {
	//  a sequence of labels (single byte of size + content) ending in a zero octet
	//  a pointer
	//  a sequence of labels ending with a pointer
	offset := 12 // 12 - header size
	var parsedQuestionSectionArray []QuestionSection

	for offset < len(requestInBytes) {
		//fmt.Println("=========")
		//fmt.Println("Offset в начале цикла", offset)
		//fmt.Println("Что лежит по данному элементу массива", int(requestInBytes[offset]))
		labelArray, newOffset := parseLabels(requestInBytes, offset)
		offset = newOffset

		typeLength := 2
		classLength := 2

		// parse type
		qType := requestInBytes[offset:(offset + typeLength)]
		offset += typeLength

		// parse class
		qClass := requestInBytes[offset:(offset + classLength)]
		offset += classLength

		parsedQuestionSectionArray = append(parsedQuestionSectionArray, QuestionSection{labelArray, qType, qClass})
	}

	return parsedQuestionSectionArray
}

func setHeader(headerOptions HeaderOptions, numberOfQestionsSections int, numberOfAnswerSections int) HeaderOptions {

	qrOpCodeAaTcRdResponse := headerOptions.qrOpCodeAaTcRd

	if numberOfAnswerSections > 0 {
		qrMask := byte(0b10000000)
		qrOpCodeAaTcRdResponse |= qrMask
	}

	// Operation Code (OPCODE)
	// Specifies the kind of query in a message.
	//opCodeMask := byte(0b01111000)
	//opCode := headerOptions.qrOpCodeAaTcRd & opCodeMask

	// hardcode values
	// Recursion Available (RA)
	// Server sets this to 1 to indicate that recursion is available.

	// Reserved (Z)
	//Used by DNSSEC queries. At inception, it was reserved for future use.

	// Authority Record Count (NSCOUNT)
	// Number of records in the Authority section.

	// Additional Record Count (ARCOUNT)
	// Number of records in the Additional section.
	//ra, z, nscount, arcount := false, byte(0), uint16(0), uint16(0)
	nscount, arcount := uint16(0), uint16(0)

	raZRcodeRequest := headerOptions.raZRcode

	// allocate a byte for the next 3 options: ra, z, rcode
	raZRcodeResponse := byte(0)

	// Recursion Available (RA)
	// Server sets this to 1 to indicate that recursion is available.
	raMask := byte(0b10000000)
	ra := raZRcodeRequest & raMask
	raZRcodeResponse |= ra

	// Reserved (Z)
	// Used by DNSSEC queries. At inception, it was reserved for future use.
	zMask := byte(0b01110000)
	z := raZRcodeRequest & zMask
	raZRcodeResponse |= z

	// Response Code (RCODE)
	// 0 (no error) if OPCODE is 0 (standard query) else 4 (not implemented)
	//rcodeMask := byte(0b00001111)

	opCodeMask := byte(0b01111000)
	opCode := headerOptions.qrOpCodeAaTcRd & opCodeMask

	if opCode != 0 {
		raZRcodeResponse |= byte(0b00000100)
	}

	return HeaderOptions{
		id:             headerOptions.id,
		qrOpCodeAaTcRd: qrOpCodeAaTcRdResponse,
		raZRcode:       raZRcodeResponse,
		qdcount:        uint16(numberOfQestionsSections),
		ancount:        uint16(numberOfAnswerSections),
		nscount:        nscount,
		arcount:        arcount,
	}
}

func setAnswerSection(questionsSection QuestionSection) AnswerSection {

	// hardcode value for ttl - any
	ttl := uint32(60)

	// add rlength of IP info
	// hardcode value - 4
	IPVersion := uint16(4)

	// add IP info
	// hardcode value - "80.156.23.56"
	data := setIPIntoBytes()

	return AnswerSection{questionSection: questionsSection, ttl: ttl, rlength: IPVersion, rdata: data}
}

func setIPIntoBytes() []byte {

	// hardcode ip address
	ip := "8.8.8.8"

	parserdId := strings.Split(ip, ".")
	var ipInByte []byte
	for _, ipPart := range parserdId {
		num, err := strconv.Atoi(ipPart)
		if err != nil {
			fmt.Println("Error:", err)
			os.Exit(1)
		}
		ipInByte = append(ipInByte, uint8(num))
	}
	return ipInByte
}
