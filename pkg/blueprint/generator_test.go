package blueprint

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadBlueprint_Simple(t *testing.T) {
	bp, err := LoadBlueprint("../../testdata/simple/plutus.json")
	if err != nil {
		t.Fatalf("failed to load blueprint: %v", err)
	}

	// Check preamble
	if bp.Preamble.Title != "blueprint/test" {
		t.Errorf("expected title 'blueprint/test', got %q", bp.Preamble.Title)
	}
	if bp.Preamble.PlutusVersion != "v3" {
		t.Errorf("expected plutusVersion 'v3', got %q", bp.Preamble.PlutusVersion)
	}

	// Check validators count
	if len(bp.Validators) != 6 {
		t.Errorf("expected 6 validators, got %d", len(bp.Validators))
	}

	// Check definitions count
	if len(bp.Definitions) != 4 {
		t.Errorf("expected 4 definitions, got %d", len(bp.Definitions))
	}
}

func TestLoadBlueprint_Complex(t *testing.T) {
	bp, err := LoadBlueprint("../../testdata/complex/plutus.json")
	if err != nil {
		t.Fatalf("failed to load blueprint: %v", err)
	}

	// Check preamble
	if bp.Preamble.Title != "treasury/funds" {
		t.Errorf("expected title 'treasury/funds', got %q", bp.Preamble.Title)
	}

	// Check validators count
	if len(bp.Validators) != 3 {
		t.Errorf("expected 3 validators, got %d", len(bp.Validators))
	}

	// Check we have enum types
	multisig, ok := bp.Definitions["multisig/MultisigScript"]
	if !ok {
		t.Error("expected to find MultisigScript definition")
	}
	if multisig == nil || !multisig.IsEnum() {
		t.Error("MultisigScript should be an enum type")
	}
	if multisig != nil && len(multisig.AnyOf) != 7 {
		t.Errorf("MultisigScript should have 7 variants, got %d", len(multisig.AnyOf))
	}
}

func TestLoadBlueprint_Tuple(t *testing.T) {
	bp, err := LoadBlueprint("../../testdata/tuple/plutus.json")
	if err != nil {
		t.Fatalf("failed to load blueprint with tuple types: %v", err)
	}

	// Check preamble
	if bp.Preamble.Title != "tuple/test" {
		t.Errorf("expected title 'tuple/test', got %q", bp.Preamble.Title)
	}

	// Check that tuple type with array items is loaded correctly
	tuple2, ok := bp.Definitions["Tuple$Int_ByteArray"]
	if !ok {
		t.Fatal("expected to find Tuple$Int_ByteArray definition")
	}
	if !tuple2.IsList() {
		t.Error("Tuple should have dataType list")
	}
	if len(tuple2.Items) != 2 {
		t.Errorf("Tuple$Int_ByteArray should have 2 items, got %d", len(tuple2.Items))
	}
	if !tuple2.Items.IsTuple() {
		t.Error("Tuple$Int_ByteArray.Items should be detected as tuple")
	}

	// Check 3-element tuple
	tuple3, ok := bp.Definitions["Tuple$Int_Int_ByteArray"]
	if !ok {
		t.Fatal("expected to find Tuple$Int_Int_ByteArray definition")
	}
	if len(tuple3.Items) != 3 {
		t.Errorf("Tuple$Int_Int_ByteArray should have 3 items, got %d", len(tuple3.Items))
	}

	// Check regular list (single item) still works
	listInt, ok := bp.Definitions["List$Int"]
	if !ok {
		t.Fatal("expected to find List$Int definition")
	}
	if len(listInt.Items) != 1 {
		t.Errorf("List$Int should have 1 item, got %d", len(listInt.Items))
	}
	if listInt.Items.IsTuple() {
		t.Error("List$Int.Items should NOT be detected as tuple")
	}
	if listInt.Items.Single() == nil {
		t.Error("List$Int.Items.Single() should return the single item")
	}
}

func TestSchema_RefName(t *testing.T) {
	tests := []struct {
		ref      string
		expected string
	}{
		{"#/definitions/Int", "Int"},
		{"#/definitions/ByteArray", "ByteArray"},
		{"#/definitions/types~1Payout", "types/Payout"},
		{"#/definitions/cardano~1assets~1PolicyId", "cardano/assets/PolicyId"},
	}

	for _, tc := range tests {
		s := &Schema{Ref: tc.ref}
		got := s.RefName()
		if got != tc.expected {
			t.Errorf("RefName(%q) = %q, want %q", tc.ref, got, tc.expected)
		}
	}
}

func TestSchema_TypeDetection(t *testing.T) {
	// Test integer
	intSchema := &Schema{DataType: "integer"}
	if !intSchema.IsInteger() {
		t.Error("should detect integer type")
	}

	// Test bytes
	bytesSchema := &Schema{DataType: "bytes"}
	if !bytesSchema.IsBytes() {
		t.Error("should detect bytes type")
	}

	// Test list
	listSchema := &Schema{DataType: "list", Items: SchemaItems{&Schema{DataType: "integer"}}}
	if !listSchema.IsList() {
		t.Error("should detect list type")
	}

	// Test map
	mapSchema := &Schema{DataType: "map"}
	if !mapSchema.IsMap() {
		t.Error("should detect map type")
	}

	// Test constructor
	idx := 0
	constructorSchema := &Schema{DataType: "constructor", Index: &idx}
	if !constructorSchema.IsConstructor() {
		t.Error("should detect constructor type")
	}

	// Test enum
	enumSchema := &Schema{
		AnyOf: []Schema{
			{Title: "VariantA", DataType: "constructor", Index: &idx},
			{Title: "VariantB", DataType: "constructor"},
		},
	}
	if !enumSchema.IsEnum() {
		t.Error("should detect enum type")
	}

	// Test Unit/Void
	unitSchema := &Schema{
		AnyOf: []Schema{
			{DataType: "constructor", Index: &idx, Fields: []Schema{}},
		},
	}
	if !unitSchema.IsUnit() {
		t.Error("should detect unit type")
	}

	// Test Boolean
	idx1 := 1
	boolSchema := &Schema{
		AnyOf: []Schema{
			{Title: "False", DataType: "constructor", Index: &idx},
			{Title: "True", DataType: "constructor", Index: &idx1},
		},
	}
	if !boolSchema.IsBoolean() {
		t.Error("should detect boolean type")
	}

	// Test Option
	optionSchema := &Schema{
		AnyOf: []Schema{
			{Title: "Some", DataType: "constructor", Index: &idx, Fields: []Schema{{DataType: "integer"}}},
			{Title: "None", DataType: "constructor"},
		},
	}
	if !optionSchema.IsOption() {
		t.Error("should detect option type")
	}
}

func TestGenerateSimple(t *testing.T) {
	bp, err := LoadBlueprint("../../testdata/simple/plutus.json")
	if err != nil {
		t.Fatalf("failed to load blueprint: %v", err)
	}

	gen := NewGenerator(bp, GeneratorOptions{PackageName: "contracts"})
	code, err := gen.Generate()
	if err != nil {
		t.Fatalf("failed to generate code: %v", err)
	}

	// Check that the code contains expected elements
	checks := []string{
		"package contracts",
		"import (",
		`"math/big"`,
		`"github.com/fxamacker/cbor/v2"`,
		"type PlutusData struct",
	}

	for _, check := range checks {
		if !strings.Contains(code, check) {
			t.Errorf("generated code missing expected element: %q", check)
		}
	}
}

func TestGenerateComplex(t *testing.T) {
	bp, err := LoadBlueprint("../../testdata/complex/plutus.json")
	if err != nil {
		t.Fatalf("failed to load blueprint: %v", err)
	}

	gen := NewGenerator(bp, GeneratorOptions{PackageName: "treasury"})
	code, err := gen.Generate()
	if err != nil {
		t.Fatalf("failed to generate code: %v", err)
	}

	// Check for enum types (interface + variants)
	// Type names now include full path to avoid collisions
	checks := []string{
		"package treasury",
		"type MultisigMultisigScript interface",
		"isMultisigMultisigScript()",
		"ToPlutusData() (PlutusData, error)",
		"type MultisigMultisigScriptSignature struct",
		"type MultisigMultisigScriptAllOf struct",
		"type MultisigMultisigScriptAnyOf struct",
		"type MultisigMultisigScriptAtLeast struct",
		"type MultisigMultisigScriptBefore struct",
		"type MultisigMultisigScriptAfter struct",
		"type MultisigMultisigScriptScript struct",
		"func (MultisigMultisigScriptSignature) isMultisigMultisigScript()",
		"func (v MultisigMultisigScriptSignature) ToPlutusData()",
		"func (v *MultisigMultisigScriptSignature) FromPlutusData(",
		"MultisigMultisigScriptFromPlutusData(pd PlutusData)",
		"type TypesPayoutStatus interface",
		"type TypesPayoutStatusActive struct",
		"type TypesPayoutStatusPaused struct",
		"type TypesPayout struct",
		"type TypesTreasuryConfiguration struct",
		"type TypesTreasuryPermissions struct",
	}

	for _, check := range checks {
		if !strings.Contains(code, check) {
			t.Errorf("generated code missing expected element: %q", check)
		}
	}
}

func TestGenerateTuple(t *testing.T) {
	bp, err := LoadBlueprint("../../testdata/tuple/plutus.json")
	if err != nil {
		t.Fatalf("failed to load blueprint: %v", err)
	}

	gen := NewGenerator(bp, GeneratorOptions{PackageName: "tuples"})
	code, err := gen.Generate()
	if err != nil {
		t.Fatalf("failed to generate code: %v", err)
	}

	// Verify the generated code contains expected elements
	checks := []string{
		"package tuples",
	}

	for _, check := range checks {
		if !strings.Contains(code, check) {
			t.Errorf("generated code missing expected element: %q", check)
		}
	}
}

func TestGeneratedCodeCompiles(t *testing.T) {
	// Skip if not running in an environment with Go compiler
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("go compiler not found, skipping compilation test")
	}

	// Create a temporary directory for the test
	tmpDir, err := os.MkdirTemp("", "aiken2go_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Generate code for complex blueprint (has actual types)
	bp, err := LoadBlueprint("../../testdata/complex/plutus.json")
	if err != nil {
		t.Fatalf("failed to load blueprint: %v", err)
	}

	gen := NewGenerator(bp, GeneratorOptions{PackageName: "contracts"})
	code, err := gen.Generate()
	if err != nil {
		t.Fatalf("failed to generate code: %v", err)
	}

	// Write the generated code
	outFile := filepath.Join(tmpDir, "contracts.go")
	if err := os.WriteFile(outFile, []byte(code), 0644); err != nil {
		t.Fatalf("failed to write generated code: %v", err)
	}

	// Create go.mod with cbor dependency
	goMod := `module testmod

go 1.21

require github.com/fxamacker/cbor/v2 v2.8.0

require github.com/x448/float16 v0.8.4 // indirect
`
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0644); err != nil {
		t.Fatalf("failed to write go.mod: %v", err)
	}

	// Run go mod tidy
	tidyCmd := exec.Command("go", "mod", "tidy")
	tidyCmd.Dir = tmpDir
	if output, err := tidyCmd.CombinedOutput(); err != nil {
		t.Fatalf("go mod tidy failed: %v\nOutput: %s", err, output)
	}

	// Try to build
	cmd := exec.Command("go", "build", ".")
	cmd.Dir = tmpDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Errorf("generated code failed to compile: %v\nOutput: %s\n\nGenerated code:\n%s", err, output, code)
	}
}
