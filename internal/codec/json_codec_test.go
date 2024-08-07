package codec_test

import (
	"testing"

	"github.com/grandper/go-serializer/internal/codec"
	"github.com/grandper/go-serializer/internal/fixture"
	"github.com/stretchr/testify/assert"
)

func TestJSONCodec(t *testing.T) {
	jsonCodec := codec.NewJSONCodec()
	const testSerializedUser1 = `{"Username":"johndoe","Age":24,"Height":1.82,"Email":"john.doe@gmail.com","VerifiedEmail":false,"Address":{"Street":"Apple Park Way","Number":1,"City":"Cubertino","State":"California","PostalCode":95014,"Country":"United States"},"CreationTime":"2024-08-24T10:25:59.000000044Z"}`

	ValidateCodec(t, jsonCodec)

	t.Run("should encode a struct", func(t *testing.T) {
		serializedUser1, err := jsonCodec.Encode(fixture.User1)
		assert.NoError(t, err)
		assert.Equal(t, testSerializedUser1, string(serializedUser1))
	})

	t.Run("should fail to encode unsupported type", func(t *testing.T) {
		_, err := jsonCodec.Encode(make(chan int))
		assert.ErrorIs(t, err, codec.ErrCannotEncodeStruct)
	})

	t.Run("should decode data", func(t *testing.T) {
		var user fixture.User
		err := jsonCodec.Decode([]byte(testSerializedUser1), &user)
		assert.NoError(t, err)
		assert.Equal(t, fixture.User1, user)
	})

	t.Run("should fail to decode invalid JSON data", func(t *testing.T) {
		var user fixture.User
		err := jsonCodec.Decode([]byte(`{`), &user)
		assert.ErrorIs(t, err, codec.ErrCannotDecodeStruct)
	})
}
