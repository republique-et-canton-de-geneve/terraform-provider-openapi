// Package provider implements the OpenAPI dynamic Terraform provider.
package provider

import (
	"context"
	"fmt"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	fwpath "github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	pfschema "github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/republique-et-canton-de-geneve/terraform-provider-openapi/internal/spec"
)

// Ensure OpenAPIProvider satisfies various provider interfaces.
var _ provider.Provider = &OpenAPIProvider{}

// OpenAPIProvider is initialised once per Terraform run. The OpenAPI spec is loaded from
// OPENAPI_SPEC at New() time so that Resources() can return the full list of dynamically
// discovered resource types before Configure() is called.
type OpenAPIProvider struct {
	version      string
	prefix       string           // resource type name prefix, e.g. "openapi" produces openapi_<name>
	untypedMode  UntypedFieldMode // how untyped OAS fields are exposed
	specs        []*spec.ResourceSpec
}

type OpenAPIProviderModel struct {
	URL           types.String `tfsdk:"url"`
	Token         types.String `tfsdk:"token"`
	Insecure      types.Bool   `tfsdk:"insecure"`
	Prefix        types.String `tfsdk:"prefix"`
	OKLevel       types.String `tfsdk:"ok_api_calls_log_level"`
	KOLevel       types.String `tfsdk:"ko_api_calls_log_level"`
	UntypedMode types.String `tfsdk:"untyped_mode"`
}

// Metadata sets the provider type name.
func (self *OpenAPIProvider) Metadata(
	ctx context.Context,
	_ provider.MetadataRequest,
	resp *provider.MetadataResponse,
) {
	resp.TypeName = "openapi"
	resp.Version = self.version
}

// Schema declares the provider-level HCL configuration attributes.
func (self *OpenAPIProvider) Schema(
	ctx context.Context,
	_ provider.SchemaRequest,
	resp *provider.SchemaResponse,
) {
	resp.Schema = pfschema.Schema{
		MarkdownDescription: "Dynamic Terraform provider driven by an OpenAPI 3 specification. " +
			"Point `OPENAPI_SPEC` at the spec file or URL before running Terraform.",
		Attributes: map[string]pfschema.Attribute{
			"url": pfschema.StringAttribute{
				MarkdownDescription: "Base URL of the API (e.g. `https://api.example.com/v1`). " +
					"May also be provided via OPENAPI_URL environment variable.",
				Optional: true,
			},
			"token": pfschema.StringAttribute{
				MarkdownDescription: "Bearer token for API authentication. " +
					"May also be provided via OPENAPI_TOKEN environment variable.",
				Optional:  true,
				Sensitive: true,
			},
			"insecure": pfschema.BoolAttribute{
				MarkdownDescription: "Skip TLS certificate verification. " +
					"May also be provided via OPENAPI_INSECURE environment variable.",
				Optional: true,
			},
			"ok_api_calls_log_level": pfschema.StringAttribute{
				MarkdownDescription: "Successful API calls log level. " +
					"One of `TRACE` (default), `DEBUG` or `INFO`. " +
					"May also be provided via OPENAPI_OK_API_CALLS_LOG_LEVEL environment variable.",
				Optional: true,
			},
			"ko_api_calls_log_level": pfschema.StringAttribute{
				MarkdownDescription: "Failed API calls log level. " +
					"One of `ERROR` (default), `WARN`, `DEBUG` or `TRACE`. " +
					"May also be provided via OPENAPI_KO_API_CALLS_LOG_LEVEL environment variable.",
				Optional: true,
			},
			"untyped_mode": pfschema.StringAttribute{
				MarkdownDescription: "How OAS fields with no declared type are exposed. " +
					"One of `json` (default) or `error`. " +
					"May also be provided via the OPENAPI_UNTYPED_MODE environment variable.",
				Optional: true,
			},
			"prefix": pfschema.StringAttribute{
				MarkdownDescription: "Prefix for all resource type names (default `openapi`). " +
					"For example, prefix `openapi` produces `openapi_vlans`.\n\n" +
					"~> **Setting this explicitly is strongly recommended.** " +
					"Resource type names are registered from OPENAPI_PREFIX before the provider " +
					"block is read, so the env var is the source of truth. " +
					"Declaring `prefix` here lets Terraform detect a mismatch early: " +
					"if `prefix` disagrees with OPENAPI_PREFIX, the provider errors at " +
					"configure time instead of silently producing wrong resource type names " +
					"or a broken state.",
				Optional: true,
			},
		},
	}
}

// Configure reads the provider HCL config and env. variables, then creates the shared Client.
func (self *OpenAPIProvider) Configure(
	ctx context.Context,
	req provider.ConfigureRequest,
	resp *provider.ConfigureResponse,
) {
	var config OpenAPIProviderModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if !config.Prefix.IsNull() && !config.Prefix.IsUnknown() {
		if config.Prefix.ValueString() != self.prefix {
			resp.Diagnostics.AddAttributeError(
				fwpath.Root("prefix"),
				"Prefix Mismatch",
				fmt.Sprintf(
					"Resource type names were registered with prefix %q (from OPENAPI_PREFIX). "+
						"Set OPENAPI_PREFIX=%s before running terraform init/plan/apply.",
					self.prefix, config.Prefix.ValueString()))
			return
		}
	}

	if !config.UntypedMode.IsNull() && !config.UntypedMode.IsUnknown() {
		if UntypedFieldMode(config.UntypedMode.ValueString()) != self.untypedMode {
			resp.Diagnostics.AddAttributeError(
				fwpath.Root("untyped_mode"),
				"Untyped Mode Mismatch",
				fmt.Sprintf(
					"Schemas were built with untyped_mode=%q (from OPENAPI_UNTYPED_MODE). "+
						"Set OPENAPI_UNTYPED_MODE=%s before running terraform init/plan/apply.",
					self.untypedMode, config.UntypedMode.ValueString()))
			return
		}
	}

	url := os.Getenv("OPENAPI_URL")
	if !config.URL.IsNull() && !config.URL.IsUnknown() {
		url = config.URL.ValueString()
	}
	if url == "" {
		resp.Diagnostics.AddAttributeError(
			fwpath.Root("url"),
			"Missing OpenAPI API URL",
			"Set the url in the provider configuration or use a non empty OPENAPI_URL.")
		return
	}

	token := os.Getenv("OPENAPI_TOKEN")
	if !config.Token.IsNull() && !config.Token.IsUnknown() {
		token = config.Token.ValueString()
	}

	insecure := os.Getenv("OPENAPI_INSECURE") == "true"
	if !config.Insecure.IsNull() && !config.Insecure.IsUnknown() {
		insecure = config.Insecure.ValueBool()
	}

	okLevel := envOr("OPENAPI_OK_LOG_LEVEL", "TRACE")
	if !config.OKLevel.IsNull() && !config.OKLevel.IsUnknown() {
		okLevel = config.OKLevel.ValueString()
	}

	koLevel := envOr("OPENAPI_KO_LOG_LEVEL", "ERROR")
	if !config.KOLevel.IsNull() && !config.KOLevel.IsUnknown() {
		koLevel = config.KOLevel.ValueString()
	}

	tflog.Debug(ctx, "Creating OpenAPI client", map[string]any{"url": url, "insecure": insecure})

	client, err := NewClient(url, token, insecure, okLevel, koLevel)
	if err != nil {
		resp.Diagnostics.AddError("Unable to create OpenAPI client", err.Error())
		return
	}

	tflog.Debug(ctx, "Configured OpenAPI client")
	resp.ResourceData = client
	resp.DataSourceData = client
}

// Resources returns one resource factory per discovered spec.
func (self *OpenAPIProvider) Resources(ctx context.Context) []func() resource.Resource {
	factories := make([]func() resource.Resource, 0, len(self.specs))
	for _, s := range self.specs {
		tfSchema, attrTypes := buildSchema(s.Fields, self.untypedMode)
		tflog.Debug(
			ctx,
			"registered resource",
			map[string]any{"type": self.prefix + "_" + s.SingularName})
		factories = append(factories, func() resource.Resource {
			return &DynamicResource{
				spec:      s,
				tfSchema:  tfSchema,
				attrTypes: attrTypes,
				prefix:    self.prefix,
			}
		})
	}
	return factories
}

// DataSources returns one data source factory per discovered spec that has a list path,
// plus the built-in manifest data source.
func (self *OpenAPIProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	factories := make([]func() datasource.DataSource, 0, len(self.specs)+1)
	factories = append(factories, func() datasource.DataSource {
		return &ManifestDataSource{specs: self.specs, prefix: self.prefix}
	})
	tflog.Debug(ctx, "registered data source", map[string]any{"type": self.prefix + "_manifest"})
	for _, s := range self.specs {
		if s.ListPath == "" {
			continue
		}
		attrTypes := buildDataSourceAttrTypes(s.Fields, self.untypedMode)
		tflog.Debug(
			ctx,
			"registered data source",
			map[string]any{"type": self.prefix + "_" + s.PluralName})
		factories = append(factories, func() datasource.DataSource {
			return &DynamicDataSource{
				spec:        s,
				prefix:      self.prefix,
				untypedMode: self.untypedMode,
				attrTypes:   attrTypes,
			}
		})
	}
	return factories
}

// New returns the provider factory used by providerserver.Serve. The spec is loaded at call time
// so Resources() is fully populated before Configure() runs. If the spec is missing or unreadable,
// the process exits immediately: there is nothing useful the provider can do without it.
func New(version string) func() provider.Provider {
	prefix := envOr("OPENAPI_PREFIX", "openapi")

	specSource := os.Getenv("OPENAPI_SPEC")
	if specSource == "" {
		fmt.Fprintln(os.Stderr, "error: OPENAPI_SPEC is not set. "+
			"Export it to the path or URL of your OpenAPI 3 spec before running Terraform.")
		os.Exit(1)
	}

	rawMode := envOr("OPENAPI_UNTYPED_MODE", string(UntypedFieldModeJSON))
	untypedMode := UntypedFieldMode(rawMode)
	switch untypedMode {
	case UntypedFieldModeJSON, UntypedFieldModeError:
	default:
		fmt.Fprintf(os.Stderr,
			"error: invalid OPENAPI_UNTYPED_MODE %q, must be json or error\n", rawMode)
		os.Exit(1)
	}

	model, err := spec.LoadModel(specSource)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: unable to load spec from %s: %s\n", specSource, err)
		os.Exit(1)
	}

	specs := spec.DiscoverResources(model)

	if untypedMode == UntypedFieldModeError {
		for _, s := range specs {
			if hasUntypedField(s.Fields) {
				fmt.Fprintf(os.Stderr,
					"error: resource %q has untyped fields and OPENAPI_UNTYPED_MODE=error\n",
					s.SingularName)
				os.Exit(1)
			}
		}
	}

	return func() provider.Provider {
		return &OpenAPIProvider{
			version:     version,
			prefix:      prefix,
			untypedMode: untypedMode,
			specs:       specs,
		}
	}
}

// envOr returns the value of the environment variable key, or fallback if unset or empty.
func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// hasUntypedField reports whether any field in the slice (or any nested field) has type "untyped".
func hasUntypedField(fields []*spec.FieldSpec) bool {
	for _, f := range fields {
		if f.Type == "untyped" {
			return true
		}
		if hasUntypedField(f.Nested) {
			return true
		}
		if f.ItemSpec != nil && hasUntypedField([]*spec.FieldSpec{f.ItemSpec}) {
			return true
		}
	}
	return false
}
