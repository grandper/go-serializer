package codec

import (
	"encoding/json"
	"fmt"
)

// JSONCodec for encoding and decoding structs in JSON.
type JSONCodec struct{}

// NewJSONCodec creates a new JSON codec.
func NewJSONCodec() JSONCodec {
	return JSONCodec{}
}

// Encode encodes a struct into JSON data.
func (c JSONCodec) Encode(v any) ([]byte, error) {
	result, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrCannotEncodeStruct, err)
	}
	return result, nil
}

// Decode decodes JSON data into a struct.
func (c JSONCodec) Decode(data []byte, v any) error {
	err := json.Unmarshal(data, v)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrCannotDecodeStruct, err)
	}
	return nil
}

// EncodesJSON reports that the JSON codec produces valid JSON.
func (c JSONCodec) EncodesJSON() bool {
	return true
}

// JSONCodec implements the Codec interface.
var _ Codec = (*JSONCodec)(nil)
