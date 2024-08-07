package codec

import (
	"fmt"

	"google.golang.org/protobuf/proto"
)

// ProtoCodec for encoding and decoding structs.
type ProtoCodec struct{}

// NewProtoCodec creates a new proto codec.
func NewProtoCodec() ProtoCodec {
	return ProtoCodec{}
}

// Encode encodes a struct into bytes.
func (c ProtoCodec) Encode(v any) ([]byte, error) {
	msg, ok := v.(proto.Message)
	if !ok {
		return nil, ErrValueIsNotProtoMessage
	}
	data, err := proto.Marshal(msg)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrCannotEncodeStruct, err)
	}
	return data, nil
}

// Decode decodes bytes into a struct.
func (c ProtoCodec) Decode(data []byte, v any) error {
	msg, ok := v.(proto.Message)
	if !ok {
		return ErrValueIsNotProtoMessage
	}
	if err := proto.Unmarshal(data, msg); err != nil {
		return fmt.Errorf("%w: %w", ErrCannotDecodeStruct, err)
	}
	return nil
}

// EncodesJSON reports that the proto codec produces binary (non-JSON) output.
func (c ProtoCodec) EncodesJSON() bool {
	return false
}

// ProtoCodec implements the Codec interface.
var _ Codec = (*ProtoCodec)(nil)
