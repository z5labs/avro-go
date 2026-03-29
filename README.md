# avro

A library for working with [Apache Avro](https://avro.apache.org/) encoded data in Go.

## Packages

| Package | Description |
|---------|-------------|
| `github.com/z5labs/avro-go` | Binary encoding and decoding of Avro primitives |
| `github.com/z5labs/avro-go/idl` | Avro IDL tokenizer, parser, and printer |

---

## `avro` — Binary Encoding

### Marshaling

Implement `BinaryMarshaler` on your type and call `MarshalBinary` to encode it:

```go
type Message struct {
    Content string
}

func (m Message) MarshalAvroBinary(w *avro.BinaryWriter) error {
    return w.WriteString(m.Content)
}

var buf bytes.Buffer
err := avro.MarshalBinary(&buf, Message{Content: "hello"})
```

`BinaryWriter` exposes one write method per Avro primitive:

| Method | Avro type |
|--------|-----------|
| `WriteBool(bool)` | boolean |
| `WriteInt(int32)` | int |
| `WriteLong(int64)` | long |
| `WriteFloat(float32)` | float |
| `WriteDouble(float64)` | double |
| `WriteBytes([]byte)` | bytes |
| `WriteFixed([]byte)` | fixed |
| `WriteString(string)` | string |

### Unmarshaling

Implement `BinaryUnmarshaler` on your type and call `UnmarshalBinary` to decode it:

```go
func (m *Message) UnmarshalAvroBinary(r *avro.BinaryReader) error {
    var err error
    m.Content, err = r.ReadString()
    return err
}

var msg Message
err := avro.UnmarshalBinary(bytes.NewReader(data), &msg)
```

`BinaryReader` mirrors `BinaryWriter` with corresponding `Read*` methods.

### Single-Object Encoding

The [Avro single-object encoding](https://avro.apache.org/docs/current/specification/#single-object-encoding) prepends a 2-byte magic header and an 8-byte schema fingerprint to the binary payload, allowing readers to identify the schema at runtime.

Implement `SingleObjectMarshaler` (embeds `BinaryMarshaler` plus a `Fingerprint() [8]byte` method) and call `MarshalSingleObject`:

```go
func (m Message) Fingerprint() [8]byte {
    var fp [8]byte
    binary.LittleEndian.PutUint64(fp[:], avro.Fingerprint64([]byte(`"string"`)))
    return fp
}

var buf bytes.Buffer
err := avro.MarshalSingleObject(&buf, msg)
```

Decode with `SingleObjectUnmarshaler` and `UnmarshalSingleObject`:

```go
var msg Message
err := avro.UnmarshalSingleObject(r, &msg)
```

`UnmarshalSingleObject` returns `ErrBadMagic` when the header is invalid and `ErrFingerprintMismatch` when the schema fingerprint in the payload does not match the one returned by `Fingerprint()`.

### Schema Fingerprinting

`Fingerprint64` computes the 64-bit Rabin fingerprint (CRC-64-AVRO) of a schema JSON string, as defined in the Avro specification:

```go
fp := avro.Fingerprint64([]byte(`"string"`))
```

---

## `idl` — Avro IDL

The `idl` package parses [Avro IDL](https://avro.apache.org/docs/current/idl-language/) source files into an AST and can print an AST back to IDL text.

### Parsing

`Parse` reads an Avro IDL source from any `io.Reader` and returns a `*File` AST:

```go
f, err := idl.Parse(strings.NewReader(`
    namespace com.example;
    schema record User {
        string name;
        int    age;
    }
`))
```

The `File` struct contains either a `*Schema` or a `*Protocol`. A `*Schema` holds the top-level named types (`Record`, `Enum`, `Fixed`) and primitive type identifiers.

### Printing

`Print` formats a `*File` AST back to Avro IDL text:

```go
var buf bytes.Buffer
err := idl.Print(&buf, f)
fmt.Println(buf.String())
```

### Tokenizing

`Tokenize` exposes the low-level lexer as an `iter.Seq2[Token, error]` iterator (Go 1.23+):

```go
for tok, err := range idl.Tokenize(r) {
    if err != nil {
        // handle error
    }
    fmt.Println(tok)
}
```

Token types include `TokenComment`, `TokenDocComment`, `TokenIdentifier`, `TokenSymbol`, `TokenString`, `TokenNumber`, and `TokenAnnotation`.

---

## License

Released under the [MIT License](LICENSE).