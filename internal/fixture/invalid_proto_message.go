package fixture

import (
	"errors"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/protoadapt"
)

// errInvalidProtoMessage is returned when marshaling an InvalidProtoMessage.
var errInvalidProtoMessage = errors.New("invalid proto message")

// InvalidProtoMessage is a struct that does not conform to a valid proto.Message.
type InvalidProtoMessage struct{}

// Reset resets the message.
func (m *InvalidProtoMessage) Reset() {}

// String returns a string representation of the message.
func (m *InvalidProtoMessage) String() string {
	return "InvalidProtoMessage"
}

// ProtoMessage marks the struct as a proto.Message.
func (m *InvalidProtoMessage) ProtoMessage() {}

// Marshal marshals the message into bytes.
func (m *InvalidProtoMessage) Marshal() ([]byte, error) {
	return nil, errInvalidProtoMessage
}

// InvalidProtoMessageInstanceV2 is an instance of InvalidProtoMessage as a proto.Message.
var InvalidProtoMessageInstanceV2 = protoadapt.MessageV2Of(&InvalidProtoMessage{})

// Ensure InvalidProtoMessage implements proto.Message.
var _ proto.Message = InvalidProtoMessageInstanceV2
