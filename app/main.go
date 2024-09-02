package main

import (
	"encoding/binary"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"unsafe"
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
			if command == "--resolver" {
				forwardingAddress := os.Args[2]

				answerBuf := make([]byte, 512)
				headerSize := 12
				bytesFromRead := 0
				offset := 0

				questionSectionSize := []int{}
				answerSectionSize := []int{}

				headerForward := parsedRequest.header

				answerSectionInBytes := []byte{}

				for _, questionSection := range parsedRequest.questionSection {

					questionSectionSize = append(questionSectionSize, questionSection.length())

					//request := Request{header: parsedRequest.header, questionSection: []QuestionSection{questionSection}}

					header := setHeader(headerForward, 1, 0)

					headerInBytes := header.setDataToByteArray()
					fmt.Println("[FORWARD] Header for forward in bytes", headerInBytes)
					questionSectionInBytes := questionSection.setDataToByteArray()

					request := append(headerInBytes, questionSectionInBytes...)
					fmt.Println("[FORWARD] Request for forward in bytes", request)

					//_, err = udpConn.WriteToUDP(request, source)
					//p :=  make([]byte, 512)
					updConnForward, err := net.Dial("udp", forwardingAddress)
					if err != nil {
						fmt.Printf("Some error %v", err)
						return
					}

					_, errWrite := updConnForward.Write(request)

					if errWrite != nil {
						fmt.Printf("Some error %v", err)
						return
					}

					bytesRead, errRead := updConnForward.Read(answerBuf[offset:])
					answerSectionSize = append(answerSectionSize, bytesRead-headerSize-questionSection.length())
					bytesFromRead += bytesRead
					answerSectionInBytes = append(answerSectionInBytes, answerBuf[len(request):]...)
					if errRead != nil {
						fmt.Printf("Some error %v", err)
						return
					}

					offset += len(answerBuf)
					//answerBuf = append(answerBuf, buf[:size]...)
				}
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

			} else {
				fmt.Fprintf(os.Stderr, "Command %s hasn't been implemented", command)
				os.Exit(1)
			}
		} else {
			response := setDNSResponse(parsedRequest)

			_, err = udpConn.WriteToUDP(response, source)
			if err != nil {
				fmt.Println("Failed to send response:", err)
			}
		}

	}

	sendUDPPacket(receivingAddress)
}

func sendRequest(conn *net.UDPConn, addr string, request []byte) {
	udpAddr, err := net.ResolveUDPAddr("udp", addr)

	_, errWrite := conn.WriteToUDP(request, udpAddr)
	if errWrite != nil {
		fmt.Printf("Couldn't send response %v", err)
	}
}

func sendUDPPacket(receivingAddress string) {

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

type Request struct {
	header          HeaderOptions
	questionSection []QuestionSection
}

type HeaderOptions struct {
	id             []byte
	qrOpCodeAaTcRd byte
	raZRcode       byte
	qdcount        uint16
	ancount        uint16
	nscount        uint16
	arcount        uint16
}

func (h *HeaderOptions) setDataToByteArray() []byte {
	// allocate array of 12 bytes
	header := make([]byte, 12)

	// white id into header
	copy(header[0:2], h.id)

	// write 5 options: qr, opCode, aa, tc, rd into header
	header[2] = h.qrOpCodeAaTcRd

	header[3] = h.raZRcode

	// white the last 4 options: qdcount, ancount, nscount, arcount into header
	binary.BigEndian.PutUint16(header[4:], h.qdcount)
	binary.BigEndian.PutUint16(header[6:], h.ancount)
	binary.BigEndian.PutUint16(header[8:], h.nscount)
	binary.BigEndian.PutUint16(header[10:], h.arcount)

	return header
}

type QuestionSection struct {
	label  LabelArray
	qType  []byte
	qClass []byte
}

func (q *QuestionSection) setDataToByteArray() []byte {

	questionSection := make([]byte, q.length())

	var label []byte
	labelLength := 0

	for _, labelPart := range q.label {

		labelPartLength := len(labelPart)            // размер стринг
		label = append(label, byte(labelPartLength)) // положить размер в 1 байт
		labelLength += 1

		label = append(label, []byte(labelPart)...) // сам лейбл
		labelLength += labelPartLength
	}
	label = append(label, byte(0)) // добавить null byte
	labelLength += 1

	copy(questionSection[0:labelLength], label)

	copy(questionSection[labelLength:labelLength+2], q.qType)

	copy(questionSection[labelLength+2:], q.qClass)

	return questionSection
}
func (q *QuestionSection) length() int {
	// 1 байт под размер
	// N байт под сам лейбл
	// 1 байт под  null byte в конце
	labelLength := 0
	for _, labelPart := range q.label {
		labelLength += 1              // 1 байт под размер
		labelLength += len(labelPart) // сам лейбл
	}
	labelLength += 1 // null byte в конце
	return labelLength + len(q.qType) + len(q.qClass)
}

type AnswerSection struct {
	questionSection QuestionSection
	ttl             uint32
	rlength         uint16
	rdata           []byte
}

func (a *AnswerSection) setDataToByteArray() []byte {
	// allocate array of bytes
	answerSection := make([]byte, a.length())
	fmt.Println("размер выделенного массива под ответ: ", len(answerSection))

	// write question section into answerSection
	copy(answerSection[0:], a.questionSection.setDataToByteArray())
	fmt.Println("первая часть в ответе question section: ", len(a.questionSection.setDataToByteArray()))
	ttlLength := 4
	ttlStartIndex := a.questionSection.length()
	binary.BigEndian.PutUint32(answerSection[ttlStartIndex:], a.ttl)

	rdlengthSize := 2
	rdlengthStartIndex := ttlStartIndex + ttlLength
	binary.BigEndian.PutUint16(answerSection[rdlengthStartIndex:], a.rlength)

	rdataStartIndex := rdlengthStartIndex + rdlengthSize
	copy(answerSection[rdataStartIndex:], a.rdata)

	return answerSection
}
func (a *AnswerSection) length() int {
	ttlSize := int(unsafe.Sizeof(a.ttl))          // TODO: 4
	rdlengthSize := int(unsafe.Sizeof(a.rlength)) // TODO: 2
	fmt.Println("a.questionSection.length()", a.questionSection.length())
	fmt.Println("ttlSize", ttlSize)
	fmt.Println("rdlengthSize", rdlengthSize)
	fmt.Println("a.rdata", len(a.rdata))

	return a.questionSection.length() + ttlSize + rdlengthSize + len(a.rdata)
}

//  response[0]=1234 >> 8; response[1]= 1234 & 0xFF;
// 1234 - это число, наверно подефолту на 32 бита, нам нужно его записать в 16 бит, 2 байта, пишем сначала один байт (>>8), затем второй (& 0xFF - отбрасываем лишние байты)
//  потому что 1234 - это 16 бит и ты делишь на 256 ( сдвиг на 8 битов)

type LabelArray []string
