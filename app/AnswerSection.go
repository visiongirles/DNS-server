package main

import (
	"encoding/binary"
	"fmt"
	"unsafe"
)

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
