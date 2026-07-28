// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/jhuiting/terraform-provider-chargebee/internal/chargebee"
)

var _ provider.Provider = (*chargebeeProvider)(nil)

// New returns a provider factory for the given build version.
func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &chargebeeProvider{version: version}
	}
}

type chargebeeProvider struct {
	version string
}

type chargebeeProviderModel struct {
	Site   types.String `tfsdk:"site"`
	APIKey types.String `tfsdk:"api_key"`
}

func (p *chargebeeProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "chargebee"
	resp.Version = p.version
}

func (p *chargebeeProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manage a Chargebee Product Catalog 2.0 (item families, items, item prices, attached items, " +
			"features, and item entitlements).",
		Attributes: map[string]schema.Attribute{
			"site": schema.StringAttribute{
				Optional:    true,
				Description: "Chargebee site name (the `<site>` in `<site>.chargebee.com`). Falls back to the `CHARGEBEE_SITE` environment variable.",
			},
			"api_key": schema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				Description: "Chargebee API key. Falls back to the `CHARGEBEE_API_KEY` environment variable.",
			},
		},
	}
}

func (p *chargebeeProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var config chargebeeProviderModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if config.Site.IsUnknown() {
		resp.Diagnostics.AddError(
			"Unknown Chargebee site",
			"The site provider attribute is derived from a value that is not known until apply; set it from a known value or the CHARGEBEE_SITE environment variable.",
		)
	}
	if config.APIKey.IsUnknown() {
		resp.Diagnostics.AddError(
			"Unknown Chargebee API key",
			"The api_key provider attribute is derived from a value that is not known until apply; set it from a known value or the CHARGEBEE_API_KEY environment variable.",
		)
	}
	if resp.Diagnostics.HasError() {
		return
	}

	site := config.Site.ValueString()
	if site == "" {
		site = os.Getenv("CHARGEBEE_SITE")
	}
	apiKey := config.APIKey.ValueString()
	if apiKey == "" {
		apiKey = os.Getenv("CHARGEBEE_API_KEY")
	}

	if site == "" {
		resp.Diagnostics.AddError(
			"Missing Chargebee site",
			"Set the `site` provider attribute or the CHARGEBEE_SITE environment variable.",
		)
	}
	if apiKey == "" {
		resp.Diagnostics.AddError(
			"Missing Chargebee API key",
			"Set the `api_key` provider attribute or the CHARGEBEE_API_KEY environment variable.",
		)
	}
	if resp.Diagnostics.HasError() {
		return
	}

	client := chargebee.NewClient(site, apiKey)
	resp.ResourceData = client
	resp.DataSourceData = client

	tflog.Info(ctx, "configured chargebee provider", map[string]any{"site": site})
}

// clientFromProviderData extracts the configured *chargebee.Client from
// resource/data-source provider data. A nil return with no diagnostics means
// the provider is not configured yet (e.g. during validation).
func clientFromProviderData(data any, diags *diag.Diagnostics) *chargebee.Client {
	if data == nil {
		return nil
	}
	client, ok := data.(*chargebee.Client)
	if !ok {
		diags.AddError("Unexpected provider data", fmt.Sprintf("expected *chargebee.Client, got %T", data))
		return nil
	}
	return client
}

func (p *chargebeeProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewItemFamilyResource,
		NewFeatureResource,
		NewItemResource,
		NewItemPriceResource,
		NewAttachedItemResource,
		NewItemEntitlementResource,
	}
}

func (p *chargebeeProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewItemFamilyDataSource,
	}
}
