package main

import (
	"encoding/binary"
	"fmt"
	"log"
	"net"
	"os"
	"unsafe"
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
			fmt.Println("Error receiving rdata:", err)
			break
		}

		receivedData := buf[:size]
		fmt.Println("Получен буфер: ", string(receivedData))

		// parse request
		//id, qrOpCodeAaTcRd, rcode, _, _, _, questionLength := parseRequest(receivedData)
		parsedRequest := parseRequest(receivedData)

		// response parsedRequest
		// set header
		header := setHeader(parsedRequest.header)
		response := header.setDataToByteArray()

		// set question section
		questionSection := parsedRequest.questionSection

		// set answer section
		var answerSectionArray []AnswerSection
		for _, questionSection := range questionSection {
			answerSection := setAnswerSection(questionSection)
			answerSectionArray = append(answerSectionArray, answerSection)
			response = append(response, questionSection.setDataToByteArray()...)
		}

		// set parsedRequest

		for _, answerSection := range answerSectionArray {
			response = append(response, answerSection.setDataToByteArray()...)
		}

		_, err = udpConn.WriteToUDP(response, source)
		if err != nil {
			fmt.Println("Failed to send response:", err)
		}
	}
}

// TODO: rdlength parser questionsection

// func setQuestionSection(receivedData []byte, parsedRequest Request) []byte {
// headerLength := 12
// return receivedData[headerLength:(headerLength + parsedRequest.questionSection.rdlength())]
// }
func parseRequest(requestInBytes []byte) Request {
	fmt.Println("Длина переданного буфера в parseRequest", len(requestInBytes))

	// parse header
	headerLength := 12
	header := requestInBytes[:headerLength]
	parsedHeader := parseHeader(header)
	//id, qrOpCodeAaTcRd, rcode := parsedHeader.id, parsedHeader.qrOpCodeAaTcRd, parsedHeader.rcode

	//parse question section

	parsedQuestionSectionArray := parseQuestionSection(requestInBytes)
	fmt.Println("=======")
	fmt.Println("Отпарсенный массив лейблов")

	for _, qs := range parsedQuestionSectionArray {
		fmt.Println("Лейбл: ", string(qs.label))
	}

	fmt.Println()
	//label, qType, qClass, questionLength := parsedQuestionSection.label, parsedQuestionSection.qType, parsedQuestionSection.qClass, parsedQuestionSection.questionLength

	return Request{header: parsedHeader, questionSection: parsedQuestionSectionArray}
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
	return HeaderOptions{id: id, qrOpCodeAaTcRd: qrOpCodeAaTcRd, rcode: rcode}
}

// TODO: вместо массива сделать dictinary: offset + label
func parseLabels(requestInBytes []byte, offset int) ([]byte, int) {
	nullByte := byte(0)
	var labelArray []byte

	for requestInBytes[offset] != nullByte {
		fmt.Println("==============")
		fmt.Printf("requestInBytes[%d] Размер лейбла: %v\n", offset, requestInBytes[offset])

		// check if pointer
		pointerMask := byte(0b11000000)

		if (requestInBytes[offset] & pointerMask) == pointerMask {
			offsetMask := byte(0b00111111)
			newOffsetInBytes := []byte{requestInBytes[offset] & offsetMask, requestInBytes[offset+1]}
			newOffset := int(binary.BigEndian.Uint16(newOffsetInBytes))
			result, _ := parseLabels(requestInBytes, newOffset)
			labelArray = append(labelArray, result...)
			offset += 2
			return labelArray, offset

		}
		labelLength := int(requestInBytes[offset])
		offset += 1

		//fmt.Println("Offset после размера контента", offset)

		content := requestInBytes[offset : offset+labelLength]
		offset += labelLength
		fmt.Println(" Label: ", string(content))
		fmt.Println("Offset после 1 байта с размером контекта и самим контентом", offset)
		labelArray = append(labelArray, content...)

	}
	labelArray = append(labelArray, nullByte)
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
		startIndex := offset
		labelArray, newOffset := parseLabels(requestInBytes, startIndex)
		offset = newOffset

		//labelLength := offset - startIndex
		typeLength := 2
		classLength := 2

		//label := requestInBytes[startIndex:labelLength]
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

func setHeader(headerOptions HeaderOptions) HeaderOptions {

	rcode := headerOptions.rcode

	// hardcode values
	ra, z, qdcount, ancount, nscount, arcount := false, byte(0), uint16(1), uint16(1), uint16(0), uint16(0)

	// allocate a byte for the next 3 options: ra, z, rcode
	raZRcode := byte(0)

	raMask := byte(0b10000000)

	if ra {
		raZRcode |= raMask
	}

	zMask := byte(0b01110000) & (z << 3)

	if z > 0 {
		raZRcode |= zMask
	}

	rcodeMask := byte(0b00001111)

	if rcode > 0 {
		raZRcode |= rcodeMask & rcode
	}

	return HeaderOptions{
		id:             headerOptions.id,
		qrOpCodeAaTcRd: headerOptions.qrOpCodeAaTcRd,
		rcode:          headerOptions.rcode,
		raZRcode:       raZRcode,
		qdcount:        qdcount,
		ancount:        ancount,
		nscount:        nscount,
		arcount:        arcount,
	}
}

func setAnswerSection(questionsSection QuestionSection) AnswerSection {

	//hardcode value for TTL - any
	ttl := uint32(60)

	// add rdlength of IP info
	// hardcode value - 4
	IPVersion := uint16(4)

	// add IP info
	// hardcode value - "80.156.23.56"
	data := setIPIntoBytes()

	return AnswerSection{label: questionsSection.label, qType: questionsSection.qType, qClass: questionsSection.qClass, TTL: ttl, rdlength: IPVersion, rdata: data}
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
	questionSection []QuestionSection
}

type HeaderOptions struct {
	id             []byte
	qrOpCodeAaTcRd byte
	rcode          byte
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
	label  []byte
	qType  []byte
	qClass []byte
}

func (q *QuestionSection) setDataToByteArray() []byte {
	// allocate array of 12 bytes

	questionSection := make([]byte, q.length())

	// white id into questionSection
	labelLength := len(q.label)
	copy(questionSection[0:labelLength], q.label)

	copy(questionSection[labelLength:labelLength+2], q.qType)

	copy(questionSection[labelLength+2:], q.qClass)

	return questionSection
}
func (q *QuestionSection) length() int {
	return len(q.label) + len(q.qType) + len(q.qClass)
}

type AnswerSection struct {
	label    []byte
	qType    []byte
	qClass   []byte
	TTL      uint32
	rdlength uint16
	rdata    []byte
}

func (a *AnswerSection) setDataToByteArray() []byte {
	// allocate array of 12 bytes
	answerSection := make([]byte, a.length())

	// white id into answerSection
	labelLength := len(a.label)
	labelStartIndex := 0
	copy(answerSection[labelStartIndex:], a.label)

	qTypeLength := 2
	qTypeStartIndex := labelStartIndex + labelLength
	copy(answerSection[qTypeStartIndex:], a.qType)

	qClassLength := 2
	qClassStartIndex := qTypeStartIndex + qTypeLength
	copy(answerSection[qClassStartIndex:], a.qClass)

	ttlLength := 4
	ttlStartIndex := qClassStartIndex + qClassLength
	binary.BigEndian.PutUint32(answerSection[ttlStartIndex:], a.TTL)

	rdlengthSize := 2
	rdlengthStartIndex := ttlStartIndex + ttlLength
	binary.BigEndian.PutUint16(answerSection[rdlengthStartIndex:], a.rdlength)

	rdataStartIndex := rdlengthStartIndex + rdlengthSize
	copy(answerSection[rdataStartIndex:], a.rdata)

	return answerSection
}
func (a *AnswerSection) length() int {
	ttlSize := int(unsafe.Sizeof(a.TTL))           // TODO: 4
	rdlengthSize := int(unsafe.Sizeof(a.rdlength)) // TODO: 2

	return len(a.label) + len(a.qType) + len(a.qClass) + ttlSize + rdlengthSize + len(a.rdata)
}

//  response[0]=1234 >> 8; response[1]= 1234 & 0xFF;
// 1234 - это число, наверно подефолту на 32 бита, нам нужно его записать в 16 бит, 2 байта, пишем сначала один байт (>>8), затем второй (& 0xFF - отбрасываем лишние байты)
//  потому что 1234 - это 16 бит и ты делишь на 256 ( сдвиг на 8 битов)
