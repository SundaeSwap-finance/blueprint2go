package blueprint

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// TestAllTypesRoundTrip tests that all Aiken types can be correctly
// marshaled to CBOR PlutusData and unmarshaled back.
// This validates the end-to-end flow with a real Aiken-generated blueprint.
func TestAllTypesRoundTrip(t *testing.T) {
	// Create temp directory for generated code
	tmpDir, err := os.MkdirTemp("", "all_types_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create subdirectory for generated types
	typesDir := filepath.Join(tmpDir, "types")
	if err := os.MkdirAll(typesDir, 0755); err != nil {
		t.Fatalf("failed to create types dir: %v", err)
	}

	// Generate Go code from the all_types blueprint
	bp, err := LoadBlueprint("../../testdata/all_types/plutus.json")
	if err != nil {
		t.Fatalf("failed to load blueprint: %v", err)
	}

	gen := NewGenerator(bp, GeneratorOptions{PackageName: "types"})
	code, err := gen.Generate()
	if err != nil {
		t.Fatalf("failed to generate code: %v", err)
	}

	// Write the generated types to types/types.go
	typesFile := filepath.Join(typesDir, "types.go")
	if err := os.WriteFile(typesFile, []byte(code), 0644); err != nil {
		t.Fatalf("failed to write types file: %v", err)
	}

	// Write the test program
	testProgram := `package main

import (
	"fmt"
	"math/big"
	"os"

	"testpkg/types"
)

func testRoundTrip[T interface {
	ToPlutusData() (types.PlutusData, error)
}](name string, original T, decode func(types.PlutusData) (T, error)) error {
	// Serialize to PlutusData
	pd, err := original.ToPlutusData()
	if err != nil {
		return fmt.Errorf("%s ToPlutusData: %v", name, err)
	}

	// Serialize to CBOR
	cborBytes, err := pd.MarshalCBOR()
	if err != nil {
		return fmt.Errorf("%s MarshalCBOR: %v", name, err)
	}

	// Deserialize from CBOR
	var decodedPd types.PlutusData
	if err := decodedPd.UnmarshalCBOR(cborBytes); err != nil {
		return fmt.Errorf("%s UnmarshalCBOR: %v", name, err)
	}

	// Deserialize back to type
	_, err = decode(decodedPd)
	if err != nil {
		return fmt.Errorf("%s FromPlutusData: %v", name, err)
	}

	hex, _ := pd.ToHex()
	fmt.Printf("✓ %s: %s\n", name, hex)
	return nil
}

func main() {
	var failed bool

	// Test SimpleString (now StringValidatorSimpleString with full path)
	simpleString := types.StringValidatorSimpleString{Message: []byte("Hello")}
	if err := testRoundTrip("SimpleString", simpleString, func(pd types.PlutusData) (types.StringValidatorSimpleString, error) {
		var v types.StringValidatorSimpleString
		return v, v.FromPlutusData(pd)
	}); err != nil {
		fmt.Fprintln(os.Stderr, err)
		failed = true
	}

	// Test SimpleInt
	simpleInt := types.StringValidatorSimpleInt{Value: big.NewInt(42)}
	if err := testRoundTrip("SimpleInt", simpleInt, func(pd types.PlutusData) (types.StringValidatorSimpleInt, error) {
		var v types.StringValidatorSimpleInt
		return v, v.FromPlutusData(pd)
	}); err != nil {
		fmt.Fprintln(os.Stderr, err)
		failed = true
	}

	// Test MultipleFields with Bool
	multipleFields := types.StringValidatorMultipleFields{
		Name:   []byte("Alice"),
		Age:    big.NewInt(30),
		Active: true,
	}
	if err := testRoundTrip("MultipleFields", multipleFields, func(pd types.PlutusData) (types.StringValidatorMultipleFields, error) {
		var v types.StringValidatorMultipleFields
		return v, v.FromPlutusData(pd)
	}); err != nil {
		fmt.Fprintln(os.Stderr, err)
		failed = true
	}

	// Test WithList (list of ints)
	withList := types.StringValidatorWithList{
		Items: []*big.Int{big.NewInt(1), big.NewInt(2), big.NewInt(3)},
	}
	if err := testRoundTrip("WithList", withList, func(pd types.PlutusData) (types.StringValidatorWithList, error) {
		var v types.StringValidatorWithList
		return v, v.FromPlutusData(pd)
	}); err != nil {
		fmt.Fprintln(os.Stderr, err)
		failed = true
	}

	// Test WithStringList (list of byte arrays)
	withStringList := types.StringValidatorWithStringList{
		Names: [][]byte{[]byte("Alice"), []byte("Bob")},
	}
	if err := testRoundTrip("WithStringList", withStringList, func(pd types.PlutusData) (types.StringValidatorWithStringList, error) {
		var v types.StringValidatorWithStringList
		return v, v.FromPlutusData(pd)
	}); err != nil {
		fmt.Fprintln(os.Stderr, err)
		failed = true
	}

	// Test WithOption (with Some value)
	withOptionSome := types.StringValidatorWithOption{
		Label:      []byte("label"),
		MaybeValue: types.OptionInt{Value: big.NewInt(100), IsSet: true},
	}
	if err := testRoundTrip("WithOption(Some)", withOptionSome, func(pd types.PlutusData) (types.StringValidatorWithOption, error) {
		var v types.StringValidatorWithOption
		return v, v.FromPlutusData(pd)
	}); err != nil {
		fmt.Fprintln(os.Stderr, err)
		failed = true
	}

	// Test WithOption (with None)
	withOptionNone := types.StringValidatorWithOption{
		Label:      []byte("label"),
		MaybeValue: types.OptionInt{IsSet: false},
	}
	if err := testRoundTrip("WithOption(None)", withOptionNone, func(pd types.PlutusData) (types.StringValidatorWithOption, error) {
		var v types.StringValidatorWithOption
		return v, v.FromPlutusData(pd)
	}); err != nil {
		fmt.Fprintln(os.Stderr, err)
		failed = true
	}

	// Test Status enum - Active variant
	statusActive := types.StringValidatorStatusActive{}
	if err := testRoundTrip("StatusActive", statusActive, func(pd types.PlutusData) (types.StringValidatorStatusActive, error) {
		var v types.StringValidatorStatusActive
		return v, v.FromPlutusData(pd)
	}); err != nil {
		fmt.Fprintln(os.Stderr, err)
		failed = true
	}

	// Test Status enum - Inactive variant
	statusInactive := types.StringValidatorStatusInactive{}
	if err := testRoundTrip("StatusInactive", statusInactive, func(pd types.PlutusData) (types.StringValidatorStatusInactive, error) {
		var v types.StringValidatorStatusInactive
		return v, v.FromPlutusData(pd)
	}); err != nil {
		fmt.Fprintln(os.Stderr, err)
		failed = true
	}

	// Test Status enum - Pending variant with data
	statusPending := types.StringValidatorStatusPending{Reason: []byte("test")}
	if err := testRoundTrip("StatusPending", statusPending, func(pd types.PlutusData) (types.StringValidatorStatusPending, error) {
		var v types.StringValidatorStatusPending
		return v, v.FromPlutusData(pd)
	}); err != nil {
		fmt.Fprintln(os.Stderr, err)
		failed = true
	}

	// Test Nested type
	nested := types.StringValidatorNested{
		Inner: types.StringValidatorSimpleString{Message: []byte("inner")},
		Count: big.NewInt(5),
	}
	if err := testRoundTrip("Nested", nested, func(pd types.PlutusData) (types.StringValidatorNested, error) {
		var v types.StringValidatorNested
		return v, v.FromPlutusData(pd)
	}); err != nil {
		fmt.Fprintln(os.Stderr, err)
		failed = true
	}

	// Test WithEnum (struct containing enum)
	withEnum := types.StringValidatorWithEnum{
		Id:     []byte("myid"),
		Status: types.StringValidatorStatusActive{},
	}
	if err := testRoundTrip("WithEnum", withEnum, func(pd types.PlutusData) (types.StringValidatorWithEnum, error) {
		var v types.StringValidatorWithEnum
		return v, v.FromPlutusData(pd)
	}); err != nil {
		fmt.Fprintln(os.Stderr, err)
		failed = true
	}

	// Test WithRecordList (list of structs)
	withRecordList := types.StringValidatorWithRecordList{
		Entries: []types.StringValidatorSimpleString{
			{Message: []byte("one")},
			{Message: []byte("two")},
		},
	}
	if err := testRoundTrip("WithRecordList", withRecordList, func(pd types.PlutusData) (types.StringValidatorWithRecordList, error) {
		var v types.StringValidatorWithRecordList
		return v, v.FromPlutusData(pd)
	}); err != nil {
		fmt.Fprintln(os.Stderr, err)
		failed = true
	}

	// Test WithOptionalNested
	withOptionalNested := types.StringValidatorWithOptionalNested{
		Data: types.OptionStringValidatorSimpleString{Value: types.StringValidatorSimpleString{Message: []byte("opt")}, IsSet: true},
	}
	if err := testRoundTrip("WithOptionalNested", withOptionalNested, func(pd types.PlutusData) (types.StringValidatorWithOptionalNested, error) {
		var v types.StringValidatorWithOptionalNested
		return v, v.FromPlutusData(pd)
	}); err != nil {
		fmt.Fprintln(os.Stderr, err)
		failed = true
	}

	if failed {
		os.Exit(1)
	}
	fmt.Println("\n✓ All round-trip tests passed!")
}
`

	// Write test program to main.go
	mainFile := filepath.Join(tmpDir, "main.go")
	if err := os.WriteFile(mainFile, []byte(testProgram), 0644); err != nil {
		t.Fatalf("failed to write main file: %v", err)
	}

	// Initialize go module
	goModContent := `module testpkg

go 1.21

require github.com/fxamacker/cbor/v2 v2.8.0

require github.com/x448/float16 v0.8.4 // indirect
`
	goModFile := filepath.Join(tmpDir, "go.mod")
	if err := os.WriteFile(goModFile, []byte(goModContent), 0644); err != nil {
		t.Fatalf("failed to write go.mod: %v", err)
	}

	// Run go mod tidy
	cmd := exec.Command("go", "mod", "tidy")
	cmd.Dir = tmpDir
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("go mod tidy failed: %v\n%s", err, output)
	}

	// Run the test program
	cmd = exec.Command("go", "run", "main.go")
	cmd.Dir = tmpDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("test program failed: %v\n%s", err, output)
	}

	t.Logf("Test output:\n%s", output)
}
