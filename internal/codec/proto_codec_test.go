package codec_test

import (
	"testing"
	"time"

	"github.com/grandper/go-serializer/internal/codec"
	"github.com/grandper/go-serializer/internal/fixture"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/durationpb"
)

func TestProtoCodec(t *testing.T) {
	protoCodec := codec.NewProtoCodec()
	const testSerializedUser1 = `{"Username":"johndoe","Age":24,"Height":1.82,"Email":"john.doe@gmail.com","VerifiedEmail":false,"Address":{"Street":"Apple Park Way","Number":1,"City":"Cubertino","State":"California","PostalCode":95014,"Country":"United States"},"CreationTime":"2024-08-24T10:25:59.000000044Z"}`

	d := durationpb.New(5 * time.Second)

	t.Run("should encode and decode a proto.Message", func(t *testing.T) {
		serializedDuration, err := protoCodec.Encode(d)
		assert.NoError(t, err)

		var duration durationpb.Duration
		err = protoCodec.Decode(serializedDuration, &duration)
		assert.NoError(t, err)
		assert.True(t, proto.Equal(d, &duration))
	})

	t.Run("should fail to encode a non-proto.Message struct", func(t *testing.T) {
		serializedUser1, err := protoCodec.Encode(fixture.User1)
		assert.ErrorIs(t, err, codec.ErrValueIsNotProtoMessage)
		assert.Nil(t, serializedUser1)
	})

	t.Run("should fail to decode into a non-proto.Message struct", func(t *testing.T) {
		var user fixture.User
		err := protoCodec.Decode([]byte(testSerializedUser1), &user)
		assert.ErrorIs(t, err, codec.ErrValueIsNotProtoMessage)
	})

	t.Run("should fail to encode an invalid proto.Message", func(t *testing.T) {
		_, err := protoCodec.Encode(fixture.InvalidProtoMessageInstanceV2)
		assert.ErrorIs(t, err, codec.ErrCannotEncodeStruct)
	})

	t.Run("should fail to decode into an invalid proto.Message", func(t *testing.T) {
		invalidProtoMessage := &durationpb.Duration{}
		err := protoCodec.Decode([]byte{0x00, 0x01, 0x02}, invalidProtoMessage)
		assert.ErrorIs(t, err, codec.ErrCannotDecodeStruct)
	})
}
