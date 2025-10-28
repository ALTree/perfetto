package protoencoder

import "encoding/binary"

// Encoder is a minimal protobuf wire format encoder
type Encoder struct {
	buf []byte
}

// NewEncoder creates a new encoder with an empty buffer
func NewEncoder() *Encoder {
	return &Encoder{buf: make([]byte, 0, 1024)}
}

// Bytes returns the encoded bytes
func (e *Encoder) Bytes() []byte {
	return e.buf
}

// Reset clears the buffer for reuse
func (e *Encoder) Reset() {
	e.buf = e.buf[:0]
}

// WriteVarint encodes and writes a varint
func (e *Encoder) WriteVarint(v uint64) {
	for v >= 0x80 {
		e.buf = append(e.buf, byte(v)|0x80)
		v >>= 7
	}
	e.buf = append(e.buf, byte(v))
}

// WriteSVarint encodes and writes a signed varint using zigzag encoding
func (e *Encoder) WriteSVarint(v int64) {
	ux := uint64(v) << 1
	if v < 0 {
		ux = ^ux
	}
	e.WriteVarint(ux)
}

// WriteTag writes a field tag (field_number << 3 | wire_type)
func (e *Encoder) WriteTag(fieldNum uint32, wireType uint32) {
	e.WriteVarint(uint64(fieldNum<<3 | wireType))
}

// WriteUint64 writes an optional uint64 field (varint wire type)
func (e *Encoder) WriteUint64(fieldNum uint32, v *uint64) {
	if v != nil {
		e.WriteTag(fieldNum, 0) // wire type 0 = varint
		e.WriteVarint(*v)
	}
}

// WriteUint32 writes an optional uint32 field (varint wire type)
func (e *Encoder) WriteUint32(fieldNum uint32, v *uint32) {
	if v != nil {
		e.WriteTag(fieldNum, 0) // wire type 0 = varint
		e.WriteVarint(uint64(*v))
	}
}

// WriteInt64 writes an optional int64 field (varint wire type, NOT zigzag)
func (e *Encoder) WriteInt64(fieldNum uint32, v *int64) {
	if v != nil {
		e.WriteTag(fieldNum, 0) // wire type 0 = varint
		// int64 uses regular varint, can be negative but wastes space
		e.WriteVarint(uint64(*v))
	}
}

// WriteInt32 writes an optional int32 field (varint wire type, NOT zigzag)
func (e *Encoder) WriteInt32(fieldNum uint32, v *int32) {
	if v != nil {
		e.WriteTag(fieldNum, 0) // wire type 0 = varint
		// int32 uses regular varint, can be negative but wastes space
		e.WriteVarint(uint64(*v))
	}
}

// WriteSInt64 writes an optional sint64 field (varint wire type WITH zigzag)
func (e *Encoder) WriteSInt64(fieldNum uint32, v *int64) {
	if v != nil {
		e.WriteTag(fieldNum, 0) // wire type 0 = varint
		e.WriteSVarint(*v)
	}
}

// WriteSInt32 writes an optional sint32 field (varint wire type WITH zigzag)
func (e *Encoder) WriteSInt32(fieldNum uint32, v *int32) {
	if v != nil {
		e.WriteTag(fieldNum, 0) // wire type 0 = varint
		e.WriteSVarint(int64(*v))
	}
}

// WriteBool writes an optional bool field (varint wire type)
func (e *Encoder) WriteBool(fieldNum uint32, v *bool) {
	if v != nil {
		e.WriteTag(fieldNum, 0) // wire type 0 = varint
		if *v {
			e.buf = append(e.buf, 1)
		} else {
			e.buf = append(e.buf, 0)
		}
	}
}

// WriteString writes an optional string field (length-delimited wire type)
func (e *Encoder) WriteString(fieldNum uint32, s *string) {
	if s != nil {
		e.WriteTag(fieldNum, 2) // wire type 2 = length-delimited
		e.WriteVarint(uint64(len(*s)))
		e.buf = append(e.buf, []byte(*s)...)
	}
}

// WriteBytes writes an optional bytes field (length-delimited wire type)
func (e *Encoder) WriteBytes(fieldNum uint32, b []byte) {
	if b != nil {
		e.WriteTag(fieldNum, 2) // wire type 2 = length-delimited
		e.WriteVarint(uint64(len(b)))
		e.buf = append(e.buf, b...)
	}
}

// WriteFixed64 writes an optional fixed64 field (64-bit wire type)
func (e *Encoder) WriteFixed64(fieldNum uint32, v *uint64) {
	if v != nil {
		e.WriteTag(fieldNum, 1) // wire type 1 = 64-bit
		var buf [8]byte
		binary.LittleEndian.PutUint64(buf[:], *v)
		e.buf = append(e.buf, buf[:]...)
	}
}

// WriteFixed32 writes an optional fixed32 field (32-bit wire type)
func (e *Encoder) WriteFixed32(fieldNum uint32, v *uint32) {
	if v != nil {
		e.WriteTag(fieldNum, 5) // wire type 5 = 32-bit
		var buf [4]byte
		binary.LittleEndian.PutUint32(buf[:], *v)
		e.buf = append(e.buf, buf[:]...)
	}
}

// WriteMessage writes a nested message field
// The encoder function should write the message content to the encoder
func (e *Encoder) WriteMessage(fieldNum uint32, encFunc func(e *Encoder)) {
	if encFunc == nil {
		return
	}

	// Create a temporary encoder for the nested message
	nested := NewEncoder()
	encFunc(nested)
	nestedBytes := nested.Bytes()

	// Write the field tag and length, then the nested message bytes
	e.WriteTag(fieldNum, 2) // wire type 2 = length-delimited
	e.WriteVarint(uint64(len(nestedBytes)))
	e.buf = append(e.buf, nestedBytes...)
}

// WriteRepeatedVarint writes a repeated varint field (unpacked)
func (e *Encoder) WriteRepeatedVarint(fieldNum uint32, values []uint64) {
	for _, v := range values {
		e.WriteTag(fieldNum, 0)
		e.WriteVarint(v)
	}
}

// WriteRepeatedFixed64 writes a repeated fixed64 field (unpacked)
func (e *Encoder) WriteRepeatedFixed64(fieldNum uint32, values []uint64) {
	for _, v := range values {
		e.WriteTag(fieldNum, 1) // wire type 1 = 64-bit
		var buf [8]byte
		binary.LittleEndian.PutUint64(buf[:], v)
		e.buf = append(e.buf, buf[:]...)
	}
}

// WriteRepeatedMessage writes repeated message fields
func (e *Encoder) WriteRepeatedMessage(fieldNum uint32, encFuncs []func(e *Encoder)) {
	for _, encFunc := range encFuncs {
		e.WriteMessage(fieldNum, encFunc)
	}
}

// WriteEnum writes an enum field (varint wire type, NOT zigzag)
func (e *Encoder) WriteEnum(fieldNum uint32, v *int32) {
	if v != nil {
		e.WriteTag(fieldNum, 0) // wire type 0 = varint
		// Enums are encoded as int32 (regular varint, not zigzag)
		e.WriteVarint(uint64(*v))
	}
}
