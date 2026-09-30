package serializer

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"sync"

	"github.com/grandper/go-serializer/internal/codec"
)

// Serializer serializes go structs.
type Serializer struct {
	registrations map[string]*registration
	typeNames     map[reflect.Type]string
	codec         codec.Codec
}

// registration is the per-type decode record: how to decode the current schema
// version, and the migrations that lift older versions up to it.
type registration struct {
	current int               // current schema version
	decode  deserializer      // decodes the current type (hot path)
	steps   map[int]Migration // fromVersion -> Migration
}

// NewJSONSerializer creates a new JSON serializer.
func NewJSONSerializer(options ...Option) *Serializer {
	return NewSerializer(codec.NewJSONCodec(), options...)
}

// NewByteSerializer creates a new byte serializer.
func NewByteSerializer(options ...Option) *Serializer {
	return NewSerializer(codec.NewByteCodec(), options...)
}

// NewProtoSerializer creates a new proto serializer.
func NewProtoSerializer(options ...Option) *Serializer {
	return NewSerializer(codec.NewProtoCodec(), options...)
}

// NewSerializer creates a new struct serializer.
func NewSerializer(c codec.Codec, options ...Option) *Serializer {
	registrations := make(map[string]*registration, len(options))
	typeNames := make(map[reflect.Type]string, len(options))
	for _, option := range options {
		// An empty name means the type could not be identified (an unnamed or
		// nil type); such an option is not registrable and is skipped, so
		// serializing that type later fails cleanly rather than matching "".
		if option.typeName == "" {
			continue
		}
		steps := make(map[int]Migration, len(option.migrations))
		for _, m := range option.migrations {
			steps[m.fromVersion] = m
		}
		registrations[option.typeName] = &registration{
			current: option.version,
			decode:  option.deserializer,
			steps:   steps,
		}
		if option.goType != nil {
			typeNames[option.goType] = option.typeName
		}
	}
	return &Serializer{
		registrations: registrations,
		typeNames:     typeNames,
		codec:         c,
	}
}

// Serialize serializes a struct.
func (s *Serializer) Serialize(aStruct any) ([]byte, error) {
	// Resolve the wire name first so an unnamed or nil value fails cleanly
	// (rather than panicking) before any encoding work is done.
	tName, err := s.typeNameFor(aStruct)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrFailedToSerialize, err)
	}
	data, err := s.codec.Encode(aStruct)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrFailedToSerialize, err)
	}
	payload, err := encodePayload(s.codec, data)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrFailedToSerialize, err)
	}
	version := defaultSchemaVersion
	if reg, ok := s.registrations[tName]; ok {
		version = reg.current
	}
	st := serializedStruct{
		Version: version,
		Type:    tName,
		Data:    payload,
	}
	serializedData, err := json.Marshal(st)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrFailedToSerialize, err)
	}
	return serializedData, nil
}

// SerializeList serializes a list of structs.
func (s *Serializer) SerializeList(structs []any) ([]byte, error) {
	return SerializeList(s, structs)
}

// SerializeList serializes a list of structs.
func SerializeList[T any](s *Serializer, structs []T) ([]byte, error) {
	serializedStructs := make([][]byte, 0, len(structs))
	for _, aStruct := range structs {
		serializedData, err := s.Serialize(aStruct)
		if err != nil {
			return nil, fmt.Errorf("%w: %w", ErrFailedToSerializeList, err)
		}
		serializedStructs = append(serializedStructs, serializedData)
	}
	result, err := json.Marshal(serializedStructs)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrFailedToSerializeList, err)
	}
	return result, nil
}

// Deserialize deserializes a struct.
func (s *Serializer) Deserialize(data []byte) (any, error) {
	var st serializedStruct
	if err := json.Unmarshal(data, &st); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrFailedToDeserialize, err)
	}
	v := st.Version
	if v <= 0 { // absent field / legacy zero -> baseline schema 1
		v = defaultSchemaVersion
	}
	reg, found := s.registrations[st.Type]
	if !found {
		return nil, fmt.Errorf("%w: %w: '%s'", ErrFailedToDeserialize, ErrTypeNotRegistered, st.Type)
	}
	payload, err := decodePayload(s.codec, st.Data)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrFailedToDeserialize, err)
	}

	c := reg.current
	switch {
	case v > c:
		return nil, fmt.Errorf("%w: %w: got schema version %d, current is %d",
			ErrFailedToDeserialize, ErrSchemaTooNew, v, c)

	case v == c: // hot path: no migration, no extra allocation
		return reg.decode(s.codec, payload)

	default: // v < c: decode as the old type, then run the chain up to c
		entry, ok := reg.steps[v]
		if !ok { // v is below the supported suffix
			return nil, fmt.Errorf("%w: %w: no migration from schema version %d",
				ErrFailedToDeserialize, ErrNoMigrationPath, v)
		}
		var value any
		value, err = entry.decode(s.codec, payload)
		if err != nil {
			return nil, err // already wrapped by the decode closure
		}
		for k := v; k < c; k++ { // steps[v..c-1] all present by the §2.4 suffix contract
			value, err = reg.steps[k].apply(value)
			if err != nil {
				return nil, fmt.Errorf("%w: migrating schema version %d->%d: %w",
					ErrFailedToDeserialize, k, k+1, err)
			}
		}
		return value, nil
	}
}

// DeserializeList serializes a list of structs.
func (s *Serializer) DeserializeList(data []byte) ([]any, error) {
	return DeserializeList[any](s, data)
}

// DeserializeList serializes a list of structs.
func DeserializeList[T any](s *Serializer, data []byte) ([]T, error) {
	var serializedStructs [][]byte
	if err := json.Unmarshal(data, &serializedStructs); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrFailedToDeserializeList, err)
	}
	structs := make([]T, 0, len(serializedStructs))
	for _, serializedData := range serializedStructs {
		aStructAny, err := s.Deserialize(serializedData)
		if err != nil {
			return nil, fmt.Errorf("%w: %w", ErrFailedToDeserializeList, err)
		}
		aStruct, ok := aStructAny.(T)
		if !ok {
			return nil, fmt.Errorf("%w: the list contains struct with bad type", ErrFailedToDeserializeList)
		}
		structs = append(structs, aStruct)
	}
	return structs, nil
}

type deserializer func(c codec.Codec, data []byte) (any, error)

// Option is an option for the serializer.
type Option struct {
	typeName     string
	goType       reflect.Type
	deserializer deserializer // decodes the CURRENT type
	version      int          // schema version; defaultSchemaVersion for unversioned registrations
	migrations   []Migration  // nil for unversioned registrations
}

// Register registers a type to make sure it can be unserialized. The type is
// identified on the wire by its full import path and name. Only named, exported
// types are registrable; an unnamed type yields an empty name and is dropped by
// NewSerializer, so serializing it later fails cleanly.
func Register[T any](value T) Option {
	// The name is derived from the type parameter, so registration works even
	// when handed a typed-nil value. Any error means an unnamed type, which is
	// unsupported; the empty name causes NewSerializer to skip the option.
	name, _ := detectStructTypeForT[T]()
	return registerAs(name, value)
}

// RegisterType registers a type from its type parameter alone, without needing
// an instance: serializer.RegisterType[User]().
func RegisterType[T any]() Option {
	var zero T
	return Register(zero)
}

// RegisterAs registers a type under a caller-supplied stable name instead of its
// import path. This keeps serialized data readable across package moves/renames
// (the name must stay stable and unique across registered types).
func RegisterAs[T any](name string, value T) Option {
	return registerAs(name, value)
}

// RegisterTypeAs registers a type from its type parameter under a caller-supplied
// stable name, without needing an instance.
func RegisterTypeAs[T any](name string) Option {
	var zero T
	return registerAs(name, zero)
}

func registerAs[T any](name string, value T) Option {
	return Option{
		typeName:     name,
		goType:       normalizedType(value),
		deserializer: newDeserializer[T](value),
		version:      defaultSchemaVersion,
		migrations:   nil,
	}
}

// Migration lifts a value written by one schema version to the next. Build one
// with From and pass it to RegisterVersioned.
type Migration struct {
	fromVersion int                    // migrates fromVersion -> fromVersion+1
	inType      reflect.Type           // the old (decoded) type, = From's Old
	outType     reflect.Type           // the produced type,       = From's New
	decode      deserializer           // bytes -> inType value (used only as a chain entry point)
	apply       func(any) (any, error) // inType value -> outType value
}

// From builds a stepwise migration from schema version fromVersion to
// fromVersion+1. fn receives the fully decoded old value and returns the next
// version's value. Old and New are inferred from fn, so callers never spell the
// historical type twice.
func From[Old, New any](fromVersion int, fn func(Old) (New, error)) Migration {
	var zero Old
	return Migration{
		fromVersion: fromVersion,
		inType:      reflect.TypeOf((*Old)(nil)).Elem(),
		outType:     reflect.TypeOf((*New)(nil)).Elem(),
		decode:      newDeserializer[Old](zero), // reuse the existing decode-closure builder
		apply: func(v any) (any, error) {
			old, ok := v.(Old)
			if !ok {
				return nil, fmt.Errorf("%w: migration for version %d received %T, want %s",
					ErrFailedToDeserialize, fromVersion, v, reflect.TypeOf((*Old)(nil)).Elem())
			}
			return fn(old)
		},
	}
}

// RegisterVersioned registers value as the CURRENT type under the stable wire
// name, at schema version, together with the migrations that lift
// older versions up to it. version is the number Serialize stamps and the
// revision Deserialize returns.
//
// It PANICS on a misconfigured chain (see validateMigrations) — a deterministic
// programmer error surfaced at construction, following the regexp.MustCompile
// idiom.
func RegisterVersioned[T any](name string, version int, value T, migrations ...Migration) Option {
	currentType := reflect.TypeOf((*T)(nil)).Elem()
	validateMigrations(name, version, currentType, migrations)
	return Option{
		typeName:     name,
		goType:       normalizedType(value),
		deserializer: newDeserializer[T](value),
		version:      version,
		migrations:   migrations,
	}
}

// validateMigrations enforces the migration-chain contract and panics on any
// violation. Let C = version and P = { m.fromVersion }. The supplied migrations
// must form a contiguous suffix { m, m+1, …, C-1 } whose types link end to end
// up to the current type T.
func validateMigrations(name string, version int, currentType reflect.Type, migrations []Migration) {
	if version < 1 {
		panic(fmt.Sprintf("serializer: RegisterVersioned(%q): version must be >= 1, got %d", name, version))
	}

	if len(migrations) == 0 {
		return
	}

	// Index by fromVersion, rejecting duplicates and out-of-range versions.
	byFrom := make(map[int]Migration, len(migrations))
	for _, m := range migrations {
		if _, dup := byFrom[m.fromVersion]; dup {
			panic(fmt.Sprintf("serializer: RegisterVersioned(%q): duplicate migration From(%d)", name, m.fromVersion))
		}
		if m.fromVersion < 1 || m.fromVersion > version-1 {
			panic(fmt.Sprintf("serializer: RegisterVersioned(%q): migration From(%d) is out of range [1, %d]",
				name, m.fromVersion, version-1))
		}
		byFrom[m.fromVersion] = m
	}

	// Contiguous suffix: P must equal { oldest, …, C-1 } with no holes.
	oldest := version - 1
	for byFrom[oldest-1].apply != nil { // walk down while the next-older step exists
		oldest--
	}
	if len(byFrom) != version-oldest {
		panic(
			fmt.Sprintf("serializer: RegisterVersioned(%q): migrations must form a contiguous run ending at version %d",
				name, version-1),
		)
	}

	// Type links: each step's output feeds the next step's input, and the last
	// step's output is the current type T.
	for k := oldest; k <= version-2; k++ {
		if byFrom[k].outType != byFrom[k+1].inType {
			panic(
				fmt.Sprintf("serializer: RegisterVersioned(%q): migration From(%d) outputs %s but From(%d) expects %s",
					name, k, byFrom[k].outType, k+1, byFrom[k+1].inType),
			)
		}
	}
	if last := byFrom[version-1]; last.outType != currentType {
		panic(fmt.Sprintf("serializer: RegisterVersioned(%q): migration From(%d) outputs %s but the current type is %s",
			name, version-1, last.outType, currentType))
	}
}

// newDeserializer builds the decode closure for a registered type. When T is a
// pointer type the closure allocates the pointee before decoding, so the codec
// receives a usable *E rather than a nil **E — proto.Unmarshal rejects the
// latter, whereas the JSON and gob codecs allocate it implicitly. Value types
// decode into a local. The pointer element type is resolved once, at
// registration time (a cold path), so Deserialize does no reflective type
// walking per call.
func newDeserializer[T any](value T) deserializer {
	if t := reflect.TypeOf(value); t != nil && t.Kind() == reflect.Ptr {
		elemType := t.Elem()
		return func(c codec.Codec, data []byte) (any, error) {
			// reflect.New yields the plain *elemType; the comma-ok assertion
			// guards the exotic case of a defined pointer type (type P *E), where
			// *E is not assignable to P, turning a would-be panic into an error.
			registeredType, ok := reflect.New(elemType).Interface().(T)
			if !ok {
				return nil, fmt.Errorf("%w: cannot construct a value of the registered type", ErrFailedToDeserialize)
			}
			if err := c.Decode(data, registeredType); err != nil {
				return nil, fmt.Errorf("%w: %w", ErrFailedToDeserialize, err)
			}
			return registeredType, nil
		}
	}
	return func(c codec.Codec, data []byte) (any, error) {
		var registeredType T
		if err := c.Decode(data, &registeredType); err != nil {
			return nil, fmt.Errorf("%w: %w", ErrFailedToDeserialize, err)
		}
		return registeredType, nil
	}
}

// typeNameFor returns the wire name for a value: the name supplied at
// registration time when known, otherwise the import-path-derived name. It
// errors for nil or unnamed values, which cannot be identified on the wire.
func (s *Serializer) typeNameFor(aStruct any) (string, error) {
	if t := normalizedType(aStruct); t != nil {
		if name, ok := s.typeNames[t]; ok {
			return name, nil
		}
	}
	return detectStructType(aStruct)
}

// encodePayload places a codec-encoded payload into the JSON envelope. JSON
// codecs embed the payload verbatim; binary codecs store it as a base64 string.
func encodePayload(c codec.Codec, data []byte) (json.RawMessage, error) {
	if c.EncodesJSON() {
		return data, nil
	}
	encoded, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	return encoded, nil
}

// decodePayload reverses encodePayload, recovering the codec payload bytes.
func decodePayload(c codec.Codec, data json.RawMessage) ([]byte, error) {
	if c.EncodesJSON() {
		return data, nil
	}
	var payload []byte
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, err
	}
	return payload, nil
}

// normalizedType returns the type of aStruct with a single level of pointer
// indirection removed, or nil when the type cannot be determined.
func normalizedType(aStruct any) reflect.Type {
	t := reflect.TypeOf(aStruct)
	if t != nil && t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return t
}

// typeNameCache memoizes the wire name for a (pointer-stripped) reflect.Type.
// The mapping is immutable per type, so a sync.Map keeps the Serialize hot path
// allocation-free for already-seen types while remaining safe for concurrent
// use — the key set is small, bounded, and read-dominated.
var typeNameCache sync.Map //nolint:gochecknoglobals // process-wide memo of an immutable reflect.Type -> name mapping

// typeName returns the wire name for a pointer-stripped reflect.Type, caching
// the result. It errors for nil types and for unnamed (anonymous, slice, map,
// ...) types, which cannot be identified unambiguously on the wire. All
// reflection and string formatting happens once per type; repeat calls are a
// single allocation-free map load.
func typeName(t reflect.Type) (string, error) {
	if t == nil {
		return "", errors.New("cannot determine the type of a nil value")
	}
	if cached, ok := typeNameCache.Load(t); ok {
		if name, isString := cached.(string); isString {
			return name, nil
		}
	}
	if t.Name() == "" {
		return "", fmt.Errorf("cannot serialize the unnamed type %q", t.String())
	}
	name := t.PkgPath() + "." + t.Name()
	typeNameCache.Store(t, name)
	return name, nil
}

// detectStructType resolves the wire name for a value's concrete type.
func detectStructType(aStruct any) (string, error) {
	return typeName(normalizedType(aStruct))
}

// detectStructTypeForT resolves the wire name from the type parameter alone, so
// registration works even when handed a typed-nil value. A pointer type is
// reduced to its element so a value and a pointer to it share one name.
func detectStructTypeForT[T any]() (string, error) {
	t := reflect.TypeOf((*T)(nil)).Elem()
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return typeName(t)
}
