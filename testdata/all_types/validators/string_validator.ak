use cardano/transaction.{Transaction, OutputReference}

// =============================================================================
// Test types covering various Aiken type constructs
// =============================================================================

/// Simple record with a single ByteArray field
pub type SimpleString {
  message: ByteArray,
}

/// Record with an integer field
pub type SimpleInt {
  value: Int,
}

/// Record with multiple primitive fields
pub type MultipleFields {
  name: ByteArray,
  age: Int,
  active: Bool,
}

/// Record with an optional field
pub type WithOption {
  label: ByteArray,
  maybe_value: Option<Int>,
}

/// Record with a list field
pub type WithList {
  items: List<Int>,
}

/// Record with a list of strings
pub type WithStringList {
  names: List<ByteArray>,
}

/// Enum type with multiple constructors
pub type Status {
  Active
  Inactive
  Pending { reason: ByteArray }
}

/// Record containing another record (nested)
pub type Nested {
  inner: SimpleString,
  count: Int,
}

/// Record containing an enum
pub type WithEnum {
  id: ByteArray,
  status: Status,
}

/// Record with optional nested type
pub type WithOptionalNested {
  data: Option<SimpleString>,
}

/// Record with list of records
pub type WithRecordList {
  entries: List<SimpleString>,
}

/// Master type that references all other types to ensure they appear in blueprint
pub type AllTypes {
  simple_string: SimpleString,
  simple_int: SimpleInt,
  multiple_fields: MultipleFields,
  with_option: WithOption,
  with_list: WithList,
  with_string_list: WithStringList,
  status: Status,
  nested: Nested,
  with_enum: WithEnum,
  with_optional_nested: WithOptionalNested,
  with_record_list: WithRecordList,
}

/// A dummy validator that accepts AllTypes as redeemer
/// We don't care about validation logic, just type definitions
validator test_types {
  spend(_datum: Option<Data>, _redeemer: AllTypes, _utxo: OutputReference, _self: Transaction) {
    True
  }
}
