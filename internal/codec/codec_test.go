package codec_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/grandper/go-serializer/internal/codec"
	"github.com/grandper/go-serializer/internal/fixture"
)

func ValidateCodec(t *testing.T, c codec.Codec) {
	t.Helper()

	t.Run("should get the same struct when encoding and decoding a struct", func(t *testing.T) {
		data, err := c.Encode(fixture.User1)
		require.NoError(t, err)

		var user fixture.User
		err = c.Decode(data, &user)
		require.NoError(t, err)

		assert.Equal(t, fixture.User1, user)
	})
}
