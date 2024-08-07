package serializer

import "errors"

var (
	// ErrFailedToSerialize is returned when serialization of a struct fails.
	ErrFailedToSerialize = errors.New("failed to serialize the struct")

	// ErrFailedToSerializeList is returned when serialization of a list fails.
	ErrFailedToSerializeList = errors.New("failed to serialize the list")

	// ErrFailedToDeserialize is returned when deserialization of a struct fails.
	ErrFailedToDeserialize = errors.New("failed to deserialize the struct")

	// ErrFailedToDeserializeList is returned when deserialization of a list fails.
	ErrFailedToDeserializeList = errors.New("failed to deserialize the list")

	// ErrSchemaTooNew is returned when an envelope's schema version is newer than
	// the current registered version, so it cannot be migrated (forward
	// compatibility is not supported). It is always wrapped together with
	// ErrFailedToDeserialize, so callers can match either sentinel.
	ErrSchemaTooNew = errors.New("schema version is newer than the registered type")

	// ErrNoMigrationPath is returned when an envelope's schema version is older
	// than the current version but no registered migration chain reaches it. It is
	// always wrapped together with ErrFailedToDeserialize, so callers can match
	// either sentinel.
	ErrNoMigrationPath = errors.New("no migration path for the schema version")
)
