// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/jhuiting/terraform-provider-chargebee/internal/chargebee"
)

var (
	_ resource.Resource                = (*itemEntitlementResource)(nil)
	_ resource.ResourceWithConfigure   = (*itemEntitlementResource)(nil)
	_ resource.ResourceWithImportState = (*itemEntitlementResource)(nil)
)

// NewItemEntitlementResource returns the chargebee_item_entitlement resource.
func NewItemEntitlementResource() resource.Resource {
	return &itemEntitlementResource{}
}

type itemEntitlementResource struct {
	client *chargebee.Client
}

type itemEntitlementResourceModel struct {
	ID        types.String `tfsdk:"id"`
	ItemID    types.String `tfsdk:"item_id"`
	ItemType  types.String `tfsdk:"item_type"`
	FeatureID types.String `tfsdk:"feature_id"`
	Value     types.String `tfsdk:"value"`
}

func (r *itemEntitlementResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_item_entitlement"
}

func (r *itemEntitlementResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Grants a feature to an item (plan, addon, or charge) via Chargebee's Item Entitlements API. " +
			"Chargebee's upsert is idempotent, so an entitlement already present in Chargebee is adopted into state " +
			"and its value updated to match this configuration on the first apply.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				Description:   "Composite identifier in the form `item_id:feature_id`.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"item_id": schema.StringAttribute{
				Required:      true,
				Description:   "Id of the item the feature is granted to. Changing this forces a new resource.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"item_type": schema.StringAttribute{
				Required:      true,
				Description:   "Type of the item: `plan`, `addon`, or `charge`. Must match the item's type. Changing this forces a new resource.",
				Validators:    []validator.String{stringvalidator.OneOf("plan", "addon", "charge")},
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"feature_id": schema.StringAttribute{
				Required:      true,
				Description:   "Id of the granted feature. Changing this forces a new resource.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"value": schema.StringAttribute{
				Optional: true,
				Computed: true,
				Description: "Entitlement value. For a switch feature use `true` (the default); " +
					"quantity and range features carry a numeric value.",
				Default: stringdefault.StaticString("true"),
			},
		},
	}
}

func (r *itemEntitlementResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = clientFromProviderData(req.ProviderData, &resp.Diagnostics)
}

func (r *itemEntitlementResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan itemEntitlementResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	created, err := r.client.UpsertItemEntitlement(ctx, itemEntitlementFromModel(plan))
	if err != nil {
		resp.Diagnostics.AddError("Failed to create item entitlement", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, itemEntitlementToModel(created))...)
}

func (r *itemEntitlementResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state itemEntitlementResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	entitlement, err := r.client.GetItemEntitlement(ctx, state.ItemID.ValueString(), state.FeatureID.ValueString())
	if chargebee.IsNotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Failed to read item entitlement", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, itemEntitlementToModel(entitlement))...)
}

func (r *itemEntitlementResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan itemEntitlementResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	updated, err := r.client.UpsertItemEntitlement(ctx, itemEntitlementFromModel(plan))
	if err != nil {
		resp.Diagnostics.AddError("Failed to update item entitlement", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, itemEntitlementToModel(updated))...)
}

func (r *itemEntitlementResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state itemEntitlementResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.RemoveItemEntitlement(ctx, itemEntitlementFromModel(state))
	if err != nil && !chargebee.IsNotFound(err) {
		resp.Diagnostics.AddError("Failed to delete item entitlement", err.Error())
	}
}

func (r *itemEntitlementResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	itemID, featureID, ok := strings.Cut(req.ID, ":")
	if !ok || itemID == "" || featureID == "" {
		resp.Diagnostics.AddError(
			"Invalid import ID",
			fmt.Sprintf("Expected import ID in the format \"item_id:feature_id\", got %q.", req.ID),
		)
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("item_id"), itemID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("feature_id"), featureID)...)
}

func itemEntitlementFromModel(m itemEntitlementResourceModel) chargebee.ItemEntitlement {
	return chargebee.ItemEntitlement{
		ItemID:    m.ItemID.ValueString(),
		ItemType:  m.ItemType.ValueString(),
		FeatureID: m.FeatureID.ValueString(),
		Value:     m.Value.ValueString(),
	}
}

func itemEntitlementToModel(e *chargebee.ItemEntitlement) itemEntitlementResourceModel {
	return itemEntitlementResourceModel{
		ID:        types.StringValue(e.ItemID + ":" + e.FeatureID),
		ItemID:    types.StringValue(e.ItemID),
		ItemType:  types.StringValue(e.ItemType),
		FeatureID: types.StringValue(e.FeatureID),
		Value:     types.StringValue(e.Value),
	}
}
