package valid

// TODO: implement a reflection-based StructConstruction helper that records
//       which fields of a struct were explicitly set vs. left at their
//       default values.
// TODO: add default validator support for a tag that checks construction
//       state, e.g. `validate:"construction=notdefault,omitnoinfo"`, so
//       callers can require a field to have been set (or to have been set
//       to something other than the zero value). Provide a registration
//       helper — for example RegisterConstructionTags(validator) — so
//       users can install this tag handler on their own validator.

// StructConstructionInfo exposes, for a given struct, which of its fields
// were explicitly set during construction and which are still at their
// zero value.
type StructConstructionInfo interface {
	// Field returns the construction info for a struct field identified by
	// its reflect.Type.Field(i).Index path. It returns false when no info
	// is available for the given field.
	Field(index []int) (StructFieldConstructionInfo, bool)
}

// StructFieldConstructionInfo reports construction facts about a single
// struct field.
type StructFieldConstructionInfo interface {
	IsDefault() bool
}
