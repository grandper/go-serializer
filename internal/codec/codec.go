package codec

// Codec is a generic interface for encoding and decoding structs.
type Codec interface {
	// Encode encodes a struct into bytes.
	Encode(v any) ([]byte, error)

	// Decode decodes bytes into a struct.
	Decode(data []byte, v any) error

	// EncodesJSON reports whether the bytes produced by Encode are themselves
	// valid JSON. When true, the serializer can embed the payload directly into
	// its JSON envelope instead of base64-encoding it, avoiding a redundant
	// encoding pass and keeping the output human-readable.
	EncodesJSON() bool
}
