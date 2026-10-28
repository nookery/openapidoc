package spec

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"
)

// ---------- fixtures ----------

// Simple is a plain struct.
type Simple struct {
	Name string `json:"name"`
}

// Tagged exercises the jsonschema struct tag.
type Tagged struct {
	Price     int     `json:"price" jsonschema:"description=Price in cents,minimum=0"`
	Title     string  `json:"title" jsonschema:"title=A title,description=Some title"`
	Tags      string  `json:"tags" jsonschema:"pattern=^[a-z]+$"`
	Kind      string  `json:"kind" jsonschema:"enum=book|disc"`
	Note      string  `json:"note,omitempty" jsonschema:"maxLength=500"`
	Secret    string  `json:"secret" jsonschema:"minimum=1,maximum=9,exclusiveMinimum=0,exclusiveMaximum=10"`
	Length    string  `json:"length" jsonschema:"minLength=1,maxLength=8"`
	Items     []int   `json:"items" jsonschema:"minItems=1,maxItems=3,uniqueItems"`
	Untagged  float64 `json:"untagged"`
	WithDef   string  `json:"with_def" jsonschema:"default=fallback,example=sample"`
	OnlyOpt   string  `json:"only_opt,omitempty"`
	EmptyTag  string  `json:"empty_tag" jsonschema:""`
	BadNumber string  `json:"bad_number" jsonschema:"minimum=notanumber"`
}

// Optional has optional and required fields.
type Optional struct {
	Required string  `json:"required"`
	Pointer  *string `json:"pointer,omitempty"`
	Omit     string  `json:"omit,omitempty"`
	Both     *int    `json:"both,omitempty"`
}

// Embedded is embedded into Other.
type Embedded struct {
	Shared string `json:"shared"`
}

// Other embeds a struct.
type Other struct {
	Embedded
	Own string `json:"own"`
}

// Recursive refers to itself.
type Recursive struct {
	Name     string       `json:"name"`
	Children []*Recursive `json:"children,omitempty"`
}

// Kinds covers the primitive kinds.
type Kinds struct {
	B     bool            `json:"b"`
	I     int             `json:"i"`
	I8    int8            `json:"i8"`
	I16   int16           `json:"i16"`
	I32   int32           `json:"i32"`
	I64   int64           `json:"i64"`
	U     uint            `json:"u"`
	U8    uint8           `json:"u8"`
	U16   uint16          `json:"u16"`
	U32   uint32          `json:"u32"`
	U64   uint64          `json:"u64"`
	F32   float32         `json:"f32"`
	F64   float64         `json:"f64"`
	S     string          `json:"s"`
	Bytes []byte          `json:"bytes"`
	Slice []string        `json:"slice"`
	Array [3]int          `json:"array"`
	Map   map[string]int  `json:"map"`
	Any   any             `json:"any"`
	Raw   json.RawMessage `json:"raw"`
}

// Ignored has fields that must not appear in the schema.
type Ignored struct {
	Exported   string `json:"exported"`
	unexported string //nolint:unused
	Skipped    string `json:"-"`
	NoTag      string
}

// ---------- basic types ----------

func TestSchemaOfPrimitives(t *testing.T) {
	r := NewReflector()

	cases := []struct {
		name       string
		value      any
		wantType   string
		wantFormat string
	}{
		{"bool", false, "boolean", ""},
		{"int", 0, "integer", "int32"},
		{"int64", int64(0), "integer", "int64"},
		{"uint", uint(0), "integer", "int32"},
		{"uint64", uint64(0), "integer", "int64"},
		{"float32", float32(0), "number", "float"},
		{"float64", float64(0), "number", "double"},
		{"string", "", "string", ""},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			schema := r.SchemaOf(reflect.TypeOf(c.value))
			if schema.Type != c.wantType {
				t.Errorf("type = %q, want %q", schema.Type, c.wantType)
			}
			if schema.Format != c.wantFormat {
				t.Errorf("format = %q, want %q", schema.Format, c.wantFormat)
			}
		})
	}
}

func TestSchemaOfNilType(t *testing.T) {
	r := NewReflector()
	if schema := r.SchemaOf(nil); schema == nil {
		t.Fatal("schema is nil")
	}
}

func TestSchemaOfPointerUnwraps(t *testing.T) {
	r := NewReflector()
	schema := r.SchemaOf(reflect.TypeOf(&Simple{}))
	if schema.Ref != componentRef("Simple") {
		t.Errorf("ref = %q, want %q", schema.Ref, componentRef("Simple"))
	}
}

func TestSchemaOfTime(t *testing.T) {
	r := NewReflector()
	schema := r.SchemaOf(reflect.TypeOf(time.Time{}))
	if schema.Type != "string" {
		t.Errorf("type = %q, want string", schema.Type)
	}
	if schema.Format != "date-time" {
		t.Errorf("format = %q, want date-time", schema.Format)
	}
}

func TestSchemaOfRawMessage(t *testing.T) {
	r := NewReflector()
	schema := r.SchemaOf(reflect.TypeOf(json.RawMessage{}))
	if schema.Type != "" {
		t.Errorf("type = %q, want empty", schema.Type)
	}
}

func TestSchemaOfCollections(t *testing.T) {
	r := NewReflector()

	slice := r.SchemaOf(reflect.TypeOf([]string{}))
	if slice.Type != "array" {
		t.Errorf("slice type = %q, want array", slice.Type)
	}
	if slice.Items == nil || slice.Items.Type != "string" {
		t.Errorf("slice items = %+v, want string", slice.Items)
	}

	// A byte slice is encoded as a base64 string.
	bytes := r.SchemaOf(reflect.TypeOf([]byte{}))
	if bytes.Type != "string" {
		t.Errorf("bytes type = %q, want string", bytes.Type)
	}
	if bytes.Format != "binary" {
		t.Errorf("bytes format = %q, want binary", bytes.Format)
	}

	array := r.SchemaOf(reflect.TypeOf([3]int{}))
	if array.Type != "array" || array.Items == nil || array.Items.Type != "integer" {
		t.Errorf("array = %+v, want array of integer", array)
	}

	m := r.SchemaOf(reflect.TypeOf(map[string]int{}))
	if m.Type != "object" {
		t.Errorf("map type = %q, want object", m.Type)
	}
	if m.AdditionalProperties == nil || m.AdditionalProperties.Type != "integer" {
		t.Errorf("map values = %+v, want integer", m.AdditionalProperties)
	}
}

func TestSchemaOfInterface(t *testing.T) {
	r := NewReflector()
	schema := r.SchemaOf(reflect.TypeOf((*any)(nil)).Elem())
	if schema.Type != "" {
		t.Errorf("type = %q, want empty for an interface", schema.Type)
	}
}

// ---------- struct fields ----------

func TestStructFieldNaming(t *testing.T) {
	r := NewReflector()
	r.SchemaOf(reflect.TypeOf(Ignored{}))
	schema := r.Schemas()["Ignored"]

	if _, ok := schema.Properties["exported"]; !ok {
		t.Error("exported field is missing")
	}
	if _, ok := schema.Properties["-"]; ok {
		t.Error(`the "-" field must be skipped`)
	}
	if _, ok := schema.Properties["Skipped"]; ok {
		t.Error(`the "-" field must be skipped even when named`)
	}
	if _, ok := schema.Properties["unexported"]; ok {
		t.Error("unexported field must be skipped")
	}
	if _, ok := schema.Properties["NoTag"]; !ok {
		t.Error("a field without a json tag must use its Go name")
	}
}

func TestEmbeddedStructIsInlined(t *testing.T) {
	r := NewReflector()
	r.SchemaOf(reflect.TypeOf(Other{}))
	schema := r.Schemas()["Other"]

	if _, ok := schema.Properties["shared"]; !ok {
		t.Error("the embedded field must be inlined")
	}
	if _, ok := schema.Properties["own"]; !ok {
		t.Error("the own field is missing")
	}
}

func TestRequiredFields(t *testing.T) {
	r := NewReflector()
	r.SchemaOf(reflect.TypeOf(Optional{}))
	schema := r.Schemas()["Optional"]

	required := map[string]bool{}
	for _, name := range schema.Required {
		required[name] = true
	}

	if !required["required"] {
		t.Error("a plain field must be required")
	}
	if required["pointer"] {
		t.Error("a pointer field must be optional")
	}
	if required["omit"] {
		t.Error("a field with omitempty must be optional")
	}
	if required["both"] {
		t.Error("a pointer field with omitempty must be optional")
	}
}

func TestRecursiveTypeTerminates(t *testing.T) {
	r := NewReflector()
	schema := r.SchemaOf(reflect.TypeOf(Recursive{}))

	if schema.Ref != componentRef("Recursive") {
		t.Fatalf("ref = %q, want %q", schema.Ref, componentRef("Recursive"))
	}

	component := r.Schemas()["Recursive"]
	if component == nil {
		t.Fatal("the component schema is missing")
	}

	children := component.Properties["children"]
	if children == nil {
		t.Fatal("children property is missing")
	}
	if children.Type != "array" {
		t.Fatalf("children type = %q, want array", children.Type)
	}
	if children.Items == nil {
		t.Fatal("children items are missing")
	}
	if children.Items.Ref != componentRef("Recursive") {
		t.Errorf("children ref = %q, want %q", children.Items.Ref, componentRef("Recursive"))
	}
}

func TestSameTypeReusedAsReference(t *testing.T) {
	r := NewReflector()
	r.SchemaOf(reflect.TypeOf(Simple{}))
	second := r.SchemaOf(reflect.TypeOf(Simple{}))

	if second.Ref != componentRef("Simple") {
		t.Errorf("ref = %q, want %q", second.Ref, componentRef("Simple"))
	}
	if len(r.Schemas()) != 1 {
		t.Errorf("schemas = %d, want 1", len(r.Schemas()))
	}
}

// ---------- struct tags ----------

func TestStructTagOverrides(t *testing.T) {
	r := NewReflector()
	r.SchemaOf(reflect.TypeOf(Tagged{}))
	schema := r.Schemas()["Tagged"]
	props := schema.Properties

	if got := props["price"].Description; got != "Price in cents" {
		t.Errorf("price description = %q, want Price in cents", got)
	}
	if props["price"].Minimum == nil || *props["price"].Minimum != 0 {
		t.Errorf("price minimum = %v, want 0", props["price"].Minimum)
	}

	if got := props["title"].Title; got != "A title" {
		t.Errorf("title title = %q, want A title", got)
	}

	if got := props["tags"].Pattern; got != "^[a-z]+$" {
		t.Errorf("tags pattern = %q, want ^[a-z]+$", got)
	}

	if got := props["kind"].Enum; len(got) != 2 {
		t.Errorf("kind enum = %v, want two values", got)
	}

	if got := props["note"].MaxLength; got == nil || *got != 500 {
		t.Errorf("note maxLength = %v, want 500", got)
	}

	if got := props["secret"].Maximum; got == nil || *got != 9 {
		t.Errorf("secret maximum = %v, want 9", got)
	}
	if got := props["secret"].ExclusiveMinimum; got == nil || *got != 0 {
		t.Errorf("secret exclusiveMinimum = %v, want 0", got)
	}
	if got := props["secret"].ExclusiveMaximum; got == nil || *got != 10 {
		t.Errorf("secret exclusiveMaximum = %v, want 10", got)
	}

	if got := props["length"].MinLength; got == nil || *got != 1 {
		t.Errorf("length minLength = %v, want 1", got)
	}

	if got := props["items"].MinItems; got == nil || *got != 1 {
		t.Errorf("items minItems = %v, want 1", got)
	}
	if !props["items"].UniqueItems {
		t.Error("items uniqueItems is not set")
	}

	if got := props["with_def"].Default; got != "fallback" {
		t.Errorf("with_def default = %q, want fallback", got)
	}
	if got := props["with_def"].Example; got != "sample" {
		t.Errorf("with_def example = %q, want sample", got)
	}
}

func TestStructTagInvalidValuesAreIgnored(t *testing.T) {
	r := NewReflector()
	r.SchemaOf(reflect.TypeOf(Tagged{}))
	schema := r.Schemas()["Tagged"]

	// An unparsable number must not set the constraint.
	if got := schema.Properties["bad_number"].Minimum; got != nil {
		t.Errorf("bad_number minimum = %v, want nil", got)
	}
}

func TestStructTagEmptyIsIgnored(t *testing.T) {
	r := NewReflector()
	r.SchemaOf(reflect.TypeOf(Tagged{}))
	schema := r.Schemas()["Tagged"]
	if got := schema.Properties["empty_tag"].Description; got != "" {
		t.Errorf("empty_tag description = %q, want empty", got)
	}
}

// ---------- helpers ----------

func TestParseJSONTag(t *testing.T) {
	name, opts := parseJSONTag("id,omitempty,string")
	if name != "id" {
		t.Errorf("name = %q, want id", name)
	}
	if !opts["omitempty"] {
		t.Error("the omitempty option is missing")
	}
	if !opts["string"] {
		t.Error("the string option is missing")
	}

	name, opts = parseJSONTag("")
	if name != "" {
		t.Errorf("name = %q, want empty", name)
	}
	if len(opts) != 0 {
		t.Errorf("options = %v, want none", opts)
	}
}

func TestUniqueNameDisambiguates(t *testing.T) {
	r := NewReflector()

	// Register a name that is already taken.
	r.schemas["Simple"] = &Schema{}

	name := r.uniqueName(reflect.TypeOf(Simple{}))
	if name == "Simple" {
		t.Error("the name must differ from the one already taken")
	}
	if name == "" {
		t.Error("the name must not be empty")
	}
}

func TestComponentRef(t *testing.T) {
	got := componentRef("User")
	want := "#/components/schemas/User"
	if got != want {
		t.Errorf("componentRef = %q, want %q", got, want)
	}
}

// ---------- edge cases ----------

// Anonymous embeds a struct without a field name.
type Anonymous struct {
	Embedded
	Own string `json:"own"`
}

// PtrEmbedded embeds a pointer to a struct.
type PtrEmbedded struct {
	*Embedded
	Own string `json:"own"`
}

// Deprecated is marked as deprecated in the tag.
type Deprecated struct {
	Old string `json:"old" jsonschema:"deprecated"`
	Req string `json:"req" jsonschema:"required"`
}

// Formatted has an explicit format.
type Formatted struct {
	Code string `json:"code" jsonschema:"format=uuid"`
}

// Colliding has the same name as another type.
type Colliding struct {
	Value string `json:"value"`
}

func TestAnonymousEmbeddedStruct(t *testing.T) {
	r := NewReflector()
	r.SchemaOf(reflect.TypeOf(Anonymous{}))
	schema := r.Schemas()["Anonymous"]

	if _, ok := schema.Properties["shared"]; !ok {
		t.Errorf("properties = %v, want an inlined shared field", keysOf(schema.Properties))
	}
	if _, ok := schema.Properties["own"]; !ok {
		t.Errorf("properties = %v, want an own field", keysOf(schema.Properties))
	}
}

func TestPointerEmbeddedStructIsInlined(t *testing.T) {
	r := NewReflector()
	r.SchemaOf(reflect.TypeOf(PtrEmbedded{}))
	schema := r.Schemas()["PtrEmbedded"]

	if _, ok := schema.Properties["shared"]; !ok {
		t.Errorf("properties = %v, want an inlined shared field", keysOf(schema.Properties))
	}
}

func TestUniqueNameFallsBackToAnonymous(t *testing.T) {
	r := NewReflector()
	name := r.uniqueName(reflect.TypeOf(struct{}{}))
	if name != "Anonymous" {
		t.Errorf("uniqueName = %q, want Anonymous", name)
	}
}

func TestUniqueNameDisambiguatesRepeatedCollisions(t *testing.T) {
	r := NewReflector()
	r.schemas["Colliding"] = &Schema{}
	r.schemas["spec_Colliding"] = &Schema{}

	name := r.uniqueName(reflect.TypeOf(Colliding{}))
	if name == "Colliding" || name == "spec_Colliding" {
		t.Errorf("uniqueName = %q, want a name that is still free", name)
	}
	if !strings.Contains(name, "Colliding") {
		t.Errorf("uniqueName = %q, want the base name to be kept", name)
	}
}

func TestDeprecatedTag(t *testing.T) {
	r := NewReflector()
	r.SchemaOf(reflect.TypeOf(Deprecated{}))
	schema := r.Schemas()["Deprecated"]

	if got := schema.Properties["old"].Description; got != "(deprecated)" {
		t.Errorf("description = %q, want (deprecated)", got)
	}
}

func TestFormatTag(t *testing.T) {
	r := NewReflector()
	r.SchemaOf(reflect.TypeOf(Formatted{}))
	schema := r.Schemas()["Formatted"]

	if got := schema.Properties["code"].Format; got != "uuid" {
		t.Errorf("format = %q, want uuid", got)
	}
}

func TestIsRequired(t *testing.T) {
	value := ""
	pointer := &value

	cases := []struct {
		name  string
		field reflect.StructField
		want  bool
	}{
		{"plain", reflect.StructField{Type: reflect.TypeOf(value)}, true},
		{"pointer", reflect.StructField{Type: reflect.TypeOf(pointer)}, false},
		{
			"omitempty",
			reflect.StructField{
				Type: reflect.TypeOf(value),
				Tag:  `json:"x,omitempty"`,
			},
			false,
		},
		{
			"jsonschema omitempty",
			reflect.StructField{
				Type: reflect.TypeOf(value),
				Tag:  `json:"x" jsonschema:"omitempty"`,
			},
			false,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, opts := parseJSONTag(c.field.Tag.Get("json"))
			if got := isRequired(c.field, opts); got != c.want {
				t.Errorf("isRequired = %v, want %v", got, c.want)
			}
		})
	}
}

func TestSchemaOfUnsupportedKind(t *testing.T) {
	r := NewReflector()
	schema := r.SchemaOf(reflect.TypeOf(make(chan int)))
	if schema == nil {
		t.Fatal("the schema is nil")
	}
}

// keysOf returns the map keys as a slice.
func keysOf[V any](m map[string]V) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
