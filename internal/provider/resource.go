package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	tfschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/republique-et-canton-de-geneve/terraform-provider-openapi/internal/spec"
)

// DynamicResource defines the resource implementation.
type DynamicResource struct {
	spec      *spec.ResourceSpec
	tfSchema  tfschema.Schema
	attrTypes map[string]attr.Type
	prefix    string
	client    *Client
}

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &DynamicResource{}
var _ resource.ResourceWithConfigure = &DynamicResource{}
var _ resource.ResourceWithImportState = &DynamicResource{}

// Metadata sets the resource type name to <prefix>_<resource-name>.
func (self *DynamicResource) Metadata(
	ctx context.Context,
	req resource.MetadataRequest,
	resp *resource.MetadataResponse,
) {
	resp.TypeName = self.prefix + "_" + self.spec.SingularName
}

// Schema returns the Terraform schema built from the OAS3 item schema at startup.
func (self *DynamicResource) Schema(
	ctx context.Context,
	_ resource.SchemaRequest,
	resp *resource.SchemaResponse,
) {
	resp.Schema = self.tfSchema
}

// Configure receives the shared Client from the provider and stores it for CRUD use.
func (self *DynamicResource) Configure(
	ctx context.Context,
	req resource.ConfigureRequest,
	resp *resource.ConfigureResponse,
) {
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(*Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf(
				"Expected *Client, got: %T. Please report this issue to the provider developers.",
				req.ProviderData))
		return
	}
	self.client = client
}

// Create POSTs the plan to the collection endpoint and stores the API response as state.
func (self *DynamicResource) Create(
	ctx context.Context,
	req resource.CreateRequest,
	resp *resource.CreateResponse,
) {
	// Read Terraform plan data into the model.
	var plan types.Object
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := attrMapToJSON(plan.Attributes(), self.spec.Fields)
	for _, f := range self.spec.Fields {
		if f.IsID && !f.Writable {
			delete(body, f.OASName) // server assigns the ID, strip it from the POST body
			break
		}
	}

	raw, err := self.client.Create(ctx, self.spec.ListPath, body)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to create "+self.spec.SingularName,
			fmt.Sprintf("Unable to create %s, got error: %s", self.spec.SingularName, err))
		return
	}

	tflog.Debug(ctx, fmt.Sprintf("Created %s successfully", self.spec.SingularName),
		map[string]any{"id": raw[self.spec.IDField]})

	// Save created resource into Terraform state.
	state, diags := jsonToObject(raw, self.spec.Fields, self.attrTypes)
	resp.Diagnostics.Append(diags...)
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

// Read GETs the item endpoint and refreshes state; removes the resource if it returns 404.
func (self *DynamicResource) Read(
	ctx context.Context,
	req resource.ReadRequest,
	resp *resource.ReadResponse,
) {
	// Read Terraform prior state data into the model.
	var state types.Object
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := extractID(state, self.spec.IDField)
	if id == "" {
		resp.Diagnostics.AddError(
			"Missing ID",
			fmt.Sprintf("Unable to read %s, id is empty", self.spec.SingularName))
		return
	}

	raw, found, err := self.client.Read(ctx, self.spec.ResolvedItemPath(id))
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to read "+self.spec.SingularName,
			fmt.Sprintf("Unable to read %s %s, got error: %s", self.spec.SingularName, id, err))
		return
	}
	if !found {
		resp.State.RemoveResource(ctx)
		return
	}

	// Save updated resource into Terraform state.
	newState, diags := jsonToObject(raw, self.spec.Fields, self.attrTypes)
	resp.Diagnostics.Append(diags...)
	resp.Diagnostics.Append(resp.State.Set(ctx, newState)...)
}

// Update sends the plan to the item endpoint using PUT or PATCH and refreshes state.
func (self *DynamicResource) Update(
	ctx context.Context,
	req resource.UpdateRequest,
	resp *resource.UpdateResponse,
) {
	// Read Terraform plan data into the model.
	var plan types.Object
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state types.Object
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := extractID(state, self.spec.IDField)
	if id == "" {
		resp.Diagnostics.AddError(
			"Missing ID",
			fmt.Sprintf("Unable to update %s, id is empty", self.spec.SingularName))
		return
	}

	body := attrMapToJSON(plan.Attributes(), self.spec.Fields)
	for _, f := range self.spec.Fields {
		if f.IsID {
			delete(body, f.OASName) // never send the ID in an update body
			break
		}
	}

	method := self.spec.UpdateMethod
	if method == "" {
		method = "PATCH"
	}

	raw, err := self.client.Update(ctx, self.spec.ResolvedItemPath(id), method, body)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to update "+self.spec.SingularName,
			fmt.Sprintf("Unable to update %s %s, got error: %s", self.spec.SingularName, id, err))
		return
	}

	tflog.Debug(ctx, fmt.Sprintf("Updated %s %s successfully", self.spec.SingularName, id))

	// Save updated resource into Terraform state.
	newState, diags := jsonToObject(raw, self.spec.Fields, self.attrTypes)
	resp.Diagnostics.Append(diags...)
	resp.Diagnostics.Append(resp.State.Set(ctx, newState)...)
}

// Delete sends a DELETE to the item endpoint.
func (self *DynamicResource) Delete(
	ctx context.Context,
	req resource.DeleteRequest,
	resp *resource.DeleteResponse,
) {
	// Read Terraform prior state data into the model.
	var state types.Object
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := extractID(state, self.spec.IDField)
	if id == "" {
		resp.Diagnostics.AddError(
			"Missing ID",
			fmt.Sprintf("Unable to delete %s, id is empty", self.spec.SingularName))
		return
	}

	if err := self.client.Delete(ctx, self.spec.ResolvedItemPath(id)); err != nil {
		resp.Diagnostics.AddError(
			"Unable to delete "+self.spec.SingularName,
			fmt.Sprintf("Unable to delete %s %s, got error: %s", self.spec.SingularName, id, err))
	}
}

// ImportState passes the import ID directly to the ID field, then Read fetches the rest.
func (self *DynamicResource) ImportState(
	ctx context.Context,
	req resource.ImportStateRequest,
	resp *resource.ImportStateResponse,
) {
	resource.ImportStatePassthroughID(ctx, path.Root(self.spec.IDField), req, resp)
}
