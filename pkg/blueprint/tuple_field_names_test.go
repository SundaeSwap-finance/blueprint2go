package blueprint

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// Helper to create a temporary blueprint file and load it
func loadBlueprintFromJSON(t *testing.T, jsonContent string) *Blueprint {
	t.Helper()
	tmpFile, err := os.CreateTemp("", "blueprint_*.json")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(jsonContent); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}
	tmpFile.Close()

	bp, err := LoadBlueprint(tmpFile.Name())
	if err != nil {
		t.Fatalf("failed to load blueprint: %v", err)
	}
	return bp
}

// TestTupleFieldNamesFromRefs tests that tuple field names are extracted from type references
func TestTupleFieldNamesFromRefs(t *testing.T) {
	// Create a blueprint with named type references
	blueprintJSON := `{
  "preamble": {
    "title": "test/tuple_names",
    "version": "1.0.0",
    "plutusVersion": "v3"
  },
  "validators": [],
  "definitions": {
    "PolicyId": {
      "dataType": "bytes"
    },
    "AssetName": {
      "dataType": "bytes"
    },
    "Asset": {
      "title": "Asset",
      "dataType": "list",
      "items": [
        {"$ref": "#/definitions/PolicyId"},
        {"$ref": "#/definitions/AssetName"}
      ]
    }
  }
}`

	bp := loadBlueprintFromJSON(t, blueprintJSON)

	gen := NewGenerator(bp, GeneratorOptions{PackageName: "test"})
	code, err := gen.Generate()
	if err != nil {
		t.Fatalf("failed to generate code: %v", err)
	}

	// Check that field names are extracted from refs
	if !strings.Contains(code, "PolicyId []byte") {
		t.Error("Expected 'PolicyId []byte' field, got generic Field0")
	}
	if !strings.Contains(code, "AssetName []byte") {
		t.Error("Expected 'AssetName []byte' field, got generic Field1")
	}

	// Should NOT contain generic field names
	if strings.Contains(code, "Field0") {
		t.Error("Should not use generic Field0 when type name is available")
	}
	if strings.Contains(code, "Field1") {
		t.Error("Should not use generic Field1 when type name is available")
	}
}

// TestTupleFieldNamesDuplicates tests handling of duplicate type names in tuples
func TestTupleFieldNamesDuplicates(t *testing.T) {
	// Create a blueprint with duplicate types in a tuple
	blueprintJSON := `{
  "preamble": {
    "title": "test/tuple_duplicates",
    "version": "1.0.0",
    "plutusVersion": "v3"
  },
  "validators": [],
  "definitions": {
    "Int": {
      "dataType": "integer"
    },
    "ByteArray": {
      "dataType": "bytes"
    },
    "ThreeInts": {
      "title": "ThreeInts",
      "dataType": "list",
      "items": [
        {"$ref": "#/definitions/Int"},
        {"$ref": "#/definitions/Int"},
        {"$ref": "#/definitions/Int"}
      ]
    },
    "MixedDuplicates": {
      "title": "MixedDuplicates",
      "dataType": "list",
      "items": [
        {"$ref": "#/definitions/Int"},
        {"$ref": "#/definitions/ByteArray"},
        {"$ref": "#/definitions/Int"},
        {"$ref": "#/definitions/ByteArray"}
      ]
    }
  }
}`

	bp := loadBlueprintFromJSON(t, blueprintJSON)

	gen := NewGenerator(bp, GeneratorOptions{PackageName: "test"})
	code, err := gen.Generate()
	if err != nil {
		t.Fatalf("failed to generate code: %v", err)
	}

	// Check ThreeInts - should have Int, Int2, Int3
	if !strings.Contains(code, "type ThreeInts struct") {
		t.Error("Expected ThreeInts struct")
	}
	// First Int should not have a number suffix
	if !strings.Contains(code, "Int *big.Int") {
		t.Error("Expected first 'Int *big.Int' field")
	}
	if !strings.Contains(code, "Int2 *big.Int") {
		t.Error("Expected 'Int2 *big.Int' for second Int")
	}
	if !strings.Contains(code, "Int3 *big.Int") {
		t.Error("Expected 'Int3 *big.Int' for third Int")
	}

	// Check MixedDuplicates - should have Int, ByteArray, Int2, ByteArray2
	if !strings.Contains(code, "type MixedDuplicates struct") {
		t.Error("Expected MixedDuplicates struct")
	}
	if !strings.Contains(code, "ByteArray []byte") {
		t.Error("Expected first 'ByteArray []byte' field")
	}
	if !strings.Contains(code, "ByteArray2 []byte") {
		t.Error("Expected 'ByteArray2 []byte' for second ByteArray")
	}
}

// TestTupleFieldNamesFallback tests fallback to Field0, Field1 when no ref name available
func TestTupleFieldNamesFallback(t *testing.T) {
	// Create a blueprint with inline type definitions (no refs)
	blueprintJSON := `{
  "preamble": {
    "title": "test/tuple_fallback",
    "version": "1.0.0",
    "plutusVersion": "v3"
  },
  "validators": [],
  "definitions": {
    "InlineTuple": {
      "title": "InlineTuple",
      "dataType": "list",
      "items": [
        {"dataType": "integer"},
        {"dataType": "bytes"}
      ]
    }
  }
}`

	bp := loadBlueprintFromJSON(t, blueprintJSON)

	gen := NewGenerator(bp, GeneratorOptions{PackageName: "test"})
	code, err := gen.Generate()
	if err != nil {
		t.Fatalf("failed to generate code: %v", err)
	}

	// Check that fallback field names are used
	if !strings.Contains(code, "Field0 *big.Int") {
		t.Error("Expected 'Field0 *big.Int' for inline integer type")
	}
	if !strings.Contains(code, "Field1 []byte") {
		t.Error("Expected 'Field1 []byte' for inline bytes type")
	}
}

// TestTupleFieldNamesNestedPath tests extraction from nested paths like cardano/assets/PolicyId
func TestTupleFieldNamesNestedPath(t *testing.T) {
	// Create a blueprint with nested path references (like cardano~1assets~1PolicyId)
	blueprintJSON := `{
  "preamble": {
    "title": "test/tuple_nested",
    "version": "1.0.0",
    "plutusVersion": "v3"
  },
  "validators": [],
  "definitions": {
    "cardano/assets/PolicyId": {
      "dataType": "bytes"
    },
    "cardano/assets/AssetName": {
      "dataType": "bytes"
    },
    "Asset": {
      "title": "Asset",
      "dataType": "list",
      "items": [
        {"$ref": "#/definitions/cardano~1assets~1PolicyId"},
        {"$ref": "#/definitions/cardano~1assets~1AssetName"}
      ]
    }
  }
}`

	bp := loadBlueprintFromJSON(t, blueprintJSON)

	gen := NewGenerator(bp, GeneratorOptions{PackageName: "test"})
	code, err := gen.Generate()
	if err != nil {
		t.Fatalf("failed to generate code: %v", err)
	}

	// Check that the last part of the path is used as field name
	if !strings.Contains(code, "PolicyId []byte") {
		t.Error("Expected 'PolicyId []byte' extracted from nested path")
	}
	if !strings.Contains(code, "AssetName []byte") {
		t.Error("Expected 'AssetName []byte' extracted from nested path")
	}
}

// TestTupleFieldNamesCompiles verifies that generated code with named fields compiles
func TestTupleFieldNamesCompiles(t *testing.T) {
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("go compiler not found, skipping compilation test")
	}

	tmpDir, err := os.MkdirTemp("", "tuple_names_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Blueprint with various tuple patterns
	blueprintJSON := `{
  "preamble": {
    "title": "test/tuple_compile",
    "version": "1.0.0",
    "plutusVersion": "v3"
  },
  "validators": [],
  "definitions": {
    "PolicyId": {"dataType": "bytes"},
    "AssetName": {"dataType": "bytes"},
    "Int": {"dataType": "integer"},
    "Asset": {
      "title": "Asset",
      "dataType": "list",
      "items": [
        {"$ref": "#/definitions/PolicyId"},
        {"$ref": "#/definitions/AssetName"}
      ]
    },
    "Point": {
      "title": "Point",
      "dataType": "list",
      "items": [
        {"$ref": "#/definitions/Int"},
        {"$ref": "#/definitions/Int"}
      ]
    },
    "InlineTuple": {
      "title": "InlineTuple",
      "dataType": "list",
      "items": [
        {"dataType": "integer"},
        {"dataType": "bytes"},
        {"dataType": "integer"}
      ]
    }
  }
}`

	bp := loadBlueprintFromJSON(t, blueprintJSON)

	gen := NewGenerator(bp, GeneratorOptions{PackageName: "tupletest"})
	code, err := gen.Generate()
	if err != nil {
		t.Fatalf("failed to generate code: %v", err)
	}

	outFile := filepath.Join(tmpDir, "tupletest.go")
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

// TestTupleFieldNamesRoundTrip tests serialization/deserialization with named fields
func TestTupleFieldNamesRoundTrip(t *testing.T) {
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("go compiler not found, skipping round-trip test")
	}

	tmpDir, err := os.MkdirTemp("", "tuple_roundtrip_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	typesDir := filepath.Join(tmpDir, "types")
	if err := os.MkdirAll(typesDir, 0755); err != nil {
		t.Fatalf("failed to create types dir: %v", err)
	}

	blueprintJSON := `{
  "preamble": {
    "title": "test/roundtrip",
    "version": "1.0.0",
    "plutusVersion": "v3"
  },
  "validators": [],
  "definitions": {
    "PolicyId": {"dataType": "bytes"},
    "AssetName": {"dataType": "bytes"},
    "Int": {"dataType": "integer"},
    "Asset": {
      "title": "Asset",
      "dataType": "list",
      "items": [
        {"$ref": "#/definitions/PolicyId"},
        {"$ref": "#/definitions/AssetName"}
      ]
    },
    "Point": {
      "title": "Point",
      "dataType": "list",
      "items": [
        {"$ref": "#/definitions/Int"},
        {"$ref": "#/definitions/Int"}
      ]
    }
  }
}`

	bp := loadBlueprintFromJSON(t, blueprintJSON)

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

	fmt.Printf("OK %s\n", name)
	return nil
}

func main() {
	var failed bool

	// Test Asset with named fields (PolicyId, AssetName)
	asset := types.Asset{
		PolicyId:  []byte{0xab, 0xcd, 0xef},
		AssetName: []byte("MyToken"),
	}
	if err := testRoundTrip("Asset", asset, func(pd types.PlutusData) (types.Asset, error) {
		var v types.Asset
		return v, v.FromPlutusData(pd)
	}); err != nil {
		fmt.Fprintln(os.Stderr, err)
		failed = true
	}

	// Test Point with duplicate type names (Int, Int2)
	point := types.Point{
		Int:  big.NewInt(10),
		Int2: big.NewInt(20),
	}
	if err := testRoundTrip("Point", point, func(pd types.PlutusData) (types.Point, error) {
		var v types.Point
		return v, v.FromPlutusData(pd)
	}); err != nil {
		fmt.Fprintln(os.Stderr, err)
		failed = true
	}

	if failed {
		os.Exit(1)
	}
	fmt.Println("\nOK All tuple field names tests passed!")
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
