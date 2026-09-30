# go-serializer

[![Test](https://github.com/grandper/go-serializer/actions/workflows/go-test.yml/badge.svg)](https://github.com/grandper/go-serializer/actions/workflows/go-test.yml)
[![Lint](https://github.com/grandper/go-serializer/actions/workflows/go-lint.yml/badge.svg)](https://github.com/grandper/go-serializer/actions/workflows/go-lint.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/grandper/go-serializer/serializer.svg)](https://pkg.go.dev/github.com/grandper/go-serializer/serializer)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

> Type-aware serialization for Go — round-trip any registered struct through JSON, gob, or Protocol Buffers **without naming the destination type at decode time**.

With `encoding/json` and most other Go serializers, decoding forces you to know
the concrete type up front:

```go
var user User
_ = json.Unmarshal(data, &user) // you must already know it's a User
```

`go-serializer` removes that requirement. `Serialize` wraps the codec-encoded
payload in a small JSON envelope that records the value's type; `Deserialize`
reads that tag and reconstructs the original value from `[]byte` alone — as long
as the type was registered when the serializer was created.

## Features

- **No destination type at decode time** — `Deserialize([]byte) (any, error)`.
- **Pluggable codecs** — JSON (readable), gob (compact binary), Protocol Buffers,
  or your own `codec.Codec`.
- **Value / pointer fidelity** — register a value, get a value; register a
  pointer, get a pointer.
- **Stable wire names** — pin a name with `RegisterAs` so data survives package
  moves and renames.
- **Typed sentinel errors** — branch on failures with `errors.Is`.
- **Schema versioning with migrations** — evolve a type's shape and migrate old
  data forward on read with `RegisterVersioned` / `From`.
- **Built for the hot path** — type-name resolution is cached and
  allocation-free after first use; safe for concurrent use.

## Contents

- [Installation](#installation)
- [Quick start](#quick-start)
- [Choosing a codec](#choosing-a-codec)
- [Serializing](#serializing)
- [Deserializing](#deserializing)
- [Registering types](#registering-types)
- [Working with Protocol Buffers](#working-with-protocol-buffers)
- [Error handling](#error-handling)
- [Envelope format and schema versioning](#envelope-format-and-schema-versioning)
- [Notes and limitations](#notes-and-limitations)
- [License](#license)

## Installation

```bash
go get github.com/grandper/go-serializer
```

```go
import "github.com/grandper/go-serializer/serializer"
```

Requires Go 1.23 or newer.

## Quick start

```go
package main

import (
	"fmt"

	"github.com/grandper/go-serializer/serializer"
)

type User struct {
	Username string
	Age      int
}

func main() {
	// Register every type you may later want to deserialize.
	s := serializer.NewJSONSerializer(serializer.Register(User{}))

	data, err := s.Serialize(User{Username: "johndoe", Age: 24})
	if err != nil {
		panic(err)
	}

	value, err := s.Deserialize(data)
	if err != nil {
		panic(err)
	}

	user := value.(User) // recover the concrete type
	fmt.Println(user.Username) // johndoe
}
```

## Choosing a codec

The envelope is always JSON, but the payload inside it is produced by the codec
you pick when constructing the serializer:

```go
s := serializer.NewJSONSerializer()  // human-readable JSON payload
s := serializer.NewByteSerializer()  // compact gob (binary) payload
s := serializer.NewProtoSerializer() // Protocol Buffers payload
```

Every constructor accepts the same registration options. To plug in a custom
codec, use the base constructor with any value whose method set satisfies the
codec interface:

```go
// Any type with these three methods can be passed to NewSerializer.
type Codec interface {
	Encode(v any) ([]byte, error)   // encode a value to bytes
	Decode(data []byte, v any) error // decode bytes into the pointer v
	EncodesJSON() bool               // true if Encode already emits valid JSON
}

s := serializer.NewSerializer(myCodec, serializer.Register(User{}))
```

`EncodesJSON` controls how the payload sits in the envelope: return `true` and
the codec's output is embedded verbatim (keeping the envelope readable); return
`false` and it is base64-encoded as a JSON string. The interface itself lives in
an internal package, so a custom codec satisfies it **structurally** — just
implement the three methods; there is no type to import.

| Constructor            | Payload codec    | Requirement                        |
| ---------------------- | ---------------- | ---------------------------------- |
| `NewJSONSerializer`    | `encoding/json`  | JSON-marshalable exported fields   |
| `NewByteSerializer`    | `encoding/gob`   | gob-encodable exported fields      |
| `NewProtoSerializer`   | Protocol Buffers | value implements `proto.Message`   |
| `NewSerializer(codec)` | your codec       | `Encode` / `Decode` / `EncodesJSON` methods |

## Serializing

A single struct (a value or a pointer both work):

```go
data, err := s.Serialize(user)
```

A list — if it is already `[]any`:

```go
data, err := s.SerializeList([]any{user1, user2})
```

Otherwise use the generic free function, which keeps the element type:

```go
data, err := serializer.SerializeList(s, []User{user1, user2})
```

## Deserializing

Unlike most serialization libraries, you do **not** pass a destination to
`Deserialize`. The trade-off is that every type you may decode must be registered
when the serializer is created:

```go
s := serializer.NewJSONSerializer(serializer.Register(User{}))

value, err := s.Deserialize(data)
user := value.(User)
```

The deserialized value has the **same kind as the type you registered**: register
a value (`Register(User{})`) and you get a `User` back; register a pointer
(`Register(&User{})`) and you get a `*User`. Recover the concrete type with a
type assertion.

For lists, `DeserializeList` returns a `[]any`:

```go
values, err := s.DeserializeList(data)
```

...or use the generic free function to recover a typed slice (it errors if any
element decodes to a type other than `T`):

```go
users, err := serializer.DeserializeList[User](s, data)
```

## Registering types

There are four ways to register a type — pick the one that fits your call site:

```go
serializer.Register(User{})              // by value,          named by import path
serializer.RegisterType[User]()          // by type parameter, named by import path
serializer.RegisterAs("user", User{})    // by value,          named "user"
serializer.RegisterTypeAs[User]("user")  // by type parameter, named "user"
```

By default a type is identified on the wire by its **full import path plus name**
(for example `github.com/you/app/model.User`), so moving or renaming the type's
package breaks deserialization of previously produced data.

Use `RegisterAs` / `RegisterTypeAs` to pin a **stable name** that survives such
changes. The name must stay stable and unique across all registered types:

```go
s := serializer.NewJSONSerializer(serializer.RegisterAs("user", User{}))
// -> envelope carries "type":"user" instead of the import path
```

## Working with Protocol Buffers

Proto messages satisfy `proto.Message` on their **pointer** type, so register the
pointer — deserialization then hands you back a pointer:

```go
s := serializer.NewProtoSerializer(serializer.Register(&timestamppb.Timestamp{}))

data, err := s.Serialize(timestamppb.Now())
if err != nil {
	// ...
}

value, err := s.Deserialize(data)
if err != nil {
	// ...
}
ts := value.(*timestamppb.Timestamp)
```

## Error handling

Every failure wraps one of the exported sentinel errors, so you can branch on the
operation that failed with `errors.Is`:

```go
value, err := s.Deserialize(data)
if errors.Is(err, serializer.ErrFailedToDeserialize) {
	// handle a failed deserialization
}
```

| Sentinel                                | Returned when                                   |
| --------------------------------------- | ----------------------------------------------- |
| `serializer.ErrFailedToSerialize`       | serializing a struct fails                      |
| `serializer.ErrFailedToSerializeList`   | serializing a list fails                        |
| `serializer.ErrFailedToDeserialize`     | deserializing a struct fails                    |
| `serializer.ErrFailedToDeserializeList` | deserializing a list fails                      |
| `serializer.ErrTypeNotRegistered`       | the type named by the envelope is not registered |
| `serializer.ErrSchemaTooNew`            | the schema version is newer than the registered type |
| `serializer.ErrNoMigrationPath`         | the schema version is older than the current type but no migration reaches it |

The underlying cause (a codec error, a type mismatch…) is wrapped and available
through `errors.Unwrap` / `%w`. `ErrTypeNotRegistered`, `ErrSchemaTooNew` and
`ErrNoMigrationPath` are each wrapped **together with** `ErrFailedToDeserialize`,
so either sentinel matches:

```go
if errors.Is(err, serializer.ErrSchemaTooNew) {
	// the data was written by a newer revision of the type than this build knows
}
```

Serializing a `nil` value or an unnamed type (an anonymous struct, a slice, …)
returns `ErrFailedToSerialize` rather than panicking — only named, exported types
can be identified on the wire.

## Envelope format and schema versioning

Every serialized value is a small JSON envelope:

```json
{ "version": 1, "type": "github.com/you/app/model.User", "data": { ... } }
```

- `version` — the **object schema version**: which revision of the registered
  type produced `data`. An unversioned registration implies schema `1`; data
  written before versioning existed carries `"version":0`, which is read as `1`.
- `type` — the registered wire name (import path + name, or the custom name from
  `RegisterAs`).
- `data` — the codec-encoded payload, embedded verbatim for JSON codecs or stored
  as a base64 string for binary codecs.

### Evolving a type with migrations

When a type's shape changes, keep the old shapes around and register the current
type with `RegisterVersioned`, passing one `From` migration per version step.
Migrations are **stepwise** — each lifts version `n` to `n+1` — and the chain is
composed at decode time, so data written by any past revision is migrated forward
to the current Go type on read. The old type for each step is inferred from the
migration function's parameter, so you never spell it twice.

```go
// Frozen historical revisions, kept once the next version ships.
type UserV1 struct{ Name string }
type UserV2 struct{ FirstName, LastName string }

// Current type, schema 3.
type User struct{ FirstName, LastName, Country string }

s := serializer.NewJSONSerializer(
	serializer.RegisterVersioned("user", 3, User{},
		serializer.From(1, func(o UserV1) (UserV2, error) {
			first, last, _ := strings.Cut(o.Name, " ")
			return UserV2{FirstName: first, LastName: last}, nil
		}),
		serializer.From(2, func(o UserV2) (User, error) {
			return User{FirstName: o.FirstName, LastName: o.LastName, Country: "US"}, nil
		}),
	),
)

// {"version":1,"type":"user","data":{"Name":"Jane Doe"}} → User, migrated 1→2→3.
v, _ := s.Deserialize(oldBytes)
```

`Serialize` stamps the registered version (`3` above). On read, a version newer
than the registered one returns `ErrSchemaTooNew`; a version older than the
oldest migration returns `ErrNoMigrationPath`. Migrations must form an unbroken
run ending one step below the current version, and a misconfigured chain panics
at construction.

## Notes and limitations

- Only **named, exported** struct types (and pointers to them) are supported;
  anonymous and unnamed types are rejected.
- A type must be **registered before it can be deserialized**; deserializing an
  unregistered type returns `ErrTypeNotRegistered`.
- The envelope is **always JSON** regardless of the codec — only the `data`
  payload uses the chosen codec.
- `Serializer` is **safe for concurrent use** by multiple goroutines once
  constructed.

## License

Released under the [MIT License](LICENSE).
