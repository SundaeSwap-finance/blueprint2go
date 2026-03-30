# blueprint2go

Generate Go types from Aiken's CIP-0057 Plutus Blueprint for building Cardano transactions.

## Overview

`blueprint2go` reads Aiken's `plutus.json` blueprint files and generates **standalone** Go types that can be serialized to/from CBOR Plutus Data format. This allows you to construct datums and redeemers in Go for Cardano transactions.

The generated code is self-contained with no external dependencies (except for the CBOR library).

## Requirements

- Go 1.21 or later

## Installation

```bash
go install github.com/SundaeSwap-finance/blueprint2go/cmd/blueprint2go@latest
```

Or build from source:

```bash
go build -o blueprint2go ./cmd/blueprint2go
```

## Usage

```bash
blueprint2go -o types.go plutus.json
```

With custom package name:

```bash
blueprint2go -o types.go -p mypackage plutus.json
```

### Options

| Flag             | Description                            |
| ---------------- | -------------------------------------- |
| `-o`, `-outfile` | Output file path (required)            |
| `-p`, `-package` | Go package name (default: `contracts`) |

## Generated Code

The generator produces:

- **PlutusData types** embedded directly in the generated file
- **Struct types** for single-constructor types (records)
- **Interface types** for enums (types with multiple constructors)
- **Variant structs** for each enum variant
- **`ToPlutusData()` methods** for serialization
- **`FromPlutusData()` methods** for deserialization
- **`<Type>FromPlutusData()` factory functions** for decoding enum types

### Type Naming

Type names include the full module path to avoid collisions when multiple modules define types with the same name:

| Aiken Definition          | Go Type Name             |
| ------------------------- | ------------------------ |
| `types/Payout`            | `TypesPayout`            |
| `v0_1/types/Settings`     | `V01TypesSettings`       |
| `multisig/MultisigScript` | `MultisigMultisigScript` |

## Working with Struct Types

For simple struct types (single constructor), use `ToPlutusData()` and `FromPlutusData()` directly:

```go
import "myproject/contracts" // generated code

// Create a datum
datum := contracts.TypesMyDatum{
    Owner:  []byte{0xde, 0xad, 0xbe, 0xef}, // ByteArray fields are []byte
    Amount: big.NewInt(1000000),
}

// Serialize to CBOR
pd, err := datum.ToPlutusData()
if err != nil {
    return err
}
cborBytes, err := pd.MarshalCBOR()

// Or get hex string directly
hexString, err := pd.ToHex()

// Deserialize from CBOR
var pd contracts.PlutusData
if err := pd.UnmarshalCBOR(cborBytes); err != nil {
    return err
}
var decoded contracts.TypesMyDatum
if err := decoded.FromPlutusData(pd); err != nil {
    return err
}
```

## Working with Enum Types

Enum types (Aiken types with multiple constructors) require special handling because you may not know which variant you're decoding until runtime.

### Example Aiken Enum

```aiken
pub type Action {
  Send { to: ByteArray, amount: Int }
  Receive { from: ByteArray }
  Cancel
}
```

### Generated Go Code Structure

```go
// Interface for the enum
type Action interface {
    isAction()
    ToPlutusData() (PlutusData, error)
}

// Factory function to decode any variant
func ActionFromPlutusData(pd PlutusData) (Action, error)

// Variant structs
type ActionSend struct {
    To     []byte
    Amount *big.Int
}
type ActionReceive struct {
    From []byte
}
type ActionCancel struct{}
```

### Encoding an Enum Value

When encoding, you know which variant you're creating:

```go
// Create a Send action
action := contracts.ActionSend{
    To:     []byte{0xde, 0xad, 0xbe, 0xef, 0x12, 0x34},
    Amount: big.NewInt(1000000),
}

// Serialize to CBOR
pd, err := action.ToPlutusData()
if err != nil {
    return err
}
cborBytes, err := pd.MarshalCBOR()
```

### Decoding an Enum Value (Unknown Variant)

When decoding, use the **factory function** to automatically detect the variant:

```go
// 1. Decode CBOR to PlutusData
var pd contracts.PlutusData
if err := pd.UnmarshalCBOR(cborBytes); err != nil {
    return err
}

// 2. Use the factory function - it examines the constructor index
//    and returns the correct variant type
action, err := contracts.ActionFromPlutusData(pd)
if err != nil {
    return err
}

// 3. Use type switch to handle each variant
switch a := action.(type) {
case contracts.ActionSend:
    fmt.Printf("Send %s to %x\n", a.Amount.String(), a.To)
case contracts.ActionReceive:
    fmt.Printf("Receive from %x\n", a.From)
case contracts.ActionCancel:
    fmt.Println("Cancel")
default:
    return fmt.Errorf("unknown action type: %T", a)
}
```

### Decoding an Enum Value (Known Variant)

If you know which variant to expect, you can decode directly:

```go
var pd contracts.PlutusData
pd.UnmarshalCBOR(cborBytes)

// Decode directly to the expected variant
var send contracts.ActionSend
if err := send.FromPlutusData(pd); err != nil {
    // Will fail if the CBOR data is not a Send variant
    return err
}
```

### How the Factory Function Works

The factory function examines `pd.Constr.Index` (the CBOR constructor tag) to determine which variant to instantiate:

| Constructor Index | Aiken Variant | Go Type         |
| ----------------- | ------------- | --------------- |
| 0                 | `Send`        | `ActionSend`    |
| 1                 | `Receive`     | `ActionReceive` |
| 2                 | `Cancel`      | `ActionCancel`  |

The index corresponds to the order of declaration in the Aiken source.

## Type Mappings

| Aiken Type          | Go Type                               |
| ------------------- | ------------------------------------- |
| `Int`               | `*big.Int`                            |
| `ByteArray`         | `[]byte`                              |
| `Bool`              | `bool`                                |
| `List<T>`           | `[]T`                                 |
| `Option<T>`         | `*T` (pointer, nil = None)            |
| `Option<Int>`       | `*big.Int` (nil = None)               |
| `Option<ByteArray>` | `*[]byte` (nil = None)                |
| `Option<EnumType>`  | `EnumType` (interface, nil = None)    |
| `Data`              | `PlutusData`                          |
| Tuple types         | Struct with `Field0`, `Field1`, etc.  |
| Named list types    | Type alias with serialization methods |

## PlutusData Format

The CBOR encoding follows the Plutus Data format:

| Plutus Type     | CBOR Encoding            |
| --------------- | ------------------------ |
| Integer         | CBOR integer/bignum      |
| ByteString      | CBOR bytes               |
| List            | CBOR array               |
| Map             | CBOR map                 |
| Constructor 0-6 | CBOR tag 121-127 + array |
| Constructor 7+  | CBOR tag 1280+n + array  |

## Testing

Run all tests:

```bash
go test ./...
```

Run with verbose output:

```bash
go test ./... -v
```

The `testdata/` directory contains sample Plutus blueprints:
