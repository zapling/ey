package ey

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// Raw values will not be quoted
type Raw string

func newYamlNode(v any) (*yaml.Node, error) {
	rv := reflect.ValueOf(v)

	// Dereference pointer and handle nil
	for rv.Kind() == reflect.Pointer || rv.Kind() == reflect.Interface {
		if rv.IsNil() {
			return newNullNode(), nil
		}
		rv = rv.Elem()
	}

	if !rv.IsValid() {
		return newNullNode(), nil
	}

	iface := rv.Interface()

	// Check if it's already a *yaml.Node
	if n, ok := iface.(*yaml.Node); ok {
		return n, nil
	}

	switch rv.Kind() {
	case reflect.Bool:
		return newScalarNode("!!bool", strconv.FormatBool(rv.Bool())), nil

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return newScalarNode("!!int", strconv.FormatInt(rv.Int(), 10)), nil

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return newScalarNode("!!int", strconv.FormatUint(rv.Uint(), 10)), nil

	case reflect.Float32, reflect.Float64:
		s := strconv.FormatFloat(rv.Float(), 'g', -1, 64)
		return newScalarNode("!!float", s), nil

	case reflect.String:
		n := newScalarNode("!!str", rv.String())
		n.Style = yaml.DoubleQuotedStyle
		if rt := reflect.TypeOf(v); rt.PkgPath() == "github.com/zapling/ey" && rt.Name() == "Raw" {
			n.Style = 0
		}
		return n, nil

	case reflect.Slice:
		if rv.IsNil() {
			return newNullNode(), nil
		}
		// []byte → binary scalar.
		if rv.Type().Elem().Kind() == reflect.Uint8 {
			n := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!binary"}
			if err := n.Encode(rv.Bytes()); err != nil {
				return nil, err
			}
			return unwrapDocument(n), nil
		}
		// return marshalIntoNode(iface)
		return newSeqNode(rv)

	case reflect.Array:
		// return marshalIntoNode(iface)
		return newSeqNode(rv)

	case reflect.Map:
		// return marshalIntoNode(iface)
		return newMapNode(rv)

	case reflect.Struct:
		return newStructNode(rv)
		// return marshalIntoNode(iface)

	default:
		// Last resort: fmt.Stringer or plain sprintf.
		if s, ok := iface.(fmt.Stringer); ok {
			return newScalarNode("!!str", s.String()), nil
		}
		return newScalarNode("!!str", fmt.Sprintf("%v", iface)), nil
	}
}

func newNullNode() *yaml.Node {
	return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!null", Value: "null"}
}

func newScalarNode(tag, value string) *yaml.Node {
	return &yaml.Node{Kind: yaml.ScalarNode, Tag: tag, Value: value}
}

func newSeqNode(rv reflect.Value) (*yaml.Node, error) {
	n := &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq"}
	for i := range rv.Len() {
		child, err := newYamlNode(rv.Index(i).Interface())
		if err != nil {
			return nil, fmt.Errorf("index %d: %w", i, err)
		}
		n.Content = append(n.Content, child)
	}
	return n, nil
}

func newMapNode(rv reflect.Value) (*yaml.Node, error) {
	n := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
	for _, k := range rv.MapKeys() {
		kNode, err := newYamlNode(k.Interface())
		if err != nil {
			return nil, fmt.Errorf("map key: %w", err)
		}

		// Remove any style from the map key
		kNode.Style = 0

		vNode, err := newYamlNode(rv.MapIndex(k).Interface())
		if err != nil {
			return nil, fmt.Errorf("map value for key %v: %w", k, err)
		}
		n.Content = append(n.Content, kNode, vNode)
	}
	return n, nil
}

func newStructNode(rv reflect.Value) (*yaml.Node, error) {
	n := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
	rt := rv.Type()
	for i := range rt.NumField() {
		f := rt.Field(i)
		if !f.IsExported() {
			continue
		}
		fv := rv.Field(i)

		// Determine the field name from yaml tag or field name.
		name := strings.ToLower(f.Name)
		if tag, ok := f.Tag.Lookup("yaml"); ok {
			if tag == "-" {
				continue
			}
			if parts := strings.Split(tag, ","); parts[0] != "" {
				name = parts[0]
			}
		}

		kNode := newScalarNode("!!str", name)
		vNode, err := newYamlNode(fv.Interface())
		if err != nil {
			return nil, fmt.Errorf("field %s: %w", f.Name, err)
		}

		n.Content = append(n.Content, kNode, vNode)
	}
	return n, nil
}

// func marshalIntoNode(v any) (*yaml.Node, error) {
// 	var doc yaml.Node
// 	if err := doc.Encode(v); err != nil {
// 		return nil, err
// 	}
// 	return unwrapDocument(&doc), nil
// }

// unwrapDocument strips the outer !!document wrapper that yaml.Node.Encode
// always adds.
func unwrapDocument(n *yaml.Node) *yaml.Node {
	if n.Kind == yaml.DocumentNode && len(n.Content) == 1 {
		return n.Content[0]
	}
	return n
}
