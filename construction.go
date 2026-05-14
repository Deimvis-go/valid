package valid

// TODO: implement xencoding/xreflectenc.StructConstruction
// TODO: add to default validator support for tag that checks construction
//       and ensures field was set or set by default (e.g. `validate:"construction=notdefault,omitnoinfo"`)
//       Also add function like RegisterConstructionTags(validator) to allow users
//       to register this tag handler on their own validator.

type StructConstructionInfo interface {
	// Field returns construction info for struct field.
	// If no info for given field, Field returns false.
	// Value for index argument can be obtained
	// using reflect.Type.Field(i).Index.
	Field(index []int) (StructFieldConstructionInfo, bool)
}

type StructFieldConstructionInfo interface {
	IsDefault() bool
}
