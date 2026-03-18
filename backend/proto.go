package main

import (
	"strings"

	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
)

// maxRecursiveDepth limits how many times the same message type can appear
// on a single ancestor path when building the field tree. This prevents
// infinite recursion in self-referencing proto types (e.g. a tree node
// with a repeated field of its own type).
const maxRecursiveDepth = 3

// describeMessage builds a JSON-serializable field tree for the frontend form builder.
// This is the main entry point — it starts with a fresh recursion tracker.
func describeMessage(md protoreflect.MessageDescriptor) []map[string]any {
	return describeMessageOnPath(md, make(map[protoreflect.FullName]int))
}

// describeMessageOnPath walks a message descriptor and builds a field tree.
// typeCount tracks how many times each message type has appeared on the current
// ancestor path, so we can stop at maxRecursiveDepth.
func describeMessageOnPath(md protoreflect.MessageDescriptor, typeCount map[protoreflect.FullName]int) []map[string]any {
	fields := make([]map[string]any, 0)
	seen := make(map[protoreflect.Name]bool) // tracks oneofs already emitted

	for i := 0; i < md.Fields().Len(); i++ {
		fd := md.Fields().Get(i)
		od := fd.ContainingOneof()

		// Real oneofs (not synthetic proto3-optional wrappers) are emitted once
		// when we encounter their first member field.
		if od != nil && !od.IsSynthetic() {
			if seen[od.Name()] {
				continue
			}
			seen[od.Name()] = true
			fields = append(fields, describeOneofOnPath(od, typeCount))
			continue
		}

		fields = append(fields, describeFieldOnPath(fd, typeCount))
	}
	return fields
}

// describeOneofOnPath builds the schema for a oneof group, listing all
// its possible variant fields.
func describeOneofOnPath(od protoreflect.OneofDescriptor, typeCount map[protoreflect.FullName]int) map[string]any {
	options := make([]map[string]any, od.Fields().Len())
	for i := range options {
		options[i] = describeFieldOnPath(od.Fields().Get(i), typeCount)
	}
	return map[string]any{
		"name":         string(od.Name()),
		"type":         "oneof",
		"oneofOptions": options,
	}
}

// describeFieldOnPath builds the schema for a single field.
// For message-typed fields, it recurses into the message's own fields,
// respecting the recursion depth limit.
func describeFieldOnPath(fd protoreflect.FieldDescriptor, typeCount map[protoreflect.FullName]int) map[string]any {
	f := map[string]any{
		"name":       string(fd.Name()),
		"number":     int32(fd.Number()),
		"repeated":   fd.Cardinality() == protoreflect.Repeated,
		"optional":   fd.HasOptionalKeyword(),
		"deprecated": isFieldDeprecated(fd),
	}

	if fd.IsMap() {
		f["type"] = "map"
		f["repeated"] = false
		f["optional"] = false
		f["mapKey"] = describeMapComponentOnPath(fd.MapKey(), typeCount)
		f["mapValue"] = describeMapComponentOnPath(fd.MapValue(), typeCount)
		return f
	}

	switch fd.Kind() {
	case protoreflect.MessageKind:
		fullName := fd.Message().FullName()
		f["type"] = "message"
		f["messageType"] = string(fullName)

		if typeCount[fullName] >= maxRecursiveDepth {
			// Recursion limit — return empty fields so the UI shows
			// a disabled toggle instead of expanding forever.
			f["fields"] = []map[string]any{}
		} else {
			// Track this type on the current path, recurse, then untrack.
			// This allows the same type to appear in parallel branches
			// but not infinitely deep on a single path.
			typeCount[fullName]++
			f["fields"] = describeMessageOnPath(fd.Message(), typeCount)
			typeCount[fullName]--
		}

	case protoreflect.EnumKind:
		type enumVal struct {
			Name   string `json:"name"`
			Number int    `json:"number"`
		}
		vals := make([]enumVal, fd.Enum().Values().Len())
		for i := range vals {
			v := fd.Enum().Values().Get(i)
			vals[i] = enumVal{Name: string(v.Name()), Number: int(v.Number())}
		}
		f["type"] = "enum"
		f["enumType"] = string(fd.Enum().FullName())
		f["enumValues"] = vals

	default:
		// Scalar types: strip the "Kind" suffix and lowercase.
		// e.g. "StringKind" → "string", "Int32Kind" → "int32"
		f["type"] = strings.ToLower(strings.TrimPrefix(fd.Kind().String(), "Kind"))
	}

	return f
}

// describeMapComponentOnPath builds schema metadata for map key/value components.
func describeMapComponentOnPath(fd protoreflect.FieldDescriptor, typeCount map[protoreflect.FullName]int) map[string]any {
	c := map[string]any{}

	switch fd.Kind() {
	case protoreflect.MessageKind:
		fullName := fd.Message().FullName()
		c["type"] = "message"
		c["messageType"] = string(fullName)
		if typeCount[fullName] >= maxRecursiveDepth {
			c["fields"] = []map[string]any{}
		} else {
			typeCount[fullName]++
			c["fields"] = describeMessageOnPath(fd.Message(), typeCount)
			typeCount[fullName]--
		}

	case protoreflect.EnumKind:
		type enumVal struct {
			Name   string `json:"name"`
			Number int    `json:"number"`
		}
		vals := make([]enumVal, fd.Enum().Values().Len())
		for i := range vals {
			v := fd.Enum().Values().Get(i)
			vals[i] = enumVal{Name: string(v.Name()), Number: int(v.Number())}
		}
		c["type"] = "enum"
		c["enumType"] = string(fd.Enum().FullName())
		c["enumValues"] = vals

	default:
		c["type"] = strings.ToLower(strings.TrimPrefix(fd.Kind().String(), "Kind"))
	}

	return c
}

// isFieldDeprecated checks the proto field options for the deprecated flag.
func isFieldDeprecated(fd protoreflect.FieldDescriptor) bool {
	opts, ok := fd.Options().(*descriptorpb.FieldOptions)
	if !ok || opts == nil {
		return false
	}
	return opts.GetDeprecated()
}

