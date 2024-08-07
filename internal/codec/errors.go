package codec

import "errors"

var (
	// ErrCannotEncodeStruct is returned when a struct cannot be encoded.
	ErrCannotEncodeStruct = errors.New("cannot encode the struct")

	// ErrCannotDecodeStruct is returned when a struct cannot be decoded.
	ErrCannotDecodeStruct = errors.New("cannot decode the struct")

	// ErrValueIsNotProtoMessage is returned when the value does not implement proto.Message.
	ErrValueIsNotProtoMessage = errors.New("value does not implement proto.Message")
)
