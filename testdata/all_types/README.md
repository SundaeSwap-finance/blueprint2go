# all_types

Test data for validating Go code generation from various Aiken types.

## Purpose

This Aiken project defines a comprehensive set of types used to test the `aiken2go` code generator. The types cover:

- Simple structs (single field)
- Structs with multiple fields
- Primitive types (Int, ByteArray, Bool)
- List types (List<Int>, List<ByteArray>, List<Struct>)
- Option types (Option<Int>, Option<Struct>)
- Enum types (multiple constructors)
- Nested types (struct containing struct)

## Important: Updating plutus.json

The `plutus.json` file is the blueprint used by `aiken2go` to generate Go types. **It is NOT automatically updated by the Go tests.**

If you modify the types in `validators/string_validator.ak`, you must manually regenerate `plutus.json`:

```sh
cd testdata/all_types
aiken build
```

This will regenerate `plutus.json` with the updated type definitions.

## Files

- `validators/string_validator.ak` - Aiken type definitions
- `plutus.json` - Generated blueprint (input for aiken2go)
- `aiken.toml` - Aiken project configuration
