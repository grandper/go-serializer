package codec_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/grandper/go-serializer/internal/codec"
	"github.com/grandper/go-serializer/internal/fixture"
)

func TestByteCodec(t *testing.T) {
	byteCodec := codec.NewByteCodec(fixture.User{})

	ValidateCodec(t, byteCodec)

	t.Run("should fail to encode unsupported type", func(t *testing.T) {
		_, err := byteCodec.Encode(make(chan int))
		require.ErrorIs(t, err, codec.ErrCannotEncodeStruct)
	})

	t.Run("should fail to decode invalid data", func(t *testing.T) {
		var user fixture.User
		err := byteCodec.Decode([]byte(`foobar`), &user)
		require.ErrorIs(t, err, codec.ErrCannotDecodeStruct)
	})
}
