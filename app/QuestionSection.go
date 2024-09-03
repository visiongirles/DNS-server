package main

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
