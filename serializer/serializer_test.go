package serializer_test

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/grandper/go-serializer/internal/fixture"
	"github.com/grandper/go-serializer/serializer"
)

func TestNewJSONSerializer(t *testing.T) {
	s := serializer.NewJSONSerializer(serializer.Register(fixture.User{}))

	validateSerializer(t, s)

	t.Run("should embed a readable JSON payload", func(t *testing.T) {
		data, err := s.Serialize(fixture.User1)
		require.NoError(t, err)

		// The JSON codec payload must be embedded verbatim (not base64-encoded), so
		// the envelope stays human-readable and is not double-encoded.
		assert.Contains(t, string(data), `"data":{`)
		assert.Contains(t, string(data), `"johndoe"`)
	})

	t.Run("should fail when the payload does not match the registered type", func(t *testing.T) {
		js := serializer.NewJSONSerializer(serializer.RegisterAs("user", fixture.User{}))

		// The envelope is well-formed and the type is registered, but the payload is
		// a JSON array that cannot be decoded into a User struct.
		envelope := `{"version":0,"type":"user","data":[1,2,3]}`
		_, err := js.Deserialize([]byte(envelope))
		require.ErrorIs(t, err, serializer.ErrFailedToDeserialize)
	})
}

func TestNewByteSerializer(t *testing.T) {
	s := serializer.NewByteSerializer(serializer.Register(fixture.User{}))

	validateSerializer(t, s)

	t.Run("should fail on a corrupt binary payload", func(t *testing.T) {
		bs := serializer.NewByteSerializer(serializer.RegisterAs("user", fixture.User{}))

		// For binary codecs the data field must be a base64-encoded JSON string; a
		// bare JSON number cannot be decoded back into the payload bytes.
		envelope := `{"version":0,"type":"user","data":123}`
		_, err := bs.Deserialize([]byte(envelope))
		require.ErrorIs(t, err, serializer.ErrFailedToDeserialize)
	})
}

func TestNewProtoSerializer(t *testing.T) {
	t.Run("should serialize and deserialize a proto.Message", func(t *testing.T) {
		s := serializer.NewProtoSerializer(serializer.Register(&timestamppb.Timestamp{}))
		now := timestamppb.Now()

		data, err := s.Serialize(now)
		require.NoError(t, err)

		deserialized, err := s.Deserialize(data)
		require.NoError(t, err)
		deserializedNow := deserialized.(*timestamppb.Timestamp)
		assert.True(t, proto.Equal(now, deserializedNow))
	})

	t.Run("should fail to serialize a non-proto.Message", func(t *testing.T) {
		s := serializer.NewProtoSerializer()

		_, err := s.Serialize(fixture.User1)
		require.ErrorIs(t, err, serializer.ErrFailedToSerialize)
	})

	t.Run("should fail to serialize a list of non-proto.Message", func(t *testing.T) {
		s := serializer.NewProtoSerializer()

		_, err := s.SerializeList([]any{fixture.User1})
		require.ErrorIs(t, err, serializer.ErrFailedToSerializeList)
	})

	t.Run("should fail on a corrupt proto payload", func(t *testing.T) {
		s := serializer.NewProtoSerializer(serializer.RegisterAs("ts", &timestamppb.Timestamp{}))

		// A base64 payload whose bytes are not a valid Timestamp fails to decode.
		badPayload, err := json.Marshal([]byte{0x00, 0x01, 0x02})
		require.NoError(t, err)
		envelope := `{"version":0,"type":"ts","data":` + string(badPayload) + `}`
		_, err = s.Deserialize([]byte(envelope))
		require.ErrorIs(t, err, serializer.ErrFailedToDeserialize)
	})
}

func TestNewSerializer(t *testing.T) {
	t.Run("should skip an unnamed registration", func(t *testing.T) {
		// Registering an unnamed (anonymous) type yields an unusable option that
		// must be skipped without disturbing valid registrations alongside it.
		s := serializer.NewJSONSerializer(
			serializer.Register(struct{ X int }{}),
			serializer.Register(fixture.User{}),
		)

		data, err := s.Serialize(fixture.User1)
		require.NoError(t, err)

		deserialized, err := s.Deserialize(data)
		require.NoError(t, err)
		assert.Equal(t, fixture.User1, deserialized)
	})
}

// validateSerializer runs the codec-agnostic round-trip contract shared by every
// serializer flavor, mirroring codec.ValidateCodec.
func validateSerializer(t *testing.T, s *serializer.Serializer) {
	t.Helper()

	const badSerialization = `{"foo":"bar"`
	list := []any{fixture.User1, fixture.User2}
	userList := []fixture.User{fixture.User1, fixture.User2}

	t.Run("should serialize and deserialize a struct", func(t *testing.T) {
		serializedUser1, err := s.Serialize(fixture.User1)
		require.NoError(t, err)

		deserializedUser1, err := s.Deserialize(serializedUser1)
		require.NoError(t, err)
		assert.Equal(t, fixture.User1, deserializedUser1)
	})

	t.Run("should serialize and deserialize a pointer to a struct", func(t *testing.T) {
		serializedUser1, err := s.Serialize(&fixture.User1)
		require.NoError(t, err)

		deserializedUser1, err := s.Deserialize(serializedUser1)
		require.NoError(t, err)
		assert.Equal(t, fixture.User1, deserializedUser1)
	})

	t.Run("should serialize and deserialize a list of structs", func(t *testing.T) {
		serializedList, err := s.SerializeList(list)
		require.NoError(t, err)

		deserializedList, err := s.DeserializeList(serializedList)
		require.NoError(t, err)
		assert.Equal(t, list, deserializedList)
	})

	t.Run("should fail to deserialize an unregistered type", func(t *testing.T) {
		serializedAddress, err := s.Serialize(fixture.Address1)
		require.NoError(t, err)

		_, err = s.Deserialize(serializedAddress)
		assert.Error(t, err)
	})

	t.Run("should serialize and deserialize a list of a given type", func(t *testing.T) {
		serializedList, err := serializer.SerializeList(s, userList)
		require.NoError(t, err)

		deserializedList, err := serializer.DeserializeList[fixture.User](s, serializedList)
		require.NoError(t, err)
		assert.Equal(t, userList, deserializedList)
	})

	t.Run("should fail to deserialize wrong bytes", func(t *testing.T) {
		_, err := s.Deserialize([]byte(badSerialization))
		assert.Error(t, err)
	})

	t.Run("should fail to deserialize wrong bytes for a given type", func(t *testing.T) {
		_, err := serializer.DeserializeList[fixture.User](s, []byte(badSerialization))
		assert.Error(t, err)
	})

	t.Run("should fail to deserialize a list as the wrong type", func(t *testing.T) {
		serializedList, err := serializer.SerializeList(s, userList)
		require.NoError(t, err)

		_, err = serializer.DeserializeList[fixture.Address](s, serializedList)
		assert.Error(t, err)
	})
}

func TestRegister(t *testing.T) {
	t.Run("should return a pointer when a pointer type is registered", func(t *testing.T) {
		serializers := map[string]*serializer.Serializer{
			"json": serializer.NewJSONSerializer(serializer.Register(&fixture.User{})),
			"byte": serializer.NewByteSerializer(serializer.Register(&fixture.User{})),
		}

		for name, s := range serializers {
			t.Run(name, func(t *testing.T) {
				data, err := s.Serialize(&fixture.User1)
				require.NoError(t, err)

				// Registering a pointer type yields a pointer on deserialization, so
				// the recovered value keeps the same kind as the registered type.
				deserialized, err := s.Deserialize(data)
				require.NoError(t, err)
				user, ok := deserialized.(*fixture.User)
				assert.True(t, ok)
				assert.Equal(t, fixture.User1, *user)
			})
		}
	})

	t.Run("should register a typed-nil pointer without panicking", func(t *testing.T) {
		s := serializer.NewJSONSerializer(serializer.Register[*fixture.User](nil))

		data, err := s.Serialize(&fixture.User1)
		require.NoError(t, err)

		deserialized, err := s.Deserialize(data)
		require.NoError(t, err)
		user, ok := deserialized.(*fixture.User)
		assert.True(t, ok)
		assert.Equal(t, fixture.User1, *user)
	})
}

func TestRegisterAs(t *testing.T) {
	t.Run("should register under a stable name", func(t *testing.T) {
		s := serializer.NewJSONSerializer(serializer.RegisterAs("user", fixture.User{}))

		data, err := s.Serialize(fixture.User1)
		require.NoError(t, err)
		// The wire type tag uses the stable name, not the import path.
		assert.Contains(t, string(data), `"type":"user"`)
		assert.NotContains(t, string(data), "internal/fixture")

		deserialized, err := s.Deserialize(data)
		require.NoError(t, err)
		assert.Equal(t, fixture.User1, deserialized)
	})
}

func TestRegisterType(t *testing.T) {
	t.Run("should register from the type parameter", func(t *testing.T) {
		s := serializer.NewByteSerializer(serializer.RegisterType[fixture.User]())

		data, err := s.Serialize(fixture.User1)
		require.NoError(t, err)

		deserialized, err := s.Deserialize(data)
		require.NoError(t, err)
		assert.Equal(t, fixture.User1, deserialized)
	})
}

func TestRegisterTypeAs(t *testing.T) {
	t.Run("should register from the type parameter under a stable name", func(t *testing.T) {
		s := serializer.NewJSONSerializer(serializer.RegisterTypeAs[fixture.User]("user"))

		data, err := s.Serialize(fixture.User1)
		require.NoError(t, err)
		assert.Contains(t, string(data), `"type":"user"`)

		deserialized, err := s.Deserialize(data)
		require.NoError(t, err)
		assert.Equal(t, fixture.User1, deserialized)
	})
}

func TestRegisterVersioned(t *testing.T) {
	t.Run("should stamp version 1 for an unversioned registration", func(t *testing.T) {
		s := serializer.NewJSONSerializer(serializer.Register(fixture.User{}))

		data, err := s.Serialize(fixture.User1)
		require.NoError(t, err)
		assert.Equal(t, 1, envelopeVersion(t, data))
	})

	t.Run("should stamp the registered schema version", func(t *testing.T) {
		s := newVersionedUserSerializer()

		data, err := s.Serialize(fixture.User1)
		require.NoError(t, err)
		assert.Equal(t, 3, envelopeVersion(t, data))
	})

	t.Run("should migrate from version 1", func(t *testing.T) {
		s := newVersionedUserSerializer()

		// A version-1 envelope carrying the userV1 shape must migrate 1->2->3.
		envelope := `{"version":1,"type":"user","data":{"Name":"Jane Doe"}}`
		v, err := s.Deserialize([]byte(envelope))
		require.NoError(t, err)
		assert.Equal(t, fixture.User{Username: "Jane Doe"}, v)
	})

	t.Run("should migrate from version 2", func(t *testing.T) {
		s := newVersionedUserSerializer()

		// A version-2 envelope carrying the userV2 shape must migrate 2->3.
		envelope := `{"version":2,"type":"user","data":{"Username":"janedoe"}}`
		v, err := s.Deserialize([]byte(envelope))
		require.NoError(t, err)
		assert.Equal(t, fixture.User{Username: "janedoe"}, v)
	})

	t.Run("should reject a schema version that is too new", func(t *testing.T) {
		s := serializer.NewJSONSerializer(serializer.RegisterAs("user", fixture.User{}))

		envelope := `{"version":999,"type":"user","data":{}}`
		_, err := s.Deserialize([]byte(envelope))
		require.ErrorIs(t, err, serializer.ErrSchemaTooNew)
		require.ErrorIs(t, err, serializer.ErrFailedToDeserialize)
	})

	t.Run("should reject a version below the migration suffix", func(t *testing.T) {
		// The oldest migration is From(2), so version 1 is below the supported suffix.
		s := serializer.NewJSONSerializer(
			serializer.RegisterVersioned("user", 3, fixture.User{},
				serializer.From(2, func(o userV2) (fixture.User, error) {
					return fixture.User{Username: o.Username}, nil
				}),
			),
		)

		envelope := `{"version":1,"type":"user","data":{"Name":"Jane Doe"}}`
		_, err := s.Deserialize([]byte(envelope))
		require.ErrorIs(t, err, serializer.ErrNoMigrationPath)
		require.ErrorIs(t, err, serializer.ErrFailedToDeserialize)
	})

	t.Run("should accept the legacy version zero", func(t *testing.T) {
		s := serializer.NewJSONSerializer(serializer.Register(fixture.User{}))

		data, err := s.Serialize(fixture.User1)
		require.NoError(t, err)

		// Data written before envelope versioning carries "version":0; its byte
		// layout is identical, so it must still deserialize.
		legacy := bytes.Replace(data, []byte(`"version":1`), []byte(`"version":0`), 1)
		assert.NotEqual(t, data, legacy)

		deserialized, err := s.Deserialize(legacy)
		require.NoError(t, err)
		assert.Equal(t, fixture.User1, deserialized)
	})

	t.Run("should panic when the version is below one", func(t *testing.T) {
		assert.Panics(t, func() {
			serializer.RegisterVersioned("user", 0, fixture.User{})
		})
	})

	t.Run("should panic on a hole in the migration suffix", func(t *testing.T) {
		// From(1) and From(3) at version 4 leave a hole at From(2).
		assert.Panics(t, func() {
			serializer.RegisterVersioned("user", 4, fixture.User{},
				serializer.From(1, func(_ userV1) (userV2, error) { return userV2{}, nil }),
				serializer.From(3, func(_ userV2) (fixture.User, error) { return fixture.User{}, nil }),
			)
		})
	})

	t.Run("should panic on a broken type link", func(t *testing.T) {
		// The last step outputs userV2, but the current type is fixture.User.
		assert.Panics(t, func() {
			serializer.RegisterVersioned("user", 2, fixture.User{},
				serializer.From(1, func(_ userV1) (userV2, error) { return userV2{}, nil }),
			)
		})
	})
}

func TestSerialize(t *testing.T) {
	t.Run("should fail to serialize a nil value", func(t *testing.T) {
		s := serializer.NewJSONSerializer()

		// Serializing nil must return an error, not panic.
		_, err := s.Serialize(nil)
		require.ErrorIs(t, err, serializer.ErrFailedToSerialize)
	})

	t.Run("should fail to serialize an unnamed type", func(t *testing.T) {
		s := serializer.NewJSONSerializer()

		// Anonymous structs have no stable wire name and are rejected.
		_, err := s.Serialize(struct{ X int }{X: 1})
		require.ErrorIs(t, err, serializer.ErrFailedToSerialize)
	})
}

func TestSerializeList(t *testing.T) {
	t.Run("should fail to serialize a list containing a nil element", func(t *testing.T) {
		s := serializer.NewJSONSerializer()

		_, err := s.SerializeList([]any{nil})
		require.ErrorIs(t, err, serializer.ErrFailedToSerializeList)
	})
}

func TestDeserialize(t *testing.T) {
	t.Run("should fail without panicking for a defined pointer type", func(t *testing.T) {
		type userPtr *fixture.User
		s := serializer.NewJSONSerializer(serializer.Register(userPtr(&fixture.User1)))

		data, err := s.Serialize(userPtr(&fixture.User1))
		require.NoError(t, err)

		// A defined pointer type cannot be reconstructed by reflect.New; the result
		// must be a clean error, never a panic.
		_, err = s.Deserialize(data)
		require.ErrorIs(t, err, serializer.ErrFailedToDeserialize)
	})
}

func TestDeserializeList(t *testing.T) {
	t.Run("should fail to deserialize a list with an unregistered element", func(t *testing.T) {
		s := serializer.NewJSONSerializer(serializer.Register(fixture.User{}))

		// Build a list whose only element is a serialized but unregistered Address.
		serializedAddress, err := s.Serialize(fixture.Address1)
		require.NoError(t, err)
		list, err := json.Marshal([][]byte{serializedAddress})
		require.NoError(t, err)

		_, err = s.DeserializeList(list)
		require.ErrorIs(t, err, serializer.ErrFailedToDeserializeList)
	})
}

// envelopeVersion extracts the version field from a serialized envelope.
func envelopeVersion(t *testing.T, data []byte) int {
	t.Helper()

	var envelope struct {
		Version int `json:"version"`
	}
	require.NoError(t, json.Unmarshal(data, &envelope))
	return envelope.Version
}

// userV1 and userV2 are frozen historical revisions of fixture.User, used to
// exercise the migration chain. They mirror how a caller keeps old shapes around
// once a newer version ships.
type userV1 struct{ Name string }

type userV2 struct{ Username string }

// newVersionedUserSerializer registers fixture.User at schema version 3 behind a
// two-step migration chain: userV1 (v1) -> userV2 (v2) -> fixture.User (v3).
func newVersionedUserSerializer() *serializer.Serializer {
	return serializer.NewJSONSerializer(
		serializer.RegisterVersioned("user", 3, fixture.User{},
			serializer.From(1, func(o userV1) (userV2, error) {
				return userV2{Username: o.Name}, nil
			}),
			serializer.From(2, func(o userV2) (fixture.User, error) {
				return fixture.User{Username: o.Username}, nil
			}),
		),
	)
}

// benchListSize is the number of elements used by the list benchmarks.
const benchListSize = 100

type benchCase struct {
	name  string
	s     *serializer.Serializer
	value any
}

// benchCases returns one case per codec, each with a matching registered value.
func benchCases() []benchCase {
	return []benchCase{
		{"json", serializer.NewJSONSerializer(serializer.Register(fixture.User{})), fixture.User1},
		{"byte", serializer.NewByteSerializer(serializer.Register(fixture.User{})), fixture.User1},
		{"proto", serializer.NewProtoSerializer(serializer.Register(&timestamppb.Timestamp{})), timestamppb.Now()},
	}
}

func (tc benchCase) list() []any {
	list := make([]any, benchListSize)
	for i := range list {
		list[i] = tc.value
	}
	return list
}

func BenchmarkSerialize(b *testing.B) {
	for _, tc := range benchCases() {
		b.Run(tc.name, func(b *testing.B) {
			_, err := tc.s.Serialize(tc.value)
			require.NoError(b, err)

			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				_, _ = tc.s.Serialize(tc.value)
			}
		})
	}
}

func BenchmarkDeserialize(b *testing.B) {
	for _, tc := range benchCases() {
		data, err := tc.s.Serialize(tc.value)
		require.NoError(b, err)

		b.Run(tc.name, func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				_, _ = tc.s.Deserialize(data)
			}
		})
	}
}

func BenchmarkSerializeList(b *testing.B) {
	for _, tc := range benchCases() {
		list := tc.list()
		_, err := tc.s.SerializeList(list)
		require.NoError(b, err)

		b.Run(tc.name, func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				_, _ = tc.s.SerializeList(list)
			}
		})
	}
}

func BenchmarkDeserializeList(b *testing.B) {
	for _, tc := range benchCases() {
		data, err := tc.s.SerializeList(tc.list())
		require.NoError(b, err)

		b.Run(tc.name, func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				_, _ = tc.s.DeserializeList(data)
			}
		})
	}
}
