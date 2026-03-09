# ey (edit-yaml)

A fluent, chainable API for building and manipulating YAML documents in Go.

`ey` wraps [`gopkg.in/yaml.v3`](https://pkg.go.dev/gopkg.in/yaml.v3) and lets you construct, traverse, and mutate YAML structures without manually managing nodes — useful for code generators, config builders, and CLI tools.

## Installation

```bash
go get github.com/your-org/ey
```

## Quick Start

```go
doc, err := ey.Parse(nil) // start with an empty document
if err != nil {
    log.Fatal(err)
}

doc.Set("name", "my-service").
    Set("version", "1.0.0").
    Dig("server").
        Set("host", "localhost").
        Set("port", 8080)

fmt.Print(doc.String())
```

```yaml
name: "my-service"
version: 1.0.0
server:
  host: "localhost"
  port: 8080
```

## Usage

### Parsing

```go
// From a raw YAML byte slice
doc, err := ey.Parse([]byte("key: value"))

// From any Go value (marshalled via yaml.Marshal first)
doc, err := ey.Parse(myStruct)

// Empty document
doc, err := ey.Parse(nil)
```

### Reading values

```go
// Get a value (returns empty Node if path doesn't exist — never panics)
node := doc.Get("server", "host")
fmt.Println(node.Value()) // "localhost"

// Traverse into nested maps
node := doc.Get("deeply", "nested", "key")
```

### Writing values

```go
// Set (creates or overwrites)
doc.Set("key", "value")

// SetDefault (no-op if key already exists)
doc.SetDefault("timeout", 30)

// Delete a key
doc.Delete("deprecated-field")
```

`Set()` accepts any Go type:

```go
// Primitives
doc.Set("name", "my-service")        // name: "my-service"
doc.Set("replicas", 3)               // replicas: 3
doc.Set("ratio", 0.75)               // ratio: 0.75
doc.Set("enabled", true)             // enabled: true

// Unquoted string (Raw)
doc.Set("policy", ey.Raw("Always"))  // policy: Always

// Slice
doc.Set("tags", []string{"a", "b"})  // tags:
                                     //   - "a"
                                     //   - "b"

// Map
doc.Set("labels", map[string]string{
    "env":  "prod",
    "team": "platform",
})
// labels:
//   env: "prod"
//   team: "platform"

// Struct (uses yaml tags if present)
type Resources struct {
    CPU    string `yaml:"cpu"`
    Memory string `yaml:"memory"`
}
doc.Dig("limits").Set("resources", Resources{CPU: "500m", Memory: "128Mi"})
// limits:
//   resources:
//     cpu: "500m"
//     memory: "128Mi"

// Nil
doc.Set("optional", nil) // optional: null
```

### Digging into nested paths

`Dig` traverses a path, **creating intermediate maps as needed**. It returns the node at the end of the path, ready for further chaining.

```go
doc.Dig("resources", "limits").
    Set("cpu", "500m").
    Set("memory", "128Mi")
```

```yaml
resources:
  limits:
    cpu: "500m"
    memory: "128Mi"
```

### Raw (unquoted) string values

By default, string values are double-quoted in the YAML output. Wrap a string in `ey.Raw` to suppress quoting — useful for YAML anchors, references, or any value that must appear bare.

```go
doc.Set("pull_policy", ey.Raw("IfNotPresent"))
doc.Set("ref", ey.Raw("*my-anchor"))
```

```yaml
pull_policy: IfNotPresent
ref: *my-anchor
```

### Working with sequences

```go
doc.Dig("ports").Append(8080, 9090, 443)

doc.Get("ports").ForEach(func(i int, n *ey.Node) {
    fmt.Println(i, n.Value())
})

first := doc.Get("ports").First()
last  := doc.Get("ports").Last()
item  := doc.Get("ports").At(1)

nodes := doc.Get("ports").Slice() // []*ey.Node
```

### Marshalling

```go
// As a []byte
b, err := doc.Marshal()

// As a string
s := doc.String()
```

## API Reference

| Method | Description |
|---|---|
| `Raw(string)` | String type whose value is written unquoted |
| `Parse(data any) (*Node, error)` | Parse a YAML byte slice, any Go value, or `nil` |
| `Get(path ...string) *Node` | Traverse a path; returns empty node if not found |
| `Dig(path ...string) *Node` | Traverse a path, creating missing maps along the way |
| `Set(key string, value any) *Node` | Set a key, overwriting if it exists |
| `SetDefault(key string, value any) *Node` | Set a key only if it doesn't already exist |
| `Delete(key string) *Node` | Remove a key from the current mapping |
| `Append(values ...any) *Node` | Append values to a sequence node |
| `At(index int) *Node` | Get a sequence item by index |
| `First() *Node` | Get the first sequence item |
| `Last() *Node` | Get the last sequence item |
| `Slice() []*Node` | Return all sequence items as a slice |
| `ForEach(fn func(i int, node *Node))` | Iterate over sequence items |
| `Value() string` | Return the scalar string value of the node |
| `IsEmpty() bool` | Report whether the node has no content |
| `Marshal() ([]byte, error)` | Encode the document to YAML bytes |
| `String() string` | Encode the document to a YAML string |
