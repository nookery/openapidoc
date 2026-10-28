package spec

import (
	"encoding/json"
	"reflect"
	"strconv"
	"strings"
	"time"
)

// Schema is a JSON Schema 2020-12 object, the schema dialect used by
// OpenAPI 3.1.
type Schema struct {
	Ref         string             `json:"$ref,omitempty"`
	Type        string             `json:"type,omitempty"`
	Format      string             `json:"format,omitempty"`
	Title       string             `json:"title,omitempty"`
	Description string             `json:"description,omitempty"`
	Properties  map[string]*Schema `json:"properties,omitempty"`
	Required    []string           `json:"required,omitempty"`
	Items       *Schema            `json:"items,omitempty"`
	Enum        []any              `json:"enum,omitempty"`
	Example     any                `json:"example,omitempty"`
	Default     any                `json:"default,omitempty"`

	// AdditionalProperties describes the value schema of an object with
	// arbitrary keys. A non-nil empty schema means "any value is allowed".
	AdditionalProperties *Schema `json:"additionalProperties,omitempty"`

	AllOf []*Schema `json:"allOf,omitempty"`
	OneOf []*Schema `json:"oneOf,omitempty"`
	AnyOf []*Schema `json:"anyOf,omitempty"`

	Minimum          *float64 `json:"minimum,omitempty"`
	Maximum          *float64 `json:"maximum,omitempty"`
	ExclusiveMinimum *float64 `json:"exclusiveMinimum,omitempty"`
	ExclusiveMaximum *float64 `json:"exclusiveMaximum,omitempty"`
	MinLength        *int     `json:"minLength,omitempty"`
	MaxLength        *int     `json:"maxLength,omitempty"`
	Pattern          string   `json:"pattern,omitempty"`
	MinItems         *int     `json:"minItems,omitempty"`
	MaxItems         *int     `json:"maxItems,omitempty"`
	UniqueItems      bool     `json:"uniqueItems,omitempty"`
}

// timeType and rawMessageType are recognised specially during reflection.
var (
	timeType       = reflect.TypeOf(time.Time{})
	rawMessageType = reflect.TypeOf(json.RawMessage{})
)

// Reflector converts Go types into schemas. Named struct types are
// registered as reusable components and referenced with $ref, so recursive
// types terminate instead of expanding forever.
type Reflector struct {
	schemas map[string]*Schema
	names   map[reflect.Type]string
}

// NewReflector creates an empty Reflector.
func NewReflector() *Reflector {
	return &Reflector{
		schemas: make(map[string]*Schema),
		names:   make(map[reflect.Type]string),
	}
}

// Schemas returns the component schemas collected so far.
func (r *Reflector) Schemas() map[string]*Schema {
	return r.schemas
}

// SchemaOf returns the schema describing t.
func (r *Reflector) SchemaOf(t reflect.Type) *Schema {
	if t == nil {
		return &Schema{}
	}
	return r.schemaOf(t, map[reflect.Type]bool{})
}

// schemaOf builds the schema for t. stack guards against recursive types
// that are reached through a named struct rather than a pointer.
func (r *Reflector) schemaOf(t reflect.Type, stack map[reflect.Type]bool) *Schema {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}

	switch t {
	case timeType:
		return &Schema{Type: "string", Format: "date-time"}
	case rawMessageType:
		return &Schema{}
	}

	switch t.Kind() {
	case reflect.Bool:
		return &Schema{Type: "boolean"}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32:
		return &Schema{Type: "integer", Format: "int32"}
	case reflect.Int64:
		return &Schema{Type: "integer", Format: "int64"}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32:
		return &Schema{Type: "integer", Format: "int32"}
	case reflect.Uint64:
		return &Schema{Type: "integer", Format: "int64"}
	case reflect.Float32:
		return &Schema{Type: "number", Format: "float"}
	case reflect.Float64:
		return &Schema{Type: "number", Format: "double"}
	case reflect.String:
		return &Schema{Type: "string"}
	case reflect.Slice, reflect.Array:
		if t.Elem().Kind() == reflect.Uint8 {
			// []byte is encoded as a base64 string by encoding/json.
			return &Schema{Type: "string", Format: "binary"}
		}
		return &Schema{Type: "array", Items: r.schemaOf(t.Elem(), stack)}
	case reflect.Map:
		return &Schema{
			Type:                 "object",
			AdditionalProperties: r.schemaOf(t.Elem(), stack),
		}
	case reflect.Struct:
		return r.refFor(t, stack)
	case reflect.Interface:
		// An interface accepts anything.
		return &Schema{}
	default:
		return &Schema{}
	}
}

// refFor registers t as a component schema and returns a reference to it.
func (r *Reflector) refFor(t reflect.Type, stack map[reflect.Type]bool) *Schema {
	if name, ok := r.names[t]; ok {
		return &Schema{Ref: componentRef(name)}
	}

	name := r.uniqueName(t)
	r.names[t] = name

	// Install a placeholder before walking the fields so that a type
	// referring to itself resolves to this component instead of recursing.
	placeholder := &Schema{Type: "object"}
	r.schemas[name] = placeholder

	props, required := r.structFields(t, stack)
	placeholder.Properties = props
	placeholder.Required = required

	return &Schema{Ref: componentRef(name)}
}

// uniqueName derives a component name for t, disambiguating collisions
// between identically named types from different packages.
func (r *Reflector) uniqueName(t reflect.Type) string {
	base := t.Name()
	if base == "" {
		base = "Anonymous"
	}
	name := base
	if _, exists := r.schemas[name]; exists {
		if pkg := t.PkgPath(); pkg != "" {
			short := pkg
			if idx := strings.LastIndex(pkg, "/"); idx >= 0 {
				short = pkg[idx+1:]
			}
			name = short + "_" + base
			for i := 2; ; i++ {
				if _, exists := r.schemas[name]; !exists {
					break
				}
				name = short + "_" + base + "_" + strconv.Itoa(i)
			}
		}
	}
	return name
}

// structFields collects the properties and required list of a struct.
func (r *Reflector) structFields(t reflect.Type, stack map[reflect.Type]bool) (map[string]*Schema, []string) {
	props := make(map[string]*Schema)
	var required []string
	r.collectFields(t, props, &required, stack)
	return props, required
}

// collectFields walks the fields of t, inlining anonymous embedded structs.
func (r *Reflector) collectFields(t reflect.Type, props map[string]*Schema, required *[]string, stack map[reflect.Type]bool) {
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)

		// Unexported fields are not serialised.
		if f.PkgPath != "" && !f.Anonymous {
			continue
		}

		name, opts := parseJSONTag(f.Tag.Get("json"))
		if name == "-" {
			continue
		}

		// An anonymous embedded struct is inlined into the parent object.
		if f.Anonymous && name == "" {
			ft := f.Type
			for ft.Kind() == reflect.Pointer {
				ft = ft.Elem()
			}
			if ft.Kind() == reflect.Struct && ft != timeType {
				r.collectFields(ft, props, required, stack)
				continue
			}
		}

		if name == "" {
			name = f.Name
		}

		schema := r.schemaOf(f.Type, stack)
		applyFieldTags(schema, f.Tag.Get("jsonschema"))

		props[name] = schema

		if isRequired(f, opts) {
			*required = append(*required, name)
		}
	}
}

// isRequired reports whether the field must be present in the payload.
// A field is optional when it is a pointer, carries the omitempty option,
// or is explicitly marked optional in the jsonschema tag.
func isRequired(f reflect.StructField, jsonOpts map[string]bool) bool {
	if jsonOpts["omitempty"] {
		return false
	}
	if strings.Contains(f.Tag.Get("jsonschema"), "omitempty") {
		return false
	}
	if f.Type.Kind() == reflect.Pointer {
		return false
	}
	return true
}

// parseJSONTag splits a json struct tag into its name and options.
func parseJSONTag(tag string) (string, map[string]bool) {
	opts := map[string]bool{}
	if tag == "" {
		return "", opts
	}
	parts := strings.Split(tag, ",")
	for _, o := range parts[1:] {
		opts[o] = true
	}
	return parts[0], opts
}

// applyFieldTags overrides schema fields from a jsonschema struct tag.
//
// The supported keys mirror the common JSON Schema vocabulary:
//
//	jsonschema:"required,description=...,example=...,enum=a|b,minimum=1,..."
func applyFieldTags(schema *Schema, tag string) {
	if tag == "" {
		return
	}
	for _, part := range strings.Split(tag, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		key, value, hasValue := strings.Cut(part, "=")
		key = strings.TrimSpace(key)

		if !hasValue {
			switch key {
			case "required":
				// Handled by the caller, which owns the required list.
			case "deprecated":
				schema.Description = strings.TrimSpace(schema.Description + " (deprecated)")
			case "uniqueItems":
				schema.UniqueItems = true
			}
			continue
		}

		switch key {
		case "description":
			schema.Description = value
		case "title":
			schema.Title = value
		case "format":
			schema.Format = value
		case "pattern":
			schema.Pattern = value
		case "default":
			schema.Default = value
		case "example":
			schema.Example = value
		case "enum":
			schema.Enum = nil
			for _, item := range strings.Split(value, "|") {
				schema.Enum = append(schema.Enum, item)
			}
		case "minimum":
			if f, err := strconv.ParseFloat(value, 64); err == nil {
				schema.Minimum = &f
			}
		case "maximum":
			if f, err := strconv.ParseFloat(value, 64); err == nil {
				schema.Maximum = &f
			}
		case "exclusiveMinimum":
			if f, err := strconv.ParseFloat(value, 64); err == nil {
				schema.ExclusiveMinimum = &f
			}
		case "exclusiveMaximum":
			if f, err := strconv.ParseFloat(value, 64); err == nil {
				schema.ExclusiveMaximum = &f
			}
		case "minLength":
			if n, err := strconv.Atoi(value); err == nil {
				schema.MinLength = &n
			}
		case "maxLength":
			if n, err := strconv.Atoi(value); err == nil {
				schema.MaxLength = &n
			}
		case "minItems":
			if n, err := strconv.Atoi(value); err == nil {
				schema.MinItems = &n
			}
		case "maxItems":
			if n, err := strconv.Atoi(value); err == nil {
				schema.MaxItems = &n
			}
		}
	}
}

// componentRef returns the JSON pointer of a component schema.
func componentRef(name string) string {
	return "#/components/schemas/" + name
}
