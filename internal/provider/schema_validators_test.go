package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// --- positiveDuration ----------------------------------------------------------------------------

func validateDuration(t *testing.T, val string) (hasError bool) {
	t.Helper()
	req := validator.StringRequest{ConfigValue: types.StringValue(val)}
	var resp validator.StringResponse
	positiveDuration{}.ValidateString(context.Background(), req, &resp)
	return resp.Diagnostics.HasError()
}

func TestPositiveDuration_valid(t *testing.T) {
	for _, v := range []string{"1s", "30m", "1h", "2h30m"} {
		if validateDuration(t, v) {
			t.Errorf("%q: expected valid, got error", v)
		}
	}
}

func TestPositiveDuration_zero_rejected(t *testing.T) {
	if !validateDuration(t, "0s") {
		t.Error("0s: expected error, got none")
	}
}

func TestPositiveDuration_negative_rejected(t *testing.T) {
	if !validateDuration(t, "-1m") {
		t.Error("-1m: expected error, got none")
	}
}

func TestPositiveDuration_invalid_string_rejected(t *testing.T) {
	if !validateDuration(t, "not-a-duration") {
		t.Error("not-a-duration: expected error, got none")
	}
}

func TestPositiveDuration_null_skipped(t *testing.T) {
	req := validator.StringRequest{ConfigValue: types.StringNull()}
	var resp validator.StringResponse
	positiveDuration{}.ValidateString(context.Background(), req, &resp)
	if resp.Diagnostics.HasError() {
		t.Error("null value: expected no error")
	}
}
