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

// OpenAPIProvider is initialised once per Terraform run. The OpenAPI spec is
// loaded from OPENAPI_SPEC at New() time so that Resources() can return the
// full list of dynamically discovered resource types before Configure() is called.
type OpenAPIProvider struct {
	version string
	prefix  string // resource type name prefix, e.g. "openapi" produces openapi_<name>
	specs   []*spec.ResourceSpec
	loadErr string // non-empty: Configure() surfaces it as a diagnostic error
}

type OpenAPIProviderModel struct {
	URL      types.String `tfsdk:"url"`
	Token    types.String `tfsdk:"token"`
	Insecure types.Bool   `tfsdk:"insecure"`
	Prefix   types.String `tfsdk:"prefix"`
	OKLevel  types.String `tfsdk:"ok_api_calls_log_level"`
	KOLevel  types.String `tfsdk:"ko_api_calls_log_level"`
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

// Configure reads the provider HCL config and environment variables, then creates the shared Client.
func (self *OpenAPIProvider) Configure(
	ctx context.Context,
	req provider.ConfigureRequest,
	resp *provider.ConfigureResponse,
) {
	if self.loadErr != "" {
		resp.Diagnostics.AddError("Missing OpenAPI Spec", self.loadErr)
		return
	}

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

	url := os.Getenv("OPENAPI_URL")
	if !config.URL.IsNull() && !config.URL.IsUnknown() {
		url = config.URL.ValueString()
	}
	if url == "" {
		resp.Diagnostics.AddAttributeError(
			fwpath.Root("url"),
			"Missing OpenAPI API URL",
			"Set the url in the provider configuration or use OPENAPI_URL and ensure its not empty.")
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
		tfSchema, attrTypes := buildSchema(s.Fields)
		specCopy := s
		schemaCopy := tfSchema
		typesCopy := attrTypes
		prefix := self.prefix
		factories = append(factories, func() resource.Resource {
			return &DynamicResource{
				spec:      specCopy,
				tfSchema:  schemaCopy,
				attrTypes: typesCopy,
				prefix:    prefix,
			}
		})
	}
	return factories
}

// DataSources returns one data source factory per discovered spec that has a list path.
func (self *OpenAPIProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	factories := make([]func() datasource.DataSource, 0, len(self.specs))
	for _, s := range self.specs {
		if s.ListPath == "" {
			continue
		}
		_, attrTypes := buildSchema(s.Fields)
		specCopy := s
		typesCopy := attrTypes
		prefix := self.prefix
		factories = append(factories, func() datasource.DataSource {
			return &DynamicDataSource{
				spec:      specCopy,
				prefix:    prefix,
				attrTypes: typesCopy,
			}
		})
	}
	return factories
}

// New returns the provider factory used by providerserver.Serve. The OpenAPI
// spec is loaded immediately from OPENAPI_SPEC so that Resources() is populated
// before Terraform calls Configure().
func New(version string) func() provider.Provider {
	return func() provider.Provider {
		p := &OpenAPIProvider{
			version: version,
			prefix:  envOr("OPENAPI_PREFIX", "openapi"),
		}

		specSource := os.Getenv("OPENAPI_SPEC")
		if specSource == "" {
			p.loadErr = "OPENAPI_SPEC is not set. " +
				"Export it to the path or URL of your OpenAPI 3 spec before running Terraform."
			return p
		}

		model, err := spec.LoadModel(specSource)
		if err != nil {
			p.loadErr = fmt.Sprintf("Unable to load spec from %s, got error: %s", specSource, err)
			return p
		}

		p.specs = spec.DiscoverResources(model)
		return p
	}
}

// envOr returns the value of the environment variable key, or fallback if unset or empty.
func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
