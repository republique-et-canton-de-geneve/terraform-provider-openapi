package provider

import (
	"context"
	"math"
	"regexp"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/republique-et-canton-de-geneve/terraform-provider-openapi/internal/spec"
)

// stringValidators builds string validators from a FieldSpec's constraints.
func stringValidators(f *spec.FieldSpec) []validator.String {
	var vals []validator.String
	if f.MinLength != nil {
		vals = append(vals, stringvalidator.LengthAtLeast(int(*f.MinLength)))
	}
	if f.MaxLength != nil {
		vals = append(vals, stringvalidator.LengthAtMost(int(*f.MaxLength)))
	}
	if f.Pattern != "" {
		vals = append(vals, stringvalidator.RegexMatches(regexp.MustCompile(f.Pattern), ""))
	}
	if len(f.Enum) > 0 {
		vals = append(vals, stringvalidator.OneOf(f.Enum...))
	}
	return vals
}

// positiveDuration validates that a string is a parseable time.Duration greater than zero.
type positiveDuration struct{}

func (positiveDuration) Description(_ context.Context) string {
	return "must be a valid duration greater than zero (e.g. \"30m\", \"1h\")"
}

func (positiveDuration) MarkdownDescription(ctx context.Context) string {
	return positiveDuration{}.Description(ctx)
}

func (positiveDuration) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	d, err := time.ParseDuration(req.ConfigValue.ValueString())
	if err != nil {
		resp.Diagnostics.AddAttributeError(req.Path, "Invalid duration", positiveDuration{}.Description(context.Background()))
		return
	}
	if d <= 0 {
		resp.Diagnostics.AddAttributeError(req.Path, "Invalid duration", "timeout must be greater than zero")
	}
}

// int64Validators builds int64 validators from a FieldSpec's constraints.
func int64Validators(f *spec.FieldSpec) []validator.Int64 {
	var vals []validator.Int64
	if f.Minimum != nil && *f.Minimum > math.MinInt64 {
		vals = append(vals, int64validator.AtLeast(int64(*f.Minimum)))
	}
	if f.Maximum != nil && *f.Maximum < math.MaxInt64 {
		vals = append(vals, int64validator.AtMost(int64(*f.Maximum)))
	}
	return vals
}
