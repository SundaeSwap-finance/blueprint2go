package blueprint

import (
	"encoding/json"
	"strings"
)

// SchemaItems represents the items field which can be either a single schema
// (for homogeneous lists) or an array of schemas (for tuples).
type SchemaItems []*Schema

// UnmarshalJSON handles both single schema objects and arrays.
func (s *SchemaItems) UnmarshalJSON(data []byte) error {
	// First try to unmarshal as an array
	var arr []*Schema
	if err := json.Unmarshal(data, &arr); err == nil {
		*s = arr
		return nil
	}

	// Then try to unmarshal as a single schema
	var single Schema
	if err := json.Unmarshal(data, &single); err != nil {
		return err
	}
	*s = []*Schema{&single}
	return nil
}

// Single returns the first item if this is a single-item list.
// Used for homogeneous list types.
func (s SchemaItems) Single() *Schema {
	if len(s) == 1 {
		return s[0]
	}
	return nil
}

// IsTuple returns true if items represents a tuple (multiple schemas).
func (s SchemaItems) IsTuple() bool {
	return len(s) > 1
}

// Schema represents a Plutus data schema from the blueprint definitions.
// It can be a primitive type (integer, bytes), a reference ($ref),
// a list, a map, a constructor, or an enum (anyOf).
type Schema struct {
	// Reference to another definition
	Ref string `json:"$ref,omitempty"`

	// Title and description for documentation
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`

	// DataType for primitive types: "integer", "bytes", "list", "map", "constructor"
	DataType string `json:"dataType,omitempty"`

	// For list types - can be a single schema (homogeneous list) or array of schemas (tuple)
	Items SchemaItems `json:"items,omitempty"`

	// For map types
	Keys   *Schema `json:"keys,omitempty"`
	Values *Schema `json:"values,omitempty"`

	// For constructor types
	Index  *int     `json:"index,omitempty"`
	Fields []Schema `json:"fields,omitempty"`

	// For enum types (union of constructors)
	AnyOf []Schema `json:"anyOf,omitempty"`
}

// IsRef returns true if this schema is a reference to another definition.
func (s *Schema) IsRef() bool {
	return s.Ref != ""
}

// RefName extracts the definition name from a $ref string.
// For example, "#/definitions/types~1Payout" returns "types/Payout".
func (s *Schema) RefName() string {
	if !s.IsRef() {
		return ""
	}
	name := strings.TrimPrefix(s.Ref, "#/definitions/")
	// Handle JSON Pointer escaping: ~1 = /, ~0 = ~
	name = strings.ReplaceAll(name, "~1", "/")
	name = strings.ReplaceAll(name, "~0", "~")
	return name
}

// IsInteger returns true if this is an integer type.
func (s *Schema) IsInteger() bool {
	return s.DataType == "integer"
}

// IsBytes returns true if this is a bytes type.
func (s *Schema) IsBytes() bool {
	return s.DataType == "bytes"
}

// IsList returns true if this is a list type.
func (s *Schema) IsList() bool {
	return s.DataType == "list"
}

// IsMap returns true if this is a map type.
func (s *Schema) IsMap() bool {
	return s.DataType == "map"
}

// IsConstructor returns true if this is a constructor type.
func (s *Schema) IsConstructor() bool {
	return s.DataType == "constructor"
}

// IsEnum returns true if this is an enum type (anyOf with multiple constructors).
func (s *Schema) IsEnum() bool {
	return len(s.AnyOf) > 0
}

// IsUnit returns true if this is a Unit/Void type.
func (s *Schema) IsUnit() bool {
	if len(s.AnyOf) == 1 {
		c := s.AnyOf[0]
		return c.IsConstructor() && c.Index != nil && *c.Index == 0 && len(c.Fields) == 0
	}
	return false
}

// IsBoolean returns true if this is a Boolean type (False/True constructors).
func (s *Schema) IsBoolean() bool {
	if len(s.AnyOf) != 2 {
		return false
	}
	return s.AnyOf[0].Title == "False" && s.AnyOf[1].Title == "True"
}

// IsOption returns true if this is an Option type (Some/None constructors).
func (s *Schema) IsOption() bool {
	if len(s.AnyOf) != 2 {
		return false
	}
	return s.AnyOf[0].Title == "Some" && s.AnyOf[1].Title == "None"
}

// OptionInnerType returns the inner type of an Option.
func (s *Schema) OptionInnerType() *Schema {
	if !s.IsOption() {
		return nil
	}
	some := s.AnyOf[0]
	if len(some.Fields) > 0 {
		return &some.Fields[0]
	}
	return nil
}

// IsOpaque returns true if this is an opaque/any Data type.
func (s *Schema) IsOpaque() bool {
	return s.DataType == "" && s.Ref == "" && len(s.AnyOf) == 0 && s.Title == "Data"
}

// IsEmpty returns true if this schema has no meaningful content.
func (s *Schema) IsEmpty() bool {
	return s.DataType == "" && s.Ref == "" && len(s.AnyOf) == 0 && s.Title == "" && len(s.Items) == 0
}

// IsSingleConstructor returns true if this is an enum with a single constructor.
func (s *Schema) IsSingleConstructor() bool {
	return len(s.AnyOf) == 1 && s.AnyOf[0].IsConstructor()
}

// UnmarshalJSON implements custom JSON unmarshaling to handle the various
// schema formats in the blueprint.
func (s *Schema) UnmarshalJSON(data []byte) error {
	// First try to unmarshal as a full schema object
	type schemaAlias Schema
	var alias schemaAlias
	if err := json.Unmarshal(data, &alias); err != nil {
		return err
	}
	*s = Schema(alias)
	return nil
}

// StandardTypeName returns the name if this is a standard Aiken/Cardano type.
func (s *Schema) StandardTypeName() string {
	refName := s.RefName()
	switch {
	case refName == "Int" || s.IsInteger():
		return "Int"
	case refName == "ByteArray" || s.IsBytes():
		return "ByteArray"
	case refName == "Data" || s.IsOpaque():
		return "Data"
	case refName == "Void" || s.IsUnit():
		return "Void"
	case strings.HasPrefix(refName, "cardano/"):
		return refName
	case strings.HasPrefix(refName, "aiken/"):
		return refName
	case strings.HasPrefix(refName, "List$"):
		return refName
	case strings.HasPrefix(refName, "Pairs$"):
		return refName
	default:
		return ""
	}
}

// IsStandardType returns true if this references a standard Aiken type.
func (s *Schema) IsStandardType() bool {
	return s.StandardTypeName() != ""
}
