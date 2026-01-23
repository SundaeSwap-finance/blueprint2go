package blueprint

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestMapTypes tests the code generation for map type patterns
func TestMapTypes(t *testing.T) {
	bp, err := LoadBlueprint("../../testdata/map_types/plutus.json")
	if err != nil {
		t.Fatalf("failed to load blueprint: %v", err)
	}

	gen := NewGenerator(bp, GeneratorOptions{PackageName: "maptypes"})
	code, err := gen.Generate()
	if err != nil {
		t.Fatalf("failed to generate code: %v", err)
	}

	// Test 1: Map types should generate proper Go map types
	t.Run("MapTypeGeneration", func(t *testing.T) {
		// SimpleIntMap should have Values as map[*big.Int]*big.Int
		if !strings.Contains(code, "Values map[*big.Int]*big.Int") {
			t.Error("Expected Values field to be 'map[*big.Int]*big.Int' type")
		}
		// StringToIntMap should have Entries as map[string]*big.Int ([]byte keys become string)
		if !strings.Contains(code, "Entries map[string]*big.Int") {
			t.Error("Expected Entries field to be 'map[string]*big.Int' type")
		}
		// StringToStringMap should have Data as map[string][]byte
		if !strings.Contains(code, "Data map[string][]byte") {
			t.Error("Expected Data field to be 'map[string][]byte' type")
		}
	})

	// Test 2: Map serialization should use NewMapPlutusData
	t.Run("MapSerialization", func(t *testing.T) {
		if !strings.Contains(code, "NewMapPlutusData(") {
			t.Error("Expected map serialization to use NewMapPlutusData")
		}
		if !strings.Contains(code, "PlutusDataMapEntry{") {
			t.Error("Expected map serialization to use PlutusDataMapEntry")
		}
	})

	// Test 3: Map deserialization should check for Map field
	t.Run("MapDeserialization", func(t *testing.T) {
		if !strings.Contains(code, ".Map == nil") {
			t.Error("Expected map deserialization to check for nil Map")
		}
		if !strings.Contains(code, "entry.Key") {
			t.Error("Expected map deserialization to access entry.Key")
		}
		if !strings.Contains(code, "entry.Value") {
			t.Error("Expected map deserialization to access entry.Value")
		}
	})
}

// TestMapTypesCompiles verifies that the generated code for map types compiles
func TestMapTypesCompiles(t *testing.T) {
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("go compiler not found, skipping compilation test")
	}

	tmpDir, err := os.MkdirTemp("", "map_types_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	bp, err := LoadBlueprint("../../testdata/map_types/plutus.json")
	if err != nil {
		t.Fatalf("failed to load blueprint: %v", err)
	}

	gen := NewGenerator(bp, GeneratorOptions{PackageName: "maptypes"})
	code, err := gen.Generate()
	if err != nil {
		t.Fatalf("failed to generate code: %v", err)
	}

	outFile := filepath.Join(tmpDir, "maptypes.go")
	if err := os.WriteFile(outFile, []byte(code), 0644); err != nil {
		t.Fatalf("failed to write generated code: %v", err)
	}

	goMod := `module testmod

go 1.21

require github.com/fxamacker/cbor/v2 v2.8.0

require github.com/x448/float16 v0.8.4 // indirect
`
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0644); err != nil {
		t.Fatalf("failed to write go.mod: %v", err)
	}

	tidyCmd := exec.Command("go", "mod", "tidy")
	tidyCmd.Dir = tmpDir
	if output, err := tidyCmd.CombinedOutput(); err != nil {
		t.Fatalf("go mod tidy failed: %v\nOutput: %s", err, output)
	}

	cmd := exec.Command("go", "build", ".")
	cmd.Dir = tmpDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Errorf("generated code failed to compile: %v\nOutput: %s\n\nGenerated code:\n%s", err, output, code)
	}
}

// TestMapTypesRoundTrip tests serialization and deserialization of map types
func TestMapTypesRoundTrip(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "map_types_roundtrip")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	typesDir := filepath.Join(tmpDir, "types")
	if err := os.MkdirAll(typesDir, 0755); err != nil {
		t.Fatalf("failed to create types dir: %v", err)
	}

	bp, err := LoadBlueprint("../../testdata/map_types/plutus.json")
	if err != nil {
		t.Fatalf("failed to load blueprint: %v", err)
	}

	gen := NewGenerator(bp, GeneratorOptions{PackageName: "types"})
	code, err := gen.Generate()
	if err != nil {
		t.Fatalf("failed to generate code: %v", err)
	}

	typesFile := filepath.Join(typesDir, "types.go")
	if err := os.WriteFile(typesFile, []byte(code), 0644); err != nil {
		t.Fatalf("failed to write types file: %v", err)
	}

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
	pd, err := original.ToPlutusData()
	if err != nil {
		return fmt.Errorf("%s ToPlutusData: %v", name, err)
	}

	cborBytes, err := pd.MarshalCBOR()
	if err != nil {
		return fmt.Errorf("%s MarshalCBOR: %v", name, err)
	}

	var decodedPd types.PlutusData
	if err := decodedPd.UnmarshalCBOR(cborBytes); err != nil {
		return fmt.Errorf("%s UnmarshalCBOR: %v", name, err)
	}

	_, err = decode(decodedPd)
	if err != nil {
		return fmt.Errorf("%s FromPlutusData: %v", name, err)
	}

	hex, _ := pd.ToHex()
	fmt.Printf("OK %s: %s\n", name, hex)
	return nil
}

func main() {
	var failed bool

	// Test SimpleIntMap (map[*big.Int]*big.Int)
	simpleIntMap := types.MapSimpleIntMap{
		Values: map[*big.Int]*big.Int{
			big.NewInt(1): big.NewInt(100),
			big.NewInt(2): big.NewInt(200),
			big.NewInt(3): big.NewInt(300),
		},
	}
	if err := testRoundTrip("SimpleIntMap", simpleIntMap, func(pd types.PlutusData) (types.MapSimpleIntMap, error) {
		var v types.MapSimpleIntMap
		return v, v.FromPlutusData(pd)
	}); err != nil {
		fmt.Fprintln(os.Stderr, err)
		failed = true
	}

	// Test StringToIntMap (map[string]*big.Int)
	stringToIntMap := types.MapStringToIntMap{
		Entries: map[string]*big.Int{
			"alice": big.NewInt(42),
			"bob":   big.NewInt(100),
		},
	}
	if err := testRoundTrip("StringToIntMap", stringToIntMap, func(pd types.PlutusData) (types.MapStringToIntMap, error) {
		var v types.MapStringToIntMap
		return v, v.FromPlutusData(pd)
	}); err != nil {
		fmt.Fprintln(os.Stderr, err)
		failed = true
	}

	// Test StringToStringMap (map[string][]byte)
	stringToStringMap := types.MapStringToStringMap{
		Data: map[string][]byte{
			"key1": []byte("value1"),
			"key2": []byte("value2"),
		},
	}
	if err := testRoundTrip("StringToStringMap", stringToStringMap, func(pd types.PlutusData) (types.MapStringToStringMap, error) {
		var v types.MapStringToStringMap
		return v, v.FromPlutusData(pd)
	}); err != nil {
		fmt.Fprintln(os.Stderr, err)
		failed = true
	}

	// Test WithMultipleMaps (struct with multiple map fields)
	withMultipleMaps := types.MapWithMultipleMaps{
		Name: []byte("test"),
		Scores: map[string]*big.Int{
			"player1": big.NewInt(100),
			"player2": big.NewInt(200),
		},
		Metadata: map[string][]byte{
			"version": []byte("1.0"),
			"author":  []byte("test"),
		},
	}
	if err := testRoundTrip("WithMultipleMaps", withMultipleMaps, func(pd types.PlutusData) (types.MapWithMultipleMaps, error) {
		var v types.MapWithMultipleMaps
		return v, v.FromPlutusData(pd)
	}); err != nil {
		fmt.Fprintln(os.Stderr, err)
		failed = true
	}

	// Test empty map
	emptyMap := types.MapSimpleIntMap{
		Values: map[*big.Int]*big.Int{},
	}
	if err := testRoundTrip("EmptyMap", emptyMap, func(pd types.PlutusData) (types.MapSimpleIntMap, error) {
		var v types.MapSimpleIntMap
		return v, v.FromPlutusData(pd)
	}); err != nil {
		fmt.Fprintln(os.Stderr, err)
		failed = true
	}

	if failed {
		os.Exit(1)
	}
	fmt.Println("\nOK All map types round-trip tests passed!")
}
`

	mainFile := filepath.Join(tmpDir, "main.go")
	if err := os.WriteFile(mainFile, []byte(testProgram), 0644); err != nil {
		t.Fatalf("failed to write main file: %v", err)
	}

	goModContent := `module testpkg

go 1.21

require github.com/fxamacker/cbor/v2 v2.8.0

require github.com/x448/float16 v0.8.4 // indirect
`
	goModFile := filepath.Join(tmpDir, "go.mod")
	if err := os.WriteFile(goModFile, []byte(goModContent), 0644); err != nil {
		t.Fatalf("failed to write go.mod: %v", err)
	}

	cmd := exec.Command("go", "mod", "tidy")
	cmd.Dir = tmpDir
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("go mod tidy failed: %v\n%s", err, output)
	}

	cmd = exec.Command("go", "run", "main.go")
	cmd.Dir = tmpDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("test program failed: %v\n%s", err, output)
	}

	t.Logf("Test output:\n%s", output)
}
