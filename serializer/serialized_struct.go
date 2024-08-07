package serializer

import "encoding/json"

// defaultSchemaVersion is the object schema version implied by an unversioned
// registration, and the version stamped for any type serialized without a
// registration.
const defaultSchemaVersion = 1

// serializedStruct represent an object that has been serialized.
//
// Data holds the codec-encoded payload. For codecs whose output is itself valid
// JSON (see codec.Codec.EncodesJSON) the payload is embedded verbatim, keeping
// the envelope human-readable and avoiding a redundant base64 pass. For binary
// codecs the payload is stored as a base64-encoded JSON string.
type serializedStruct struct {
	// Version is the object schema version: which revision of the registered type
	// produced Data. Deserialize uses it to pick a migration chain that lifts an
	// older revision up to the current Go type. A value of 0 (an absent field) is
	// normalized to defaultSchemaVersion on read.
	Version int             `json:"version"`
	Type    string          `json:"type"`
	Data    json.RawMessage `json:"data"`
}
