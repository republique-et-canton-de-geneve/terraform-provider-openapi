package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/republique-et-canton-de-geneve/terraform-provider-openapi/internal/spec"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &DynamicDataSource{}
var _ datasource.DataSourceWithConfigure = &DynamicDataSource{}

// DynamicDataSource defines the data source implementation.
type DynamicDataSource struct {
	spec      *spec.ResourceSpec
	prefix    string
	attrTypes map[string]attr.Type
	client    *Client
}

type listState struct {
	Items types.List `tfsdk:"items"`
}

// Metadata sets the data source type name to <prefix>_<resource-name>.
func (self *DynamicDataSource) Metadata(
	ctx context.Context,
	req datasource.MetadataRequest,
	resp *datasource.MetadataResponse,
) {
	resp.TypeName = self.prefix + "_" + self.spec.Name
}

// Schema returns a single computed "items" list containing one object per API item.
func (self *DynamicDataSource) Schema(
	ctx context.Context,
	_ datasource.SchemaRequest,
	resp *datasource.SchemaResponse,
) {
	resp.Schema = buildDataSourceSchema(self.spec.Fields)
}

// Configure receives the shared Client from the provider.
func (self *DynamicDataSource) Configure(
	ctx context.Context,
	req datasource.ConfigureRequest,
	resp *datasource.ConfigureResponse,
) {
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(*Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf(
				"Expected *Client, got: %T. Please report this issue to the provider developers.",
				req.ProviderData))
		return
	}
	self.client = client
}

// Read GETs the collection endpoint and saves the items into Terraform state.
func (self *DynamicDataSource) Read(
	ctx context.Context,
	_ datasource.ReadRequest,
	resp *datasource.ReadResponse,
) {
	if self.spec.ListPath == "" {
		resp.Diagnostics.AddError(
			"No Collection Endpoint",
			fmt.Sprintf(
				"Resource %q has no list path and cannot be used as a data source.",
				self.spec.Name))
		return
	}

	raw, err := self.client.List(ctx, self.spec.ListPath)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to list "+self.spec.Name,
			fmt.Sprintf("Unable to list %s, got error: %s", self.spec.Name, err))
		return
	}

	itemsType := types.ObjectType{AttrTypes: self.attrTypes}
	elems := make([]attr.Value, 0, len(raw))
	for _, item := range raw {
		obj, diags := jsonToObject(item, self.spec.Fields, self.attrTypes)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		elems = append(elems, obj)
	}

	list, diags := types.ListValue(itemsType, elems)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Save updated items into Terraform state.
	resp.Diagnostics.Append(resp.State.Set(ctx, &listState{Items: list})...)
}
