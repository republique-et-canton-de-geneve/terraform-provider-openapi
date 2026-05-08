package provider

import (
	"testing"

	"github.com/republique-et-canton-de-geneve/terraform-provider-openapi/internal/spec"
)

// --- collectUntypedFields ------------------------------------------------------------------------

func TestCollectUntypedFields(t *testing.T) {
	cases := []struct {
		name   string
		fields []*spec.FieldSpec
		want   []string
	}{
		{
			name:   "no untyped fields",
			fields: []*spec.FieldSpec{
				{Name: "id", Type: "string"},
				{Name: "count", Type: "integer"},
			},
			want:   nil,
		},
		{
			name:   "top-level untyped",
			fields: []*spec.FieldSpec{{Name: "payload", Type: "untyped"}},
			want:   []string{"payload"},
		},
		{
			name: "nested untyped inside object",
			fields: []*spec.FieldSpec{{
				Name: "meta",
				Type: "object",
				Nested: []*spec.FieldSpec{
					{Name: "id", Type: "string"},
					{Name: "extra", Type: "untyped"},
				},
			}},
			want: []string{"meta.extra"},
		},
		{
			name: "untyped array item",
			fields: []*spec.FieldSpec{{
				Name:     "tags",
				Type:     "array",
				ItemSpec: &spec.FieldSpec{Name: "item", Type: "untyped"},
			}},
			want: []string{"tags[].item"},
		},
		{
			name: "mixed top-level and nested",
			fields: []*spec.FieldSpec{
				{Name: "payload", Type: "untyped"},
				{
					Name: "meta",
					Type: "object",
					Nested: []*spec.FieldSpec{
						{Name: "extra", Type: "untyped"},
					},
				},
			},
			want: []string{"payload", "meta.extra"},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := collectUntypedFields(c.fields, "")
			if len(got) != len(c.want) {
				t.Fatalf("got %v, want %v", got, c.want)
			}
			for i, v := range got {
				if v != c.want[i] {
					t.Errorf("[%d] got %q, want %q", i, v, c.want[i])
				}
			}
		})
	}
}
