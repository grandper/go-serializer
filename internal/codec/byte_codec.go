package codec

import (
	"bytes"
	"encoding/gob"
	"fmt"
)

// ByteCodec for encoding and decoding structs.
type ByteCodec struct{}

// NewByteCodec creates a new byte codec.
func NewByteCodec(eventTypesToRegister ...any) ByteCodec {
	for _, aStruct := range eventTypesToRegister {
		gob.Register(aStruct)
	}
	return ByteCodec{}
}

// Encode encodes a struct into bytes.
func (c ByteCodec) Encode(v any) ([]byte, error) {
	var buffer bytes.Buffer
	enc := gob.NewEncoder(&buffer)
	if err := enc.Encode(v); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrCannotEncodeStruct, err)
	}
	return buffer.Bytes(), nil
}

// Decode decodes bytes into a struct.
func (c ByteCodec) Decode(data []byte, v any) error {
	buffer := bytes.NewBuffer(data)
	dec := gob.NewDecoder(buffer)
	if err := dec.Decode(v); err != nil {
		return fmt.Errorf("%w: %w", ErrCannotDecodeStruct, err)
	}
	return nil
}

// EncodesJSON reports that the byte codec produces binary (non-JSON) output.
func (c ByteCodec) EncodesJSON() bool {
	return false
}

// ByteCodec implements the Codec interface.
var _ Codec = (*ByteCodec)(nil)
