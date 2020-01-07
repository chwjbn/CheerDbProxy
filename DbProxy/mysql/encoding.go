package mysql

import (
	"bytes"
	"encoding/binary"
)

// This file contains the data encoding and decoding functions.

//
// Encoding methods.
//
// The same assumptions are made for all the encoding functions:
// - there is enough space to write the data in the buffer. If not, we
// will panic with out of bounds.
// - all functions start writing at 'pos' in the buffer, and return the next position.

// LenEncIntSize returns the number of bytes required to encode a
// variable-length integer.
func LenEncIntSize(i uint64) int {
	switch {
	case i < 251:
		return 1
	case i < 1<<16:
		return 3
	case i < 1<<24:
		return 4
	default:
		return 9
	}
}

// WriteLenEncInt write uint64 to []byte
func WriteLenEncInt(data []byte, pos int, i uint64) int {
	switch {
	case i < 251:
		data[pos] = byte(i)
		return pos + 1
	case i < 1<<16:
		data[pos] = 0xfc
		data[pos+1] = byte(i)
		data[pos+2] = byte(i >> 8)
		return pos + 3
	case i < 1<<24:
		data[pos] = 0xfd
		data[pos+1] = byte(i)
		data[pos+2] = byte(i >> 8)
		data[pos+3] = byte(i >> 16)
		return pos + 4
	default:
		data[pos] = 0xfe
		data[pos+1] = byte(i)
		data[pos+2] = byte(i >> 8)
		data[pos+3] = byte(i >> 16)
		data[pos+4] = byte(i >> 24)
		data[pos+5] = byte(i >> 32)
		data[pos+6] = byte(i >> 40)
		data[pos+7] = byte(i >> 48)
		data[pos+8] = byte(i >> 56)
		return pos + 9
	}
}

// AppendLenEncInt append LenEncInt []byte to data
func AppendLenEncInt(data []byte, i uint64) []byte {
	switch {
	case i <= 250:
		return append(data, byte(i))

	case i <= 0xffff:
		return append(data, 0xfc, byte(i), byte(i>>8))

	case i <= 0xffffff:
		return append(data, 0xfd, byte(i), byte(i>>8), byte(i>>16))

	case i <= 0xffffffffffffffff:
		return append(data, 0xfe, byte(i), byte(i>>8), byte(i>>16), byte(i>>24),
			byte(i>>32), byte(i>>40), byte(i>>48), byte(i>>56))
	}

	return data
}

// LenNullString return lenght Null terminated string
func LenNullString(value string) int {
	return len(value) + 1
}

// WriteNullString write NULL terminated strign to []byte
func WriteNullString(data []byte, pos int, value string) int {
	pos += copy(data[pos:], value)
	data[pos] = 0
	return pos + 1
}

func writeEOFString(data []byte, pos int, value string) int {
	pos += copy(data[pos:], value)
	return pos
}

// WriteByte write byte to []byte
func WriteByte(data []byte, pos int, value byte) int {
	data[pos] = value
	return pos + 1
}

// WriteUint16 write uint16 to []byte
func WriteUint16(data []byte, pos int, value uint16) int {
	data[pos] = byte(value)
	data[pos+1] = byte(value >> 8)
	return pos + 2
}

// AppendUint16 append uint16 to []byte
func AppendUint16(data []byte, n uint16) []byte {
	data = append(data, byte(n))
	data = append(data, byte(n>>8))
	return data
}

// WriteUint32 write uint32 to []byte
func WriteUint32(data []byte, pos int, value uint32) int {
	data[pos] = byte(value)
	data[pos+1] = byte(value >> 8)
	data[pos+2] = byte(value >> 16)
	data[pos+3] = byte(value >> 24)
	return pos + 4
}

// AppendUint32 append uint32 to []byte
func AppendUint32(data []byte, n uint32) []byte {
	data = append(data, byte(n))
	data = append(data, byte(n>>8))
	data = append(data, byte(n>>16))
	data = append(data, byte(n>>24))
	return data
}

// WriteUint64 write uint64 to []byte
func WriteUint64(data []byte, pos int, value uint64) int {
	data[pos] = byte(value)
	data[pos+1] = byte(value >> 8)
	data[pos+2] = byte(value >> 16)
	data[pos+3] = byte(value >> 24)
	data[pos+4] = byte(value >> 32)
	data[pos+5] = byte(value >> 40)
	data[pos+6] = byte(value >> 48)
	data[pos+7] = byte(value >> 56)
	return pos + 8
}

// AppendUint64 append uint64 to []byte
func AppendUint64(data []byte, n uint64) []byte {
	data = append(data, byte(n))
	data = append(data, byte(n>>8))
	data = append(data, byte(n>>16))
	data = append(data, byte(n>>24))
	data = append(data, byte(n>>32))
	data = append(data, byte(n>>40))
	data = append(data, byte(n>>48))
	data = append(data, byte(n>>56))
	return data
}

// LenEncStringSize  calculate length of lenenc_str
// https://dev.mysql.com/doc/internals/en/describing-packets.html#type-lenenc_str
func LenEncStringSize(value string) int {
	l := len(value)
	return LenEncIntSize(uint64(l)) + l
}

// WriteLenEncString write string to []byte, return pos
func WriteLenEncString(data []byte, pos int, value string) int {
	pos = WriteLenEncInt(data, pos, uint64(len(value)))
	return writeEOFString(data, pos, value)
}

// AppendLenEncStringBytes append bytes of len enc string  to data
func AppendLenEncStringBytes(data, b []byte) []byte {
	data = AppendLenEncInt(data, uint64(len(b)))
	data = append(data, b...)
	return data
}

// WriteZeroes write 0 to []byte
func WriteZeroes(data []byte, pos int, len int) int {
	for i := 0; i < len; i++ {
		data[pos+i] = 0
	}
	return pos + len
}

//
// Decoding methods.
//
// The same assumptions are made for all the decoding functions:
// - they return the decode data, the new position to read from, and ak 'ok' flag.
// - all functions start reading at 'pos' in the buffer, and return the next position.
//

// ReadByte read one byte from []byte
func ReadByte(data []byte, pos int) (byte, int, bool) {
	if pos >= len(data) {
		return 0, 0, false
	}
	return data[pos], pos + 1, true
}

// ReadBytes read []byte from pos with sized size
func ReadBytes(data []byte, pos int, size int) ([]byte, int, bool) {
	if pos+size-1 >= len(data) {
		return nil, 0, false
	}
	return data[pos : pos+size], pos + size, true
}

// ReadBytesCopy returns a copy of the bytes in the packet.
// Useful to remember contents of ephemeral packets.
func ReadBytesCopy(data []byte, pos int, size int) ([]byte, int, bool) {
	if pos+size-1 >= len(data) {
		return nil, 0, false
	}
	result := make([]byte, size)
	copy(result, data[pos:pos+size])
	return result, pos + size, true
}

// ReadNullString read Null terminated string from []byte, return string,pos,if end.
func ReadNullString(data []byte, pos int) (string, int, bool) {
	end := bytes.IndexByte(data[pos:], 0)
	if end == -1 {
		return "", 0, false
	}
	return string(data[pos : pos+end]), pos + end + 1, true
}

// ReadUint16 read uint32 from []byte
func ReadUint16(data []byte, pos int) (uint16, int, bool) {
	if pos+1 >= len(data) {
		return 0, 0, false
	}
	return binary.LittleEndian.Uint16(data[pos : pos+2]), pos + 2, true
}

// ReadUint32 read uint32 from []byte
func ReadUint32(data []byte, pos int) (uint32, int, bool) {
	if pos+3 >= len(data) {
		return 0, 0, false
	}
	return binary.LittleEndian.Uint32(data[pos : pos+4]), pos + 4, true
}

// ReadUint64 read uint64 from []byte
func ReadUint64(data []byte, pos int) (uint64, int, bool) {
	if pos+7 >= len(data) {
		return 0, 0, false
	}
	return binary.LittleEndian.Uint64(data[pos : pos+8]), pos + 8, true
}

// ReadLenEncInt read info of len encoded int, return length, next pos(skip len self to data), is null, handle result
// https://dev.mysql.com/doc/internals/en/integer.html#packet-Protocol::FixedLengthInteger
func ReadLenEncInt(data []byte, pos int) (uint64, int, bool, bool) {
	isNull := false
	if pos >= len(data) {
		return 0, 0, isNull, false
	}
	switch data[pos] {
	// 251: NULL
	case 0xfb:
		isNull = true
		return 0, pos + 1, isNull, true
	case 0xfc:
		// Encoded in the next 2 bytes.
		if pos+2 >= len(data) {
			return 0, 0, isNull, false
		}
		return uint64(data[pos+1]) |
			uint64(data[pos+2])<<8, pos + 3, isNull, true
	case 0xfd:
		// Encoded in the next 3 bytes.
		if pos+3 >= len(data) {
			return 0, 0, isNull, false
		}
		return uint64(data[pos+1]) |
			uint64(data[pos+2])<<8 |
			uint64(data[pos+3])<<16, pos + 4, isNull, true
	case 0xfe:
		// Encoded in the next 8 bytes.
		if pos+8 >= len(data) {
			return 0, 0, isNull, false
		}
		return uint64(data[pos+1]) |
			uint64(data[pos+2])<<8 |
			uint64(data[pos+3])<<16 |
			uint64(data[pos+4])<<24 |
			uint64(data[pos+5])<<32 |
			uint64(data[pos+6])<<40 |
			uint64(data[pos+7])<<48 |
			uint64(data[pos+8])<<56, pos + 9, isNull, true
	}
	// 0-250
	return uint64(data[pos]), pos + 1, isNull, true
}

func readLenEncString(data []byte, pos int) (string, int, bool) {
	size, pos, _, ok := ReadLenEncInt(data, pos)
	if !ok {
		return "", 0, false
	}
	s := int(size)
	if pos+s-1 >= len(data) {
		return "", 0, false
	}
	return string(data[pos : pos+s]), pos + s, true
}

// return next pos、handle result
func skipLenEncString(data []byte, pos int) (int, bool) {
	size, pos, _, ok := ReadLenEncInt(data, pos)
	if !ok {
		return 0, false
	}
	s := int(size)
	if pos+s-1 >= len(data) {
		return 0, false
	}
	return pos + s, true
}

// ReadLenEncStringAsBytes read len encoded string, return []byte format, next pos, is null, handle result
func ReadLenEncStringAsBytes(data []byte, pos int) ([]byte, int, bool, bool) {
	size, pos, isNull, ok := ReadLenEncInt(data, pos)
	if !ok {
		return nil, 0, isNull, false
	}
	s := int(size)
	if pos+s-1 >= len(data) {
		return nil, 0, isNull, false
	}
	return data[pos : pos+s], pos + s, isNull, true
}
