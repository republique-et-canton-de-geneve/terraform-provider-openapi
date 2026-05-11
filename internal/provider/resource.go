package provider

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	tfschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/republique-et-canton-de-geneve/terraform-provider-openapi/internal/spec"
)

// DynamicResource defines the resource implementation.
type DynamicResource struct {
	spec         *spec.ResourceSpec
	tfSchema     tfschema.Schema
	attrTypes    map[string]attr.Type
	timeoutsType types.ObjectType
	prefix       string
	client       *Client
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
	var plan types.Object
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	timeoutsBlock := extractTimeoutsBlock(plan.Attributes(), self.timeoutsType)
	timeout := resolveTimeout(timeoutsBlock, "create", self.spec.Timeouts.Create)
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

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

	state, diags := self.buildStateWithTimeouts(raw, timeoutsBlock)
	resp.Diagnostics.Append(diags...)
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

// Read GETs the item endpoint and refreshes state; removes the resource if it returns 404.
func (self *DynamicResource) Read(
	ctx context.Context,
	req resource.ReadRequest,
	resp *resource.ReadResponse,
) {
	var state types.Object
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	timeoutsBlock := extractTimeoutsBlock(state.Attributes(), self.timeoutsType)
	timeout := resolveTimeout(timeoutsBlock, "read", self.spec.Timeouts.Read)
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	id := extractID(state, self.spec.IDField)
	if id == "" {
		resp.Diagnostics.AddError(
			"Missing ID",
			fmt.Sprintf("Unable to read %s, id is empty", self.spec.SingularName))
		return
	}

	raw, found, err := self.client.Read(ctx, self.spec.ResolvedItemPath(id), timeout)
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

	newState, diags := self.buildStateWithTimeouts(raw, timeoutsBlock)
	resp.Diagnostics.Append(diags...)
	resp.Diagnostics.Append(resp.State.Set(ctx, newState)...)
}

// Update sends the plan to the item endpoint using PUT or PATCH and refreshes state.
func (self *DynamicResource) Update(
	ctx context.Context,
	req resource.UpdateRequest,
	resp *resource.UpdateResponse,
) {
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

	timeoutsBlock := extractTimeoutsBlock(plan.Attributes(), self.timeoutsType)
	timeout := resolveTimeout(timeoutsBlock, "update", self.spec.Timeouts.Update)
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

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

	newState, diags := self.buildStateWithTimeouts(raw, timeoutsBlock)
	resp.Diagnostics.Append(diags...)
	resp.Diagnostics.Append(resp.State.Set(ctx, newState)...)
}

// Delete sends a DELETE to the item endpoint.
func (self *DynamicResource) Delete(
	ctx context.Context,
	req resource.DeleteRequest,
	resp *resource.DeleteResponse,
) {
	var state types.Object
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	timeoutsBlock := extractTimeoutsBlock(state.Attributes(), self.timeoutsType)
	deleteTimeout := resolveTimeout(timeoutsBlock, "delete", self.spec.Timeouts.Delete)
	readTimeout := resolveTimeout(timeoutsBlock, "read", self.spec.Timeouts.Read)
	ctx, cancel := context.WithTimeout(ctx, deleteTimeout)
	defer cancel()

	id := extractID(state, self.spec.IDField)
	if id == "" {
		resp.Diagnostics.AddError(
			"Missing ID",
			fmt.Sprintf("Unable to delete %s, id is empty", self.spec.SingularName))
		return
	}

	if err := self.client.Delete(ctx, self.spec.ResolvedItemPath(id), readTimeout); err != nil {
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

// buildStateWithTimeouts builds a types.Object combining the API response fields and the
// timeouts block so both are persisted to Terraform state.
func (self *DynamicResource) buildStateWithTimeouts(
	raw map[string]any,
	timeoutsBlock types.Object,
) (types.Object, diag.Diagnostics) {
	resState, diags := jsonToObject(raw, self.spec.Fields, self.attrTypes)
	if diags.HasError() {
		return types.ObjectNull(nil), diags
	}

	allAttrTypes := make(map[string]attr.Type, len(self.attrTypes)+1)
	for k, v := range self.attrTypes {
		allAttrTypes[k] = v
	}
	allAttrTypes["timeouts"] = self.timeoutsType

	allAttrs := make(map[string]attr.Value, len(resState.Attributes())+1)
	for k, v := range resState.Attributes() {
		allAttrs[k] = v
	}
	allAttrs["timeouts"] = timeoutsBlock

	return types.ObjectValue(allAttrTypes, allAttrs)
}

// extractTimeoutsBlock retrieves the "timeouts" key from a plan/state attribute map as a
// types.Object. Returns a null object (with correct type) when the block is absent.
func extractTimeoutsBlock(attrs map[string]attr.Value, timeoutsType types.ObjectType) types.Object {
	if v, ok := attrs["timeouts"]; ok {
		if obj, ok := v.(types.Object); ok {
			return obj
		}
	}
	return types.ObjectNull(timeoutsType.AttrTypes)
}

// resolveTimeout returns the effective timeout for one operation.
// Priority: user-configured value in the timeouts block > x-timeout spec default > 5m (Aria default).
func resolveTimeout(block types.Object, op string, specDefault string) time.Duration {
	const fallbackDur = 20 * time.Minute // Terraform's standard default resource timeout

	base := fallbackDur
	if specDefault != "" {
		if d, err := time.ParseDuration(specDefault); err == nil && d > 0 {
			base = d
		}
	}

	if block.IsNull() || block.IsUnknown() {
		return base
	}

	if v, ok := block.Attributes()[op]; ok {
		if s, ok := v.(types.String); ok && !s.IsNull() && !s.IsUnknown() {
			if d, err := time.ParseDuration(s.ValueString()); err == nil {
				return d
			}
		}
	}
	return base
}
