// Package serializer serializes and deserializes Go structs without requiring
// the caller to supply a destination type at decode time.
//
// A Serializer pairs a codec (JSON, gob, or protobuf) with a set of registered
// types. Serialize wraps the codec-encoded payload in a small JSON envelope that
// records the value's type, and Deserialize uses that type tag to reconstruct
// the original value — provided the type was registered when the Serializer was
// created:
//
//	s := serializer.NewJSONSerializer(serializer.Register(User{}))
//	data, err := s.Serialize(user)
//	value, err := s.Deserialize(data) // value holds a User
//
// Types are identified on the wire by their full import path and name. Use
// RegisterAs to pin a stable name that survives package moves or renames, and
// RegisterType when no instance is handy:
//
//	serializer.RegisterType[User]()
//	serializer.RegisterAs("user", User{})
//
// Only named, exported struct types are supported. The deserialized value has
// the same kind (value or pointer) as the type that was registered.
package serializer
