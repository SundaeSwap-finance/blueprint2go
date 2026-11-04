package blueprint

import (
	"bytes"
	"encoding/hex"
	"math/big"
	"os"
	"strings"
	"testing"
)

func TestPlutusData_NestedPreviewContinuation(t *testing.T) {
	// Confirmed Preview transaction 0659755da92cde8bdb225942159dcc21eb692bdaab7b8d992d3f3ff6ca184fc5,
	// output 3, commits a RealFi/Liqwid loop through nested inline datums.
	fixture, err := os.ReadFile("testdata/preview_0659755_output3.hex")
	if err != nil {
		t.Fatal(err)
	}
	raw, err := hex.DecodeString(strings.TrimSpace(string(fixture)))
	if err != nil {
		t.Fatal(err)
	}
	var decoded PlutusData
	if err := decoded.UnmarshalCBOR(raw); err != nil {
		t.Fatal(err)
	}
	if decoded.Constr == nil || decoded.Constr.Index != 0 || len(decoded.Constr.Fields) != 7 {
		t.Fatal("expected the seven-field Sundae v4 order datum")
	}
	budget := decoded.Constr.Fields[2].Integer
	if budget == nil || budget.Cmp(big.NewInt(7_000_000)) != 0 {
		t.Fatalf("unexpected service budget: %v", budget)
	}
	encoded, err := decoded.MarshalCBOR()
	if err != nil {
		t.Fatal(err)
	}
	var roundTrip PlutusData
	if err := roundTrip.UnmarshalCBOR(encoded); err != nil {
		t.Fatal(err)
	}
	if !decoded.Equals(roundTrip) {
		t.Fatal("round trip changes the committed continuation")
	}
}

func TestPlutusData_RejectsExcessiveNesting(t *testing.T) {
	raw := append(bytes.Repeat([]byte{0x81}, 16_385), 0x00)
	var decoded PlutusData
	err := decoded.UnmarshalCBOR(raw)
	if err == nil || !strings.Contains(err.Error(), "exceeded max nested level 16384") {
		t.Fatalf("expected nesting limit error, got %v", err)
	}
}
