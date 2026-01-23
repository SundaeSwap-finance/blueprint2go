package blueprint

import (
	"encoding/hex"
	"math/big"
	"testing"
)

// fromHex is a test helper that decodes hex and unmarshals PlutusData
func fromHex(t *testing.T, h string) PlutusData {
	t.Helper()
	data, err := hex.DecodeString(h)
	if err != nil {
		t.Fatalf("invalid hex: %v", err)
	}
	var pd PlutusData
	if err := pd.UnmarshalCBOR(data); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	return pd
}

func TestPlutusData_Integer(t *testing.T) {
	// Test positive integer
	pd := NewIntPlutusData(big.NewInt(42))
	data, err := pd.MarshalCBOR()
	if err != nil {
		t.Fatalf("failed to marshal integer: %v", err)
	}

	var decoded PlutusData
	if err := decoded.UnmarshalCBOR(data); err != nil {
		t.Fatalf("failed to unmarshal integer: %v", err)
	}

	if decoded.Integer == nil || decoded.Integer.Cmp(big.NewInt(42)) != 0 {
		t.Errorf("expected 42, got %v", decoded.Integer)
	}
}

func TestPlutusData_ByteString(t *testing.T) {
	pd := NewBytesPlutusData([]byte{0xde, 0xad, 0xbe, 0xef})
	data, err := pd.MarshalCBOR()
	if err != nil {
		t.Fatalf("failed to marshal bytes: %v", err)
	}

	var decoded PlutusData
	if err := decoded.UnmarshalCBOR(data); err != nil {
		t.Fatalf("failed to unmarshal bytes: %v", err)
	}

	if string(decoded.ByteString) != string([]byte{0xde, 0xad, 0xbe, 0xef}) {
		t.Errorf("expected deadbeef, got %x", decoded.ByteString)
	}
}

func TestPlutusData_List(t *testing.T) {
	pd := NewListPlutusData(
		NewIntPlutusData(big.NewInt(1)),
		NewIntPlutusData(big.NewInt(2)),
		NewIntPlutusData(big.NewInt(3)),
	)
	data, err := pd.MarshalCBOR()
	if err != nil {
		t.Fatalf("failed to marshal list: %v", err)
	}

	var decoded PlutusData
	if err := decoded.UnmarshalCBOR(data); err != nil {
		t.Fatalf("failed to unmarshal list: %v", err)
	}

	if len(decoded.List) != 3 {
		t.Errorf("expected 3 items, got %d", len(decoded.List))
	}
}

func TestPlutusData_Constructor(t *testing.T) {
	// Test constructor index 0 (tag 121)
	pd := NewConstrPlutusData(0, NewIntPlutusData(big.NewInt(42)))
	data, err := pd.MarshalCBOR()
	if err != nil {
		t.Fatalf("failed to marshal constructor: %v", err)
	}

	var decoded PlutusData
	if err := decoded.UnmarshalCBOR(data); err != nil {
		t.Fatalf("failed to unmarshal constructor: %v", err)
	}

	if decoded.Constr == nil {
		t.Fatal("expected constructor")
	}
	if decoded.Constr.Index != 0 {
		t.Errorf("expected index 0, got %d", decoded.Constr.Index)
	}
	if len(decoded.Constr.Fields) != 1 {
		t.Errorf("expected 1 field, got %d", len(decoded.Constr.Fields))
	}
}

func TestPlutusData_Constructor_HighIndex(t *testing.T) {
	// Test constructor index 7+ (tag 1280+)
	pd := NewConstrPlutusData(10)
	data, err := pd.MarshalCBOR()
	if err != nil {
		t.Fatalf("failed to marshal constructor: %v", err)
	}

	var decoded PlutusData
	if err := decoded.UnmarshalCBOR(data); err != nil {
		t.Fatalf("failed to unmarshal constructor: %v", err)
	}

	if decoded.Constr == nil {
		t.Fatal("expected constructor")
	}
	if decoded.Constr.Index != 10 {
		t.Errorf("expected index 10, got %d", decoded.Constr.Index)
	}
}

func TestPlutusData_Unit(t *testing.T) {
	// Unit/Void is constructor 0 with no fields
	pd := NewConstrPlutusData(0)

	hex, err := pd.ToHex()
	if err != nil {
		t.Fatalf("failed to marshal unit: %v", err)
	}
	// Constructor 0 with empty fields: d87980 (tag 121 + empty array)
	if hex != "d87980" {
		t.Errorf("expected d87980, got %s", hex)
	}
}

func TestPlutusData_Bool(t *testing.T) {
	// False = constructor 0, True = constructor 1
	falsePd := NewConstrPlutusData(0)
	truePd := NewConstrPlutusData(1)

	falseHex, _ := falsePd.ToHex()
	trueHex, _ := truePd.ToHex()

	if falseHex != "d87980" {
		t.Errorf("False: expected d87980, got %s", falseHex)
	}
	if trueHex != "d87a80" {
		t.Errorf("True: expected d87a80, got %s", trueHex)
	}
}

func TestPlutusData_FromHex(t *testing.T) {
	// Test decoding from known hex
	pd := fromHex(t, "d87980") // Unit/False

	if pd.Constr == nil || pd.Constr.Index != 0 {
		t.Error("expected constructor 0")
	}
}

func TestPlutusData_RoundTrip(t *testing.T) {
	// Complex nested structure
	original := NewConstrPlutusData(0,
		NewIntPlutusData(big.NewInt(12345)),
		NewBytesPlutusData([]byte("hello")),
		NewListPlutusData(
			NewIntPlutusData(big.NewInt(1)),
			NewIntPlutusData(big.NewInt(2)),
		),
		NewConstrPlutusData(1,
			NewBytesPlutusData([]byte("nested")),
		),
	)

	data, err := original.MarshalCBOR()
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var decoded PlutusData
	if err := decoded.UnmarshalCBOR(data); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if !original.Equals(decoded) {
		t.Error("round-trip failed: values don't match")
	}
}

func TestPlutusData_IndefiniteLengthArrays(t *testing.T) {
	// This test verifies that CBOR encoding matches Aiken's format:
	// - Non-empty arrays use indefinite-length encoding (0x9f...0xff)
	// - Empty arrays use definite-length encoding (0x80)
	// - Tags use minimal encoding (0xd8 0x79 for tag 121, not 0xd9 0x00 0x79)

	t.Run("EmptyConstructor", func(t *testing.T) {
		// Constructor 0 with no fields should be: d879 80 (tag 121 + empty array)
		pd := NewConstrPlutusData(0)
		data, err := pd.MarshalCBOR()
		if err != nil {
			t.Fatalf("failed to marshal: %v", err)
		}
		expected := []byte{0xd8, 0x79, 0x80}
		if string(data) != string(expected) {
			t.Errorf("expected %x, got %x", expected, data)
		}
	})

	t.Run("NonEmptyConstructor", func(t *testing.T) {
		// Constructor 0 with one integer field should use indefinite array:
		// d879 9f 01 ff (tag 121 + indefinite array + int 1 + break)
		pd := NewConstrPlutusData(0, NewIntPlutusData(big.NewInt(1)))
		data, err := pd.MarshalCBOR()
		if err != nil {
			t.Fatalf("failed to marshal: %v", err)
		}
		expected := []byte{0xd8, 0x79, 0x9f, 0x01, 0xff}
		if string(data) != string(expected) {
			t.Errorf("expected %x, got %x", expected, data)
		}
	})

	t.Run("List", func(t *testing.T) {
		// List with two integers should use indefinite array:
		// 9f 01 02 ff (indefinite array + int 1 + int 2 + break)
		pd := NewListPlutusData(
			NewIntPlutusData(big.NewInt(1)),
			NewIntPlutusData(big.NewInt(2)),
		)
		data, err := pd.MarshalCBOR()
		if err != nil {
			t.Fatalf("failed to marshal: %v", err)
		}
		expected := []byte{0x9f, 0x01, 0x02, 0xff}
		if string(data) != string(expected) {
			t.Errorf("expected %x, got %x", expected, data)
		}
	})

	t.Run("NestedStructure", func(t *testing.T) {
		// Constr(0, [Constr(1, [])]) should be:
		// d879 9f d87a 80 ff
		// tag121 + indef[ tag122 + empty[] ] + break
		pd := NewConstrPlutusData(0, NewConstrPlutusData(1))
		data, err := pd.MarshalCBOR()
		if err != nil {
			t.Fatalf("failed to marshal: %v", err)
		}
		expected := []byte{0xd8, 0x79, 0x9f, 0xd8, 0x7a, 0x80, 0xff}
		if string(data) != string(expected) {
			t.Errorf("expected %x, got %x", expected, data)
		}
	})

	t.Run("MatchesAikenFormat", func(t *testing.T) {
		// This is the exact CBOR produced by Aiken for a ProtocolRedeemer(Mint, [Request(...)])
		// We decode it and re-encode to verify byte-for-byte match
		aikenHex := "d8799f9fd8799fd8799fd8799fd8799f450102000403ffd8799fd8799fd8799f450102000403ffffffffd87980ff01d8799f4ed8799f48736f6d65486173680cffffffffff"

		pd := fromHex(t, aikenHex)

		reencoded, err := pd.ToHex()
		if err != nil {
			t.Fatalf("failed to re-encode: %v", err)
		}

		if reencoded != aikenHex {
			t.Errorf("CBOR mismatch:\nexpected: %s\ngot:      %s", aikenHex, reencoded)
		}
	})
}
