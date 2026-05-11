package provider

import (
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// --- resolveTimeout ------------------------------------------------------------------------------

func TestResolveTimeout_defaults_to_20m_when_no_spec_and_null_block(t *testing.T) {
	block := types.ObjectNull(map[string]attr.Type{
		"create": types.StringType,
		"read":   types.StringType,
		"update": types.StringType,
		"delete": types.StringType,
	})
	got := resolveTimeout(block, "create", "")
	if got != 20*time.Minute {
		t.Fatalf("got %v, want 20m", got)
	}
}

func TestResolveTimeout_uses_spec_default_when_block_null(t *testing.T) {
	block := types.ObjectNull(map[string]attr.Type{
		"create": types.StringType,
		"read":   types.StringType,
		"update": types.StringType,
		"delete": types.StringType,
	})
	got := resolveTimeout(block, "create", "30m")
	if got != 30*time.Minute {
		t.Fatalf("got %v, want 30m", got)
	}
}

func TestResolveTimeout_user_value_overrides_spec_default(t *testing.T) {
	block, _ := types.ObjectValue(
		map[string]attr.Type{
			"create": types.StringType,
			"read":   types.StringType,
			"update": types.StringType,
			"delete": types.StringType,
		},
		map[string]attr.Value{
			"create": types.StringValue("60m"),
			"read":   types.StringNull(),
			"update": types.StringNull(),
			"delete": types.StringNull(),
		},
	)
	got := resolveTimeout(block, "create", "30m")
	if got != 60*time.Minute {
		t.Fatalf("got %v, want 60m", got)
	}
}

func TestResolveTimeout_null_op_in_block_falls_back_to_spec(t *testing.T) {
	block, _ := types.ObjectValue(
		map[string]attr.Type{
			"create": types.StringType,
			"read":   types.StringType,
			"update": types.StringType,
			"delete": types.StringType,
		},
		map[string]attr.Value{
			"create": types.StringNull(),
			"read":   types.StringNull(),
			"update": types.StringNull(),
			"delete": types.StringValue("10m"),
		},
	)
	// "create" is null in block; fall back to spec default
	got := resolveTimeout(block, "create", "30m")
	if got != 30*time.Minute {
		t.Fatalf("got %v, want 30m", got)
	}
	// "delete" has a user value; use it
	got = resolveTimeout(block, "delete", "5m")
	if got != 10*time.Minute {
		t.Fatalf("got %v, want 10m", got)
	}
}

func TestResolveTimeout_invalid_duration_falls_back(t *testing.T) {
	block, _ := types.ObjectValue(
		map[string]attr.Type{
			"create": types.StringType,
			"read":   types.StringType,
			"update": types.StringType,
			"delete": types.StringType,
		},
		map[string]attr.Value{
			"create": types.StringValue("not-a-duration"),
			"read":   types.StringNull(),
			"update": types.StringNull(),
			"delete": types.StringNull(),
		},
	)
	got := resolveTimeout(block, "create", "30m")
	if got != 30*time.Minute {
		t.Fatalf("got %v, want 30m (fallback on bad user value)", got)
	}
}

func TestResolveTimeout_zero_spec_default_uses_fallback(t *testing.T) {
	block := types.ObjectNull(map[string]attr.Type{
		"create": types.StringType,
		"read":   types.StringType,
		"update": types.StringType,
		"delete": types.StringType,
	})
	got := resolveTimeout(block, "create", "0s")
	if got != 20*time.Minute {
		t.Fatalf("got %v, want 20m fallback for zero spec default", got)
	}
}

func TestResolveTimeout_negative_spec_default_uses_fallback(t *testing.T) {
	block := types.ObjectNull(map[string]attr.Type{
		"create": types.StringType,
		"read":   types.StringType,
		"update": types.StringType,
		"delete": types.StringType,
	})
	got := resolveTimeout(block, "create", "-1m")
	if got != 20*time.Minute {
		t.Fatalf("got %v, want 20m fallback for negative spec default", got)
	}
}

// --- extractTimeoutsBlock ------------------------------------------------------------------------

func TestExtractTimeoutsBlock_present(t *testing.T) {
	timeoutsType := types.ObjectType{AttrTypes: map[string]attr.Type{
		"create": types.StringType,
		"read":   types.StringType,
		"update": types.StringType,
		"delete": types.StringType,
	}}
	expected, _ := types.ObjectValue(timeoutsType.AttrTypes, map[string]attr.Value{
		"create": types.StringValue("30m"),
		"read":   types.StringNull(),
		"update": types.StringNull(),
		"delete": types.StringNull(),
	})
	attrs := map[string]attr.Value{"timeouts": expected}
	got := extractTimeoutsBlock(attrs, timeoutsType)
	if got.IsNull() {
		t.Fatal("expected non-null block")
	}
	if s, ok := got.Attributes()["create"].(types.String); !ok || s.ValueString() != "30m" {
		t.Fatalf("create attribute: got %v", got.Attributes()["create"])
	}
}

func TestExtractTimeoutsBlock_absent_returns_null(t *testing.T) {
	timeoutsType := types.ObjectType{AttrTypes: map[string]attr.Type{
		"create": types.StringType,
		"read":   types.StringType,
		"update": types.StringType,
		"delete": types.StringType,
	}}
	got := extractTimeoutsBlock(map[string]attr.Value{}, timeoutsType)
	if !got.IsNull() {
		t.Fatal("expected null block when key absent")
	}
}
