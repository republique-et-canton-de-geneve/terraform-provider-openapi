package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	dsschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/republique-et-canton-de-geneve/terraform-provider-openapi/internal/spec"
)

var _ datasource.DataSource = &ManifestDataSource{}

// ManifestDataSource is a built-in data source that exposes every resource and data source the
// provider discovered from the OpenAPI spec, so users can introspect available types from within
// Terraform itself.
type ManifestDataSource struct {
	specs  []*spec.ResourceSpec
	prefix string
}

type manifestState struct {
	Resources []manifestEntry `tfsdk:"resources"`
}

type manifestEntry struct {
	ResourceType   types.String `tfsdk:"resource_type"`
	DatasourceType types.String `tfsdk:"datasource_type"`
	ItemPath       types.String `tfsdk:"item_path"`
	ListPath       types.String `tfsdk:"list_path"`
	CanCreate      types.Bool   `tfsdk:"can_create"`
	CanUpdate      types.Bool   `tfsdk:"can_update"`
	CanDelete      types.Bool   `tfsdk:"can_delete"`
}

// Metadata sets the data source type name to <prefix>_manifest.
func (self *ManifestDataSource) Metadata(
	_ context.Context,
	req datasource.MetadataRequest,
	resp *datasource.MetadataResponse,
) {
	resp.TypeName = self.prefix + "_manifest"
}

// Schema declares the manifest data source attributes.
func (self *ManifestDataSource) Schema(
	_ context.Context,
	_ datasource.SchemaRequest,
	resp *datasource.SchemaResponse,
) {
	entry := dsschema.NestedAttributeObject{
		Attributes: map[string]dsschema.Attribute{
			"resource_type": dsschema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Terraform resource type name, e.g. `openapi_vlan`.",
			},
			"datasource_type": dsschema.StringAttribute{
				Computed: true,
				MarkdownDescription: "Terraform data source type name, " +
					"empty if the resource has no list endpoint.",
			},
			"item_path": dsschema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "OpenAPI item path template, e.g. `/api/v1/vlans/{id}/`.",
			},
			"list_path": dsschema.StringAttribute{
				Computed: true,
				MarkdownDescription: "OpenAPI collection path, " +
					"empty if the resource has no list endpoint.",
			},
			"can_create": dsschema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether the resource supports creation (POST).",
			},
			"can_update": dsschema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether the resource supports updates (PATCH/PUT).",
			},
			"can_delete": dsschema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether the resource supports deletion (DELETE).",
			},
		},
	}
	resp.Schema = dsschema.Schema{
		MarkdownDescription: fmt.Sprintf(
			"Built-in data source that lists every resource and data source discovered from the "+
				"OpenAPI spec. "+
				"Use it to introspect available types without reading the spec directly.\n\n"+
				"```hcl\ndata \"%s_manifest\" \"all\" {}\n```", self.prefix),
		Attributes: map[string]dsschema.Attribute{
			"resources": dsschema.ListNestedAttribute{
				Computed:            true,
				NestedObject:        entry,
				MarkdownDescription: "One entry per discovered resource.",
			},
		},
	}
}

// Read populates the manifest from the in-memory spec list — no HTTP call needed.
func (self *ManifestDataSource) Read(
	ctx context.Context,
	_ datasource.ReadRequest,
	resp *datasource.ReadResponse,
) {
	entries := make([]manifestEntry, 0, len(self.specs))
	for _, s := range self.specs {
		dsType := ""
		listPath := ""
		if s.ListPath != "" {
			dsType = self.prefix + "_" + s.PluralName
			listPath = s.ListPath
		}
		entries = append(entries, manifestEntry{
			ResourceType:   types.StringValue(self.prefix + "_" + s.SingularName),
			DatasourceType: types.StringValue(dsType),
			ItemPath:       types.StringValue(s.ItemPath),
			ListPath:       types.StringValue(listPath),
			CanCreate:      types.BoolValue(s.HasCreate),
			CanUpdate:      types.BoolValue(s.HasUpdate),
			CanDelete:      types.BoolValue(s.HasDelete),
		})
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &manifestState{Resources: entries})...)
}
