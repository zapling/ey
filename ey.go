package ey

import (
	"bytes"

	"gopkg.in/yaml.v3"
)

func Parse(data any) (*Node, error) {
	var (
		dataBytes []byte
		ok        bool
		err       error
	)
	dataBytes, ok = data.([]byte)
	if !ok {
		dataBytes, err = yaml.Marshal(data)
		if err != nil {
			return nil, err
		}
	}

	var root yaml.Node
	if err := yaml.Unmarshal(dataBytes, &root); err != nil {
		return nil, err
	}

	if root.Kind == 0 || data == nil {
		root.Kind = yaml.DocumentNode
		root.Content = []*yaml.Node{
			{
				Kind: yaml.MappingNode,
			},
		}
	}

	return &Node{Node: &root}, nil
}

type Node struct {
	*yaml.Node
	parent *yaml.Node
	key    string
}

func (n *Node) Marshal() ([]byte, error) {
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	err := enc.Encode(n.Node)
	return buf.Bytes(), err
}

func (n *Node) String() string {
	out, _ := n.Marshal()
	return string(out)
}

// rootMapping returns the inner MappingNode for a DocumentNode, creating one
// if absent.  If the node is already a MappingNode it is returned as-is.
// Returns nil when the node cannot be resolved to a mapping.
func rootMapping(node *yaml.Node) *yaml.Node {
	if node == nil {
		return nil
	}
	if node.Kind == yaml.DocumentNode {
		if len(node.Content) == 0 {
			m := &yaml.Node{Kind: yaml.MappingNode}
			node.Content = append(node.Content, m)
		}
		return node.Content[0]
	}
	if node.Kind == yaml.MappingNode {
		return node
	}
	return nil
}

// Dig traverses the path, creating missing maps along the way.
// With no arguments it returns the root mapping node.
func (n *Node) Dig(path ...string) *Node {
	if n == nil || n.Node == nil {
		return &Node{}
	}

	node := rootMapping(n.Node)
	if node == nil {
		return &Node{}
	}

	var parent *yaml.Node
	var lastKey string

	for _, key := range path {
		parent = node
		lastKey = key

		child := findValue(node, key)
		if child == nil {
			keyNode := &yaml.Node{Kind: yaml.ScalarNode, Value: key}
			valNode := &yaml.Node{Kind: yaml.MappingNode}
			node.Content = append(node.Content, keyNode, valNode)
			child = valNode
		}
		node = child
	}

	return &Node{Node: node, parent: parent, key: lastKey}
}

// Get retrieves a value at the given path, returns empty Node if not found.
// Does NOT create missing paths (unlike Dig)
func (n *Node) Get(path ...string) *Node {
	if n == nil || n.Node == nil {
		return &Node{}
	}

	node := rootMapping(n.Node)
	if node == nil {
		return &Node{}
	}

	var parent *yaml.Node
	var lastKey string

	for _, key := range path {
		parent = node
		lastKey = key

		child := findValue(node, key)
		if child == nil {
			return &Node{}
		}
		node = child
	}

	return &Node{Node: node, parent: parent, key: lastKey}
}

// mapping returns the MappingNode to mutate, handling both DocumentNode
// wrappers and plain MappingNodes.  Returns nil when mutation is not valid.
func (n *Node) mapping() *yaml.Node {
	if n == nil || n.Node == nil {
		return nil
	}
	return rootMapping(n.Node)
}

// Set sets a key to a value
func (n *Node) Set(key string, value any) *Node {
	node, err := newYamlNode(value)
	if err != nil {
		return n
	}
	return n.setNode(key, node, true)
}

// SetDefault sets key to a value, if the key does not already exist
func (n *Node) SetDefault(key string, value any) *Node {
	node, err := newYamlNode(value)
	if err != nil {
		return n
	}
	return n.setNode(key, node, false)
}

func (n *Node) setNode(key string, node *yaml.Node, overrideExisting bool) *Node {
	m := n.mapping()
	if m == nil {
		return n
	}

	if existing := findValue(m, key); existing != nil && overrideExisting {
		*existing = *node
	} else if overrideExisting {
		m.Content = append(
			m.Content,
			newScalarNode("", key),
			node,
		)
	}
	return n
}

func (n *Node) Append(values ...any) *Node {
	if n == nil || n.Node == nil {
		return n
	}

	for _, value := range values {
		node, err := newYamlNode(value)
		if err != nil {
			return n
		}

		if n.Kind != yaml.SequenceNode {
			n.Kind = yaml.SequenceNode
			n.Content = nil
		}
		n.Content = append(n.Content, node)
	}

	return n
}

func (n *Node) At(index int) *Node {
	if n == nil || n.Node == nil || n.Kind != yaml.SequenceNode {
		return &Node{}
	}
	if index < 0 || index >= len(n.Content) {
		return &Node{}
	}
	return &Node{Node: n.Content[index], parent: n.Node}
}

func (n *Node) First() *Node {
	return n.At(0)
}

func (n *Node) Last() *Node {
	if n == nil || n.Node == nil || n.Kind != yaml.SequenceNode || len(n.Content) == 0 {
		return &Node{}
	}
	return n.At(len(n.Content) - 1)
}

// Delete removes the key
func (n *Node) Delete(key string) *Node {
	m := n.mapping()
	if m == nil {
		return n
	}
	for i := 0; i < len(m.Content); i += 2 {
		if m.Content[i].Value == key {
			m.Content = append(m.Content[:i], m.Content[i+2:]...)
			break
		}
	}
	return n
}

func (n *Node) IsEmpty() bool {
	if n == nil || n.Node == nil {
		return true
	}
	m := rootMapping(n.Node)
	if m == nil {
		return true
	}
	return len(m.Content) == 0
}

func (n *Node) Slice() []*Node {
	var nodes []*Node
	if n == nil || n.Node == nil || n.Kind != yaml.SequenceNode {
		return nodes
	}
	for _, item := range n.Content {
		nodes = append(nodes, &Node{Node: item, parent: n.Node})
	}
	return nodes
}

func (n *Node) ForEach(fn func(i int, node *Node)) {
	for i, node := range n.Slice() {
		fn(i, node)
	}
}

// Value returns the node value
func (n *Node) Value() string {
	if n == nil || n.Node == nil {
		return ""
	}
	return n.Node.Value
}

// findValue finds a value node for a given key in a mapping
func findValue(node *yaml.Node, key string) *yaml.Node {
	if node.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i < len(node.Content); i += 2 {
		if node.Content[i].Value == key {
			return node.Content[i+1]
		}
	}
	return nil
}
