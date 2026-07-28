// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/jhuiting/terraform-provider-chargebee/internal/chargebee"
)

var (
	_ datasource.DataSource              = (*itemFamilyDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*itemFamilyDataSource)(nil)
)

// NewItemFamilyDataSource returns the chargebee_item_family data source.
func NewItemFamilyDataSource() datasource.DataSource {
	return &itemFamilyDataSource{}
}

type itemFamilyDataSource struct {
	client *chargebee.Client
}

type itemFamilyDataSourceModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	Status      types.String `tfsdk:"status"`
}

func (d *itemFamilyDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_item_family"
}

func (d *itemFamilyDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Looks up an existing Chargebee product family by id.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Required:    true,
				Description: "Unique identifier of the item family.",
			},
			"name": schema.StringAttribute{
				Computed:    true,
				Description: "Display name of the item family.",
			},
			"description": schema.StringAttribute{
				Computed:    true,
				Description: "Description of the item family.",
			},
			"status": schema.StringAttribute{
				Computed:    true,
				Description: "Status of the item family in Chargebee.",
			},
		},
	}
}

func (d *itemFamilyDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = clientFromProviderData(req.ProviderData, &resp.Diagnostics)
}

func (d *itemFamilyDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config itemFamilyDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	family, err := d.client.GetItemFamily(ctx, config.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read item family", err.Error())
		return
	}

	state := itemFamilyDataSourceModel{
		ID:          types.StringValue(family.ID),
		Name:        types.StringValue(family.Name),
		Description: stringOrNull(family.Description),
		Status:      types.StringValue(family.Status),
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}
