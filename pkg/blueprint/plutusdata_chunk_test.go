package blueprint

import (
	"bytes"
	"encoding/hex"
	"math/big"
	"strings"
	"testing"
)

// hexRepeat returns the hex encoding of n copies of b.
func hexRepeat(b byte, n int) string {
	return strings.Repeat(hex.EncodeToString([]byte{b}), n)
}

// TestPlutusData_ByteStringChunking pins the wire form of bytestrings around the
// 64-byte chunking boundary. Cardano's Plutus data encoder emits bytestrings
// longer than 64 bytes as an indefinite-length byte string split into chunks of
// at most 64 bytes; shorter ones stay a single definite-length byte string.
func TestPlutusData_ByteStringChunking(t *testing.T) {
	tests := []struct {
		name string
		n    int
		want string
	}{
		{
			name: "empty",
			n:    0,
			want: "40",
		},
		{
			name: "one byte",
			n:    1,
			want: "41" + hexRepeat(0xab, 1),
		},
		{
			name: "exactly 64 bytes is not chunked",
			n:    64,
			want: "5840" + hexRepeat(0xab, 64),
		},
		{
			name: "65 bytes chunks into 64+1",
			n:    65,
			want: "5f" + "5840" + hexRepeat(0xab, 64) + "41" + hexRepeat(0xab, 1) + "ff",
		},
		{
			name: "128 bytes chunks into 64+64 with no remainder",
			n:    128,
			want: "5f" + "5840" + hexRepeat(0xab, 64) + "5840" + hexRepeat(0xab, 64) + "ff",
		},
		{
			name: "129 bytes chunks into 64+64+1",
			n:    129,
			want: "5f" + "5840" + hexRepeat(0xab, 64) + "5840" + hexRepeat(0xab, 64) + "41" + hexRepeat(0xab, 1) + "ff",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pd := NewBytesPlutusData(bytes.Repeat([]byte{0xab}, tt.n))
			got, err := pd.MarshalCBOR()
			if err != nil {
				t.Fatalf("MarshalCBOR: %v", err)
			}
			if h := hex.EncodeToString(got); h != tt.want {
				t.Errorf("MarshalCBOR(%d bytes)\n got %s\nwant %s", tt.n, h, tt.want)
			}
		})
	}
}

// TestPlutusData_ByteStringChunkingGoldenVectors pins fully literal wire bytes so
// the encoding cannot silently change again.
func TestPlutusData_ByteStringChunkingGoldenVectors(t *testing.T) {
	tests := []struct {
		name string
		pd   PlutusData
		want string
	}{
		{
			// The minimal repro: 100 bytes of 0xab -> 64-byte chunk + 36-byte chunk.
			name: "100 bytes bare",
			pd:   NewBytesPlutusData(bytes.Repeat([]byte{0xab}, 100)),
			want: "5f5840" +
				"abababababababababababababababababababababababababababababababab" + // 32 bytes
				"abababababababababababababababababababababababababababababababab" + // 32 bytes
				"5824" +
				"abababababababababababababababababababababababababababababababababababab" + // 36 bytes
				"ff",
		},
		{
			name: "short bytes unchanged",
			pd:   NewBytesPlutusData([]byte{0xde, 0xad, 0xbe, 0xef}),
			want: "44deadbeef",
		},
		{
			name: "inside constructor field",
			pd:   NewConstrPlutusData(0, NewBytesPlutusData(bytes.Repeat([]byte{0x01}, 70))),
			want: "d8799f5f5840" +
				"0101010101010101010101010101010101010101010101010101010101010101" +
				"0101010101010101010101010101010101010101010101010101010101010101" +
				"46010101010101" +
				"ffff",
		},
		{
			name: "as list item",
			pd:   NewListPlutusData(NewBytesPlutusData(bytes.Repeat([]byte{0x02}, 65))),
			want: "9f5f5840" +
				"0202020202020202020202020202020202020202020202020202020202020202" +
				"0202020202020202020202020202020202020202020202020202020202020202" +
				"4102" +
				"ffff",
		},
		{
			name: "as map key and value",
			pd: NewMapPlutusData(PlutusDataMapEntry{
				Key:   NewBytesPlutusData(bytes.Repeat([]byte{0x03}, 65)),
				Value: NewBytesPlutusData(bytes.Repeat([]byte{0x04}, 65)),
			}),
			want: "bf5f5840" +
				"0303030303030303030303030303030303030303030303030303030303030303" +
				"0303030303030303030303030303030303030303030303030303030303030303" +
				"4103" +
				"ff" +
				"5f5840" +
				"0404040404040404040404040404040404040404040404040404040404040404" +
				"0404040404040404040404040404040404040404040404040404040404040404" +
				"4104" +
				"ff" +
				"ff",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.pd.MarshalCBOR()
			if err != nil {
				t.Fatalf("MarshalCBOR: %v", err)
			}
			if h := hex.EncodeToString(got); h != tt.want {
				t.Errorf("MarshalCBOR\n got %s\nwant %s", h, tt.want)
			}
		})
	}
}

// TestPlutusData_ByteStringChunkingDecode asserts the two wire forms remain
// interchangeable on decode, so this change is encode-only.
func TestPlutusData_ByteStringChunkingDecode(t *testing.T) {
	payload := bytes.Repeat([]byte{0xab}, 100)

	unchunked, err := hex.DecodeString("5864" + hexRepeat(0xab, 100))
	if err != nil {
		t.Fatalf("bad fixture: %v", err)
	}
	chunked, err := hex.DecodeString("5f" + "5840" + hexRepeat(0xab, 64) + "5824" + hexRepeat(0xab, 36) + "ff")
	if err != nil {
		t.Fatalf("bad fixture: %v", err)
	}

	var fromUnchunked, fromChunked PlutusData
	if err := fromUnchunked.UnmarshalCBOR(unchunked); err != nil {
		t.Fatalf("UnmarshalCBOR(unchunked): %v", err)
	}
	if err := fromChunked.UnmarshalCBOR(chunked); err != nil {
		t.Fatalf("UnmarshalCBOR(chunked): %v", err)
	}

	if !bytes.Equal(fromUnchunked.ByteString, payload) {
		t.Errorf("unchunked decoded to %x, want %x", fromUnchunked.ByteString, payload)
	}
	if !bytes.Equal(fromChunked.ByteString, payload) {
		t.Errorf("chunked decoded to %x, want %x", fromChunked.ByteString, payload)
	}
	if !bytes.Equal(fromChunked.ByteString, fromUnchunked.ByteString) {
		t.Errorf("chunked and unchunked decode differently: %x vs %x",
			fromChunked.ByteString, fromUnchunked.ByteString)
	}

	// Round trip through the new encoder.
	original := NewBytesPlutusData(payload)
	encoded, err := original.MarshalCBOR()
	if err != nil {
		t.Fatalf("MarshalCBOR: %v", err)
	}
	var roundTripped PlutusData
	if err := roundTripped.UnmarshalCBOR(encoded); err != nil {
		t.Fatalf("UnmarshalCBOR(Marshal(x)): %v", err)
	}
	if !bytes.Equal(roundTripped.ByteString, payload) {
		t.Errorf("round trip lost data: got %x, want %x", roundTripped.ByteString, payload)
	}
}

// TestPlutusData_EqualsLongByteString checks Equals still works for values whose
// encoding is now chunked, including nested inside a constructor.
func TestPlutusData_EqualsLongByteString(t *testing.T) {
	long := bytes.Repeat([]byte{0xab}, 100)
	other := bytes.Repeat([]byte{0xab}, 100)
	other[99] = 0xac

	if !NewBytesPlutusData(long).Equals(NewBytesPlutusData(bytes.Repeat([]byte{0xab}, 100))) {
		t.Error("equal long bytestrings compared unequal")
	}
	if NewBytesPlutusData(long).Equals(NewBytesPlutusData(other)) {
		t.Error("differing long bytestrings compared equal")
	}
	// Differing only in length across the chunk boundary.
	if NewBytesPlutusData(bytes.Repeat([]byte{0xab}, 64)).Equals(NewBytesPlutusData(bytes.Repeat([]byte{0xab}, 65))) {
		t.Error("64- and 65-byte bytestrings compared equal")
	}

	a := NewConstrPlutusData(1, NewBytesPlutusData(long), NewIntPlutusData(big.NewInt(7)))
	b := NewConstrPlutusData(1, NewBytesPlutusData(bytes.Repeat([]byte{0xab}, 100)), NewIntPlutusData(big.NewInt(7)))
	if !a.Equals(b) {
		t.Error("constructors holding equal long bytestrings compared unequal")
	}
}
