// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/jhuiting/terraform-provider-chargebee/internal/chargebee"
)

var (
	_ resource.Resource                = (*itemFamilyResource)(nil)
	_ resource.ResourceWithConfigure   = (*itemFamilyResource)(nil)
	_ resource.ResourceWithImportState = (*itemFamilyResource)(nil)
)

// NewItemFamilyResource returns the chargebee_item_family resource.
func NewItemFamilyResource() resource.Resource {
	return &itemFamilyResource{}
}

type itemFamilyResource struct {
	client *chargebee.Client
}

type itemFamilyResourceModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	Status      types.String `tfsdk:"status"`
}

func (r *itemFamilyResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_item_family"
}

func (r *itemFamilyResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "A Chargebee product family (Product Catalog 2.0). " +
			"If a family with the given id already exists in Chargebee, it is adopted into state and its " +
			"name and description are updated to match this configuration. " +
			"Note: Chargebee cannot archive item families; if deletion is refused because the family is in use, " +
			"the family is left in place with a warning and removed from state.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Required:    true,
				Description: "Unique identifier of the item family. Changing this forces a new resource.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Display name of the item family.",
			},
			"description": schema.StringAttribute{
				Optional:    true,
				Description: "Description of the item family.",
			},
			"status": schema.StringAttribute{
				Computed:    true,
				Description: "Status of the item family in Chargebee.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *itemFamilyResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = clientFromProviderData(req.ProviderData, &resp.Diagnostics)
}

func (r *itemFamilyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan itemFamilyResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	family := chargebee.ItemFamily{
		ID:          plan.ID.ValueString(),
		Name:        plan.Name.ValueString(),
		Description: plan.Description.ValueString(),
	}

	created, err := r.client.CreateItemFamily(ctx, family)
	if chargebee.IsAlreadyExists(err) {
		created = r.adoptItemFamily(ctx, family, plan, &resp.Diagnostics)
		if resp.Diagnostics.HasError() {
			return
		}
	} else if err != nil {
		resp.Diagnostics.AddError("Failed to create item family", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, itemFamilyToModel(created))...)
}

// adoptItemFamily brings a pre-existing Chargebee item family under Terraform
// management by updating its mutable fields to match the configuration. It
// errors if the result still differs from the plan (for example, Chargebee
// cannot clear a description that the configuration omits).
func (r *itemFamilyResource) adoptItemFamily(ctx context.Context, plan chargebee.ItemFamily, planModel itemFamilyResourceModel, diags *diag.Diagnostics) *chargebee.ItemFamily {
	updated, err := r.client.UpdateItemFamily(ctx, plan)
	if err != nil {
		diags.AddError("Failed to adopt existing item family", err.Error())
		return nil
	}

	updatedModel := itemFamilyToModel(updated)
	conflicts := unreconciledFields([]adoptionField{
		{"name", planModel.Name, updatedModel.Name},
		{"description", planModel.Description, updatedModel.Description},
	})
	if len(conflicts) > 0 {
		diags.Append(adoptionConflictError("item family", plan.ID, conflicts, unreconciledConflictNote))
		return nil
	}

	diags.AddWarning(
		"Adopted existing item family",
		fmt.Sprintf("Item family %q already existed in Chargebee and is now managed by Terraform. "+
			"Its name and description were updated to match this configuration.", plan.ID),
	)
	return updated
}

func (r *itemFamilyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state itemFamilyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	family, err := r.client.GetItemFamily(ctx, state.ID.ValueString())
	if chargebee.IsNotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Failed to read item family", err.Error())
		return
	}
	if family.Status == chargebee.StatusDeleted {
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, itemFamilyToModel(family))...)
}

func (r *itemFamilyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan itemFamilyResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	updated, err := r.client.UpdateItemFamily(ctx, chargebee.ItemFamily{
		ID:          plan.ID.ValueString(),
		Name:        plan.Name.ValueString(),
		Description: plan.Description.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to update item family", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, itemFamilyToModel(updated))...)
}

func (r *itemFamilyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state itemFamilyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := state.ID.ValueString()
	outcome, err := chargebee.DeleteWithArchiveFallback(ctx,
		func(ctx context.Context) error { return r.client.DeleteItemFamily(ctx, id) },
		nil,
	)
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete item family", err.Error())
		return
	}
	if outcome == chargebee.LeftInPlace {
		resp.Diagnostics.AddWarning(
			"Item family not deleted",
			fmt.Sprintf("Chargebee refused to delete item family %q because it is in use. "+
				"It was removed from Terraform state but still exists in Chargebee.", id),
		)
	}
}

func (r *itemFamilyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func itemFamilyToModel(f *chargebee.ItemFamily) itemFamilyResourceModel {
	return itemFamilyResourceModel{
		ID:          types.StringValue(f.ID),
		Name:        types.StringValue(f.Name),
		Description: stringOrNull(f.Description),
		Status:      types.StringValue(f.Status),
	}
}

// stringOrNull maps Chargebee's empty string for unset optional fields to null,
// so unset config attributes do not show permanent drift.
func stringOrNull(s string) types.String {
	if s == "" {
		return types.StringNull()
	}
	return types.StringValue(s)
}
