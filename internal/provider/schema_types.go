package provider

// UntypedFieldMode controls how OAS fields with no declared type are exposed in the schema.
type UntypedFieldMode string

const (
	UntypedFieldModeJSON  UntypedFieldMode = "json"
	UntypedFieldModeError UntypedFieldMode = "error"
)
