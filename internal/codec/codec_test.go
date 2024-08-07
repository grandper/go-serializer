package codec_test

import (
	"testing"

	"github.com/grandper/go-serializer/internal/codec"
	"github.com/grandper/go-serializer/internal/fixture"
	"github.com/stretchr/testify/assert"
)

func ValidateCodec(t *testing.T, c codec.Codec) {
	t.Helper()

	t.Run("should get the same struct when encoding and decoding a struct", func(t *testing.T) {
		data, err := c.Encode(fixture.User1)
		assert.NoError(t, err)

		var user fixture.User
		err = c.Decode(data, &user)
		assert.NoError(t, err)

		assert.Equal(t, fixture.User1, user)
	})
}
