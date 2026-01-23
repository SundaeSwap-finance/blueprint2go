package blueprint

import (
	"bytes"
	"errors"
	"fmt"
	"math/big"
	"reflect"

	"github.com/fxamacker/cbor/v2"
)

// PlutusData represents a Plutus Data value that can be serialized to CBOR.
type PlutusData struct {
	Constr     *ConstrPlutusData
	Integer    *big.Int
	ByteString []byte
	List       []PlutusData
	Map        []PlutusDataMapEntry
}

// ConstrPlutusData represents a constructor with an index and fields.
type ConstrPlutusData struct {
	Index  uint64
	Fields []PlutusData
}

// PlutusDataMapEntry represents a key-value pair in a Plutus Data map.
type PlutusDataMapEntry struct {
	Key   PlutusData
	Value PlutusData
}

const (
	cborTagConstr0    = 121
	cborTagConstr6    = 127
	cborTagConstrBase = 1280
)

// NewConstrPlutusData creates a new constructor PlutusData.
func NewConstrPlutusData(index uint64, fields ...PlutusData) PlutusData {
	return PlutusData{Constr: &ConstrPlutusData{Index: index, Fields: fields}}
}

// NewIntPlutusData creates a new integer PlutusData.
func NewIntPlutusData(i *big.Int) PlutusData {
	return PlutusData{Integer: i}
}

// NewBytesPlutusData creates a new bytestring PlutusData.
func NewBytesPlutusData(b []byte) PlutusData {
	return PlutusData{ByteString: b}
}

// NewListPlutusData creates a new list PlutusData.
func NewListPlutusData(items ...PlutusData) PlutusData {
	return PlutusData{List: items}
}

// NewMapPlutusData creates a new map PlutusData.
func NewMapPlutusData(entries ...PlutusDataMapEntry) PlutusData {
	return PlutusData{Map: entries}
}

// MarshalCBOR serializes PlutusData to CBOR bytes using indefinite-length arrays.
func (p PlutusData) MarshalCBOR() ([]byte, error) {
	return p.toCBORBytes()
}

// UnmarshalCBOR deserializes PlutusData from CBOR bytes.
func (p *PlutusData) UnmarshalCBOR(data []byte) error {
	dm, err := cbor.DecOptions{BigIntDec: cbor.BigIntDecodePointer}.DecMode()
	if err != nil {
		return err
	}
	var raw interface{}
	if err := dm.Unmarshal(data, &raw); err != nil {
		return err
	}
	result, err := plutusDataFromCBORValue(raw)
	if err != nil {
		return err
	}
	*p = result
	return nil
}

func (p PlutusData) toCBORBytes() ([]byte, error) {
	em, err := cbor.EncOptions{BigIntConvert: cbor.BigIntConvertShortest}.EncMode()
	if err != nil {
		return nil, err
	}
	switch {
	case p.Constr != nil:
		var buf bytes.Buffer
		// Encode constructor tag
		var tag uint64
		if p.Constr.Index <= 6 {
			tag = cborTagConstr0 + p.Constr.Index
		} else {
			tag = cborTagConstrBase + p.Constr.Index - 7
		}
		// Write CBOR tag header (minimal encoding)
		if tag < 24 {
			buf.WriteByte(0xc0 + byte(tag))
		} else if tag < 256 {
			buf.WriteByte(0xd8)
			buf.WriteByte(byte(tag))
		} else {
			buf.WriteByte(0xd9)
			buf.WriteByte(byte(tag >> 8))
			buf.WriteByte(byte(tag))
		}
		// Empty arrays use definite-length encoding, non-empty use indefinite
		if len(p.Constr.Fields) == 0 {
			buf.WriteByte(0x80) // empty array
		} else {
			buf.WriteByte(0x9f) // indefinite-length array start
			for _, f := range p.Constr.Fields {
				fieldBytes, err := f.toCBORBytes()
				if err != nil {
					return nil, err
				}
				buf.Write(fieldBytes)
			}
			buf.WriteByte(0xff) // break
		}
		return buf.Bytes(), nil
	case p.Integer != nil:
		return em.Marshal(p.Integer)
	case p.ByteString != nil:
		return em.Marshal(p.ByteString)
	case p.List != nil:
		var buf bytes.Buffer
		// Write indefinite-length array start
		buf.WriteByte(0x9f)
		for _, item := range p.List {
			itemBytes, err := item.toCBORBytes()
			if err != nil {
				return nil, err
			}
			buf.Write(itemBytes)
		}
		// Write indefinite-length array end (break)
		buf.WriteByte(0xff)
		return buf.Bytes(), nil
	case p.Map != nil:
		var buf bytes.Buffer
		// Empty maps use definite-length, non-empty use indefinite
		if len(p.Map) == 0 {
			buf.WriteByte(0xa0) // empty map (definite-length)
		} else {
			buf.WriteByte(0xbf) // indefinite-length map start
			for _, entry := range p.Map {
				keyBytes, err := entry.Key.toCBORBytes()
				if err != nil {
					return nil, err
				}
				buf.Write(keyBytes)
				valBytes, err := entry.Value.toCBORBytes()
				if err != nil {
					return nil, err
				}
				buf.Write(valBytes)
			}
			buf.WriteByte(0xff) // break
		}
		return buf.Bytes(), nil
	default:
		// Empty constructor 0
		return []byte{0xd8, 0x79, 0x9f, 0xff}, nil
	}
}

func plutusDataFromCBORValue(v interface{}) (PlutusData, error) {
	switch val := v.(type) {
	case cbor.Tag:
		var index uint64
		switch {
		case val.Number >= cborTagConstr0 && val.Number <= cborTagConstr6:
			index = val.Number - cborTagConstr0
		case val.Number >= cborTagConstrBase:
			index = val.Number - cborTagConstrBase + 7
		default:
			return PlutusData{}, fmt.Errorf("unsupported CBOR tag: %d", val.Number)
		}
		content, ok := val.Content.([]interface{})
		if !ok {
			return PlutusData{}, errors.New("constructor content is not an array")
		}
		fields := make([]PlutusData, len(content))
		for i, item := range content {
			pd, err := plutusDataFromCBORValue(item)
			if err != nil {
				return PlutusData{}, err
			}
			fields[i] = pd
		}
		return PlutusData{Constr: &ConstrPlutusData{Index: index, Fields: fields}}, nil
	case *big.Int:
		return PlutusData{Integer: val}, nil
	case int64:
		return PlutusData{Integer: big.NewInt(val)}, nil
	case uint64:
		return PlutusData{Integer: new(big.Int).SetUint64(val)}, nil
	case []byte:
		return PlutusData{ByteString: val}, nil
	case cbor.ByteString:
		return PlutusData{ByteString: []byte(val)}, nil
	case []interface{}:
		items := make([]PlutusData, len(val))
		for i, item := range val {
			pd, err := plutusDataFromCBORValue(item)
			if err != nil {
				return PlutusData{}, err
			}
			items[i] = pd
		}
		return PlutusData{List: items}, nil
	case map[interface{}]interface{}:
		entries := make([]PlutusDataMapEntry, 0, len(val))
		for k, v := range val {
			key, err := plutusDataFromCBORValue(k)
			if err != nil {
				return PlutusData{}, err
			}
			value, err := plutusDataFromCBORValue(v)
			if err != nil {
				return PlutusData{}, err
			}
			entries = append(entries, PlutusDataMapEntry{Key: key, Value: value})
		}
		return PlutusData{Map: entries}, nil
	case nil:
		// CBOR null - not standard PlutusData, but some serializers use it
		// Treat as Void/Unit (constructor 0 with no fields)
		return PlutusData{Constr: &ConstrPlutusData{Index: 0, Fields: []PlutusData{}}}, nil
	default:
		return PlutusData{}, fmt.Errorf("unsupported CBOR type: %T", v)
	}
}

// ToHex returns the CBOR encoding as a hex string.
func (p PlutusData) ToHex() (string, error) {
	data, err := p.MarshalCBOR()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", data), nil
}

// Equals compares two PlutusData values for equality.
func (p PlutusData) Equals(other PlutusData) bool {
	a, err := p.MarshalCBOR()
	if err != nil {
		return false
	}
	b, err := other.MarshalCBOR()
	if err != nil {
		return false
	}
	return bytes.Equal(a, b)
}

func plutusDataTypeString(pd PlutusData) string {
	switch {
	case pd.Constr != nil:
		return fmt.Sprintf("constructor(%d)", pd.Constr.Index)
	case pd.Integer != nil:
		return "integer"
	case pd.ByteString != nil:
		return "bytes"
	case pd.List != nil:
		return "list"
	case pd.Map != nil:
		return "map"
	default:
		return "null"
	}
}

var _ = errors.New
var _ = big.NewInt
var _ = PlutusData{}
var _ = reflect.DeepEqual
