package main

import "encoding/binary"

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
