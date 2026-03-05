package blueprint

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestJSONMarshal_BoolTemplate verifies that the bool_type template produces MarshalJSON.
func TestJSONMarshal_BoolTemplate(t *testing.T) {
	var buf strings.Builder
	err := templates.ExecuteTemplate(&buf, "bool_type.go.tmpl", map[string]string{"Name": "MyBool"})
	if err != nil {
		t.Fatalf("failed to execute template: %v", err)
	}
	code := buf.String()

	if !strings.Contains(code, "func (v MyBool) MarshalJSON() ([]byte, error)") {
		t.Error("Expected bool template to produce MarshalJSON method")
	}
	if !strings.Contains(code, "json.Marshal(bool(v))") {
		t.Error("Expected bool MarshalJSON to use json.Marshal(bool(v))")
	}
}

// TestJSONMarshal_UnitTemplate verifies that the unit_type template produces MarshalJSON.
func TestJSONMarshal_UnitTemplate(t *testing.T) {
	var buf strings.Builder
	err := templates.ExecuteTemplate(&buf, "unit_type.go.tmpl", map[string]string{"Name": "MyUnit"})
	if err != nil {
		t.Fatalf("failed to execute template: %v", err)
	}
	code := buf.String()

	if !strings.Contains(code, "func (v MyUnit) MarshalJSON() ([]byte, error)") {
		t.Error("Expected unit template to produce MarshalJSON method")
	}
	if !strings.Contains(code, `"unit"`) {
		t.Error("Expected unit MarshalJSON to return \"unit\"")
	}
}

// TestJSONMarshal_OptionTypes verifies that Option types get MarshalJSON
// and that the generated code compiles and produces correct JSON.
func TestJSONMarshal_OptionTypes(t *testing.T) {
	bp, err := LoadBlueprint("../../testdata/all_types/plutus.json")
	if err != nil {
		t.Fatalf("failed to load blueprint: %v", err)
	}

	gen := NewGenerator(bp, GeneratorOptions{PackageName: "types"})
	code, err := gen.Generate()
	if err != nil {
		t.Fatalf("failed to generate code: %v", err)
	}

	// Option types should have MarshalJSON
	if !strings.Contains(code, "func (v OptionInt) MarshalJSON() ([]byte, error)") {
		t.Error("Expected OptionInt to have MarshalJSON method")
	}

	// Verify generated code compiles with a test program
	tmpDir, err := os.MkdirTemp("", "option_json_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	typesDir := filepath.Join(tmpDir, "types")
	if err := os.MkdirAll(typesDir, 0755); err != nil {
		t.Fatalf("failed to create types dir: %v", err)
	}

	if err := os.WriteFile(filepath.Join(typesDir, "types.go"), []byte(code), 0644); err != nil {
		t.Fatalf("failed to write types file: %v", err)
	}

	testProgram := `package main

import (
	"encoding/json"
	"fmt"
	"math/big"
	"os"

	"testpkg/types"
)

func main() {
	// Test Option None -> null
	optNone := types.OptionInt{IsSet: false}
	b, err := json.Marshal(optNone)
	if err != nil {
		fmt.Fprintf(os.Stderr, "OptionInt None marshal error: %v\n", err)
		os.Exit(1)
	}
	if string(b) != "null" {
		fmt.Fprintf(os.Stderr, "OptionInt None: expected null, got %s\n", string(b))
		os.Exit(1)
	}

	// Test Option Some(*big.Int) -> string value
	optSome := types.OptionInt{IsSet: true, Value: big.NewInt(42)}
	b, err = json.Marshal(optSome)
	if err != nil {
		fmt.Fprintf(os.Stderr, "OptionInt Some marshal error: %v\n", err)
		os.Exit(1)
	}
	if string(b) != ` + "`" + `"42"` + "`" + ` {
		fmt.Fprintf(os.Stderr, "OptionInt Some: expected \"42\", got %s\n", string(b))
		os.Exit(1)
	}

	fmt.Println("OK")
}
`

	mainFile := filepath.Join(tmpDir, "main.go")
	if err := os.WriteFile(mainFile, []byte(testProgram), 0644); err != nil {
		t.Fatalf("failed to write main.go: %v", err)
	}

	goMod := "module testpkg\n\ngo 1.21\n\nrequire github.com/fxamacker/cbor/v2 v2.7.0\n\nrequire github.com/x448/float16 v0.8.4 // indirect\n"
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0644); err != nil {
		t.Fatalf("failed to write go.mod: %v", err)
	}

	// Run go mod tidy to resolve dependencies
	tidyCmd := exec.Command("go", "mod", "tidy")
	tidyCmd.Dir = tmpDir
	if out, err := tidyCmd.CombinedOutput(); err != nil {
		t.Fatalf("go mod tidy failed: %v\n%s", err, out)
	}

	cmd := exec.Command("go", "run", "main.go")
	cmd.Dir = tmpDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("test program failed: %v\n%s\nGenerated code:\n%s", err, output, code)
	}
	if !strings.Contains(string(output), "OK") {
		t.Fatalf("unexpected output: %s", output)
	}
}

// TestJSONMarshal_StructTypes verifies that struct types get MarshalJSON
// with proper field conversion (*big.Int -> string, []byte -> hex).
func TestJSONMarshal_StructTypes(t *testing.T) {
	bp, err := LoadBlueprint("../../testdata/all_types/plutus.json")
	if err != nil {
		t.Fatalf("failed to load blueprint: %v", err)
	}

	gen := NewGenerator(bp, GeneratorOptions{PackageName: "types"})
	code, err := gen.Generate()
	if err != nil {
		t.Fatalf("failed to generate code: %v", err)
	}

	// Struct types should have MarshalJSON
	if !strings.Contains(code, "func (v StringValidatorSimpleString) MarshalJSON() ([]byte, error)") {
		t.Error("Expected StringValidatorSimpleString to have MarshalJSON method")
	}

	// Verify generated code compiles and marshals correctly
	tmpDir, err := os.MkdirTemp("", "struct_json_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	typesDir := filepath.Join(tmpDir, "types")
	if err := os.MkdirAll(typesDir, 0755); err != nil {
		t.Fatalf("failed to create types dir: %v", err)
	}

	if err := os.WriteFile(filepath.Join(typesDir, "types.go"), []byte(code), 0644); err != nil {
		t.Fatalf("failed to write types file: %v", err)
	}

	testProgram := `package main

import (
	"encoding/json"
	"fmt"
	"math/big"
	"os"

	"testpkg/types"
)

func main() {
	// Test struct with *big.Int -> string in JSON
	s := types.StringValidatorSimpleInt{Value: big.NewInt(42)}
	b, err := json.Marshal(s)
	if err != nil {
		fmt.Fprintf(os.Stderr, "SimpleInt marshal error: %v\n", err)
		os.Exit(1)
	}
	expected := ` + "`" + `{"value":"42"}` + "`" + `
	if string(b) != expected {
		fmt.Fprintf(os.Stderr, "SimpleInt: expected %s, got %s\n", expected, string(b))
		os.Exit(1)
	}

	// Test struct with []byte -> hex in JSON
	s2 := types.StringValidatorSimpleString{Message: []byte{0xde, 0xad}}
	b, err = json.Marshal(s2)
	if err != nil {
		fmt.Fprintf(os.Stderr, "SimpleString marshal error: %v\n", err)
		os.Exit(1)
	}
	expected2 := ` + "`" + `{"message":"dead"}` + "`" + `
	if string(b) != expected2 {
		fmt.Fprintf(os.Stderr, "SimpleString: expected %s, got %s\n", expected2, string(b))
		os.Exit(1)
	}

	fmt.Println("OK")
}
`

	mainFile := filepath.Join(tmpDir, "main.go")
	if err := os.WriteFile(mainFile, []byte(testProgram), 0644); err != nil {
		t.Fatalf("failed to write main.go: %v", err)
	}

	goMod := "module testpkg\n\ngo 1.21\n\nrequire github.com/fxamacker/cbor/v2 v2.7.0\n\nrequire github.com/x448/float16 v0.8.4 // indirect\n"
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0644); err != nil {
		t.Fatalf("failed to write go.mod: %v", err)
	}

	tidyCmd := exec.Command("go", "mod", "tidy")
	tidyCmd.Dir = tmpDir
	if out, err := tidyCmd.CombinedOutput(); err != nil {
		t.Fatalf("go mod tidy failed: %v\n%s", err, out)
	}

	cmd := exec.Command("go", "run", "main.go")
	cmd.Dir = tmpDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("test program failed: %v\n%s", err, output)
	}
	if !strings.Contains(string(output), "OK") {
		t.Fatalf("unexpected output: %s", output)
	}
}

// TestJSONMarshal_EnumVariants verifies tagged union format for enum variants.
func TestJSONMarshal_EnumVariants(t *testing.T) {
	bp, err := LoadBlueprint("../../testdata/all_types/plutus.json")
	if err != nil {
		t.Fatalf("failed to load blueprint: %v", err)
	}

	gen := NewGenerator(bp, GeneratorOptions{PackageName: "types"})
	code, err := gen.Generate()
	if err != nil {
		t.Fatalf("failed to generate code: %v", err)
	}

	// Empty variants should have MarshalJSON
	if !strings.Contains(code, "func (v StringValidatorStatusActive) MarshalJSON() ([]byte, error)") {
		t.Error("Expected StringValidatorStatusActive (empty variant) to have MarshalJSON method")
	}

	// Verify generated code compiles and produces tagged unions
	tmpDir, err := os.MkdirTemp("", "enum_json_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	typesDir := filepath.Join(tmpDir, "types")
	if err := os.MkdirAll(typesDir, 0755); err != nil {
		t.Fatalf("failed to create types dir: %v", err)
	}

	if err := os.WriteFile(filepath.Join(typesDir, "types.go"), []byte(code), 0644); err != nil {
		t.Fatalf("failed to write types file: %v", err)
	}

	testProgram := `package main

import (
	"encoding/json"
	"fmt"
	"os"

	"testpkg/types"
)

func main() {
	// Test empty variant -> tagged union
	active := types.StringValidatorStatusActive{}
	b, err := json.Marshal(active)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Active marshal error: %v\n", err)
		os.Exit(1)
	}
	expected := ` + "`" + `{"constructor":"Active"}` + "`" + `
	if string(b) != expected {
		fmt.Fprintf(os.Stderr, "Active: expected %s, got %s\n", expected, string(b))
		os.Exit(1)
	}

	fmt.Println("OK")
}
`

	mainFile := filepath.Join(tmpDir, "main.go")
	if err := os.WriteFile(mainFile, []byte(testProgram), 0644); err != nil {
		t.Fatalf("failed to write main.go: %v", err)
	}

	goMod := "module testpkg\n\ngo 1.21\n\nrequire github.com/fxamacker/cbor/v2 v2.7.0\n\nrequire github.com/x448/float16 v0.8.4 // indirect\n"
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0644); err != nil {
		t.Fatalf("failed to write go.mod: %v", err)
	}

	tidyCmd := exec.Command("go", "mod", "tidy")
	tidyCmd.Dir = tmpDir
	if out, err := tidyCmd.CombinedOutput(); err != nil {
		t.Fatalf("go mod tidy failed: %v\n%s", err, out)
	}

	cmd2 := exec.Command("go", "run", "main.go")
	cmd2.Dir = tmpDir
	output2, err := cmd2.CombinedOutput()
	if err != nil {
		t.Fatalf("test program failed: %v\n%s", err, output2)
	}
	if !strings.Contains(string(output2), "OK") {
		t.Fatalf("unexpected output: %s", output2)
	}
}

// TestJSONMarshal_TupleTypes verifies that tuple types get MarshalJSON.
func TestJSONMarshal_TupleTypes(t *testing.T) {
	bp, err := LoadBlueprint("../../testdata/tuple/plutus.json")
	if err != nil {
		t.Fatalf("failed to load blueprint: %v", err)
	}

	gen := NewGenerator(bp, GeneratorOptions{PackageName: "types"})
	code, err := gen.Generate()
	if err != nil {
		t.Fatalf("failed to generate code: %v", err)
	}

	// TupleIntBytearray (from Tuple$Int_ByteArray) should have its own MarshalJSON
	if !strings.Contains(code, "func (v TupleIntBytearray) MarshalJSON() ([]byte, error)") {
		t.Error("Expected TupleIntBytearray to have MarshalJSON method")
	}
}

// TestJSONMarshal_ListTypeAliases verifies that list aliases with primitive inner
// types ([]byte, *big.Int) get MarshalJSON with proper conversion.
// Note: List$ prefixed types are handled inline, so we test with a custom
// blueprint that has a non-prefixed list definition.
// TestJSONMarshal_Integration is an end-to-end test that generates code from
// the all_types blueprint, compiles it, and verifies json.Marshal produces
// readable output for structs, enums, options, and nested types.
func TestJSONMarshal_Integration(t *testing.T) {
	bp, err := LoadBlueprint("../../testdata/all_types/plutus.json")
	if err != nil {
		t.Fatalf("failed to load blueprint: %v", err)
	}

	gen := NewGenerator(bp, GeneratorOptions{PackageName: "types"})
	code, err := gen.Generate()
	if err != nil {
		t.Fatalf("failed to generate code: %v", err)
	}

	tmpDir, err := os.MkdirTemp("", "integration_json_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	typesDir := filepath.Join(tmpDir, "types")
	if err := os.MkdirAll(typesDir, 0755); err != nil {
		t.Fatalf("failed to create types dir: %v", err)
	}

	if err := os.WriteFile(filepath.Join(typesDir, "types.go"), []byte(code), 0644); err != nil {
		t.Fatalf("failed to write types file: %v", err)
	}

	testProgram := `package main

import (
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"strings"

	"testpkg/types"
)

func check(name, expected string, v interface{}) {
	b, err := json.Marshal(v)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: marshal error: %v\n", name, err)
		os.Exit(1)
	}
	if string(b) != expected {
		fmt.Fprintf(os.Stderr, "%s:\n  expected: %s\n  got:      %s\n", name, expected, string(b))
		os.Exit(1)
	}
}

func checkContains(name string, v interface{}, substrings ...string) {
	b, err := json.Marshal(v)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: marshal error: %v\n", name, err)
		os.Exit(1)
	}
	for _, s := range substrings {
		if !strings.Contains(string(b), s) {
			fmt.Fprintf(os.Stderr, "%s: expected to contain %q, got: %s\n", name, s, string(b))
			os.Exit(1)
		}
	}
}

func main() {
	// Struct: *big.Int -> string, []byte -> hex
	check("SimpleInt",
		` + "`" + `{"value":"42"}` + "`" + `,
		types.StringValidatorSimpleInt{Value: big.NewInt(42)})

	check("SimpleString",
		` + "`" + `{"message":"deadbeef"}` + "`" + `,
		types.StringValidatorSimpleString{Message: []byte{0xde, 0xad, 0xbe, 0xef}})

	// Option None -> null
	check("OptionInt None", "null",
		types.OptionInt{IsSet: false})

	// Option Some -> value
	check("OptionInt Some",
		` + "`" + `"99"` + "`" + `,
		types.OptionInt{IsSet: true, Value: big.NewInt(99)})

	// Enum empty variant -> tagged union
	check("Status Active",
		` + "`" + `{"constructor":"Active"}` + "`" + `,
		types.StringValidatorStatusActive{})

	// Struct with multiple field types
	checkContains("MultipleFields",
		types.StringValidatorMultipleFields{
			Name:   []byte("test"),
			Age:    big.NewInt(100),
			Active: true,
		},
		` + "`" + `"name":"74657374"` + "`" + `,
		` + "`" + `"age":"100"` + "`" + `,
		` + "`" + `"active":true` + "`" + `)

	fmt.Println("OK")
}
`

	mainFile := filepath.Join(tmpDir, "main.go")
	if err := os.WriteFile(mainFile, []byte(testProgram), 0644); err != nil {
		t.Fatalf("failed to write main.go: %v", err)
	}

	goMod := "module testpkg\n\ngo 1.21\n\nrequire github.com/fxamacker/cbor/v2 v2.7.0\n\nrequire github.com/x448/float16 v0.8.4 // indirect\n"
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0644); err != nil {
		t.Fatalf("failed to write go.mod: %v", err)
	}

	tidyCmd := exec.Command("go", "mod", "tidy")
	tidyCmd.Dir = tmpDir
	if out, err := tidyCmd.CombinedOutput(); err != nil {
		t.Fatalf("go mod tidy failed: %v\n%s", err, out)
	}

	cmd := exec.Command("go", "run", "main.go")
	cmd.Dir = tmpDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("test program failed: %v\n%s", err, output)
	}
	if !strings.Contains(string(output), "OK") {
		t.Fatalf("unexpected output: %s", output)
	}
}

func TestJSONMarshal_ListTypeAliases(t *testing.T) {
	bp := &Blueprint{
		Preamble: Preamble{Title: "test"},
		Definitions: map[string]*Schema{
			"Int":       {DataType: "integer"},
			"ByteArray": {DataType: "bytes"},
			"custom/Amounts": {
				Title:    "Amounts",
				DataType: "list",
				Items:    SchemaItems{&Schema{Ref: "#/definitions/Int"}},
			},
			"custom/Hashes": {
				Title:    "Hashes",
				DataType: "list",
				Items:    SchemaItems{&Schema{Ref: "#/definitions/ByteArray"}},
			},
		},
	}

	gen := NewGenerator(bp, GeneratorOptions{PackageName: "types"})
	code, err := gen.Generate()
	if err != nil {
		t.Fatalf("failed to generate code: %v", err)
	}

	// CustomAmounts (list of *big.Int) should have MarshalJSON with bigIntSlice
	if !strings.Contains(code, "func (v CustomAmounts) MarshalJSON() ([]byte, error)") {
		t.Errorf("Expected CustomAmounts to have MarshalJSON method")
	}
	if !strings.Contains(code, "bigIntSlice") {
		t.Errorf("Expected CustomAmounts MarshalJSON to use bigIntSlice")
	}

	// CustomHashes (list of []byte) should have MarshalJSON with hexBytesSlice
	if !strings.Contains(code, "func (v CustomHashes) MarshalJSON() ([]byte, error)") {
		t.Errorf("Expected CustomHashes to have MarshalJSON method")
	}
	if !strings.Contains(code, "hexBytesSlice") {
		t.Errorf("Expected CustomHashes MarshalJSON to use hexBytesSlice")
	}
}

