// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"math"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/jhuiting/terraform-provider-chargebee/internal/chargebee"
)

var (
	_ resource.Resource                = (*attachedItemResource)(nil)
	_ resource.ResourceWithConfigure   = (*attachedItemResource)(nil)
	_ resource.ResourceWithImportState = (*attachedItemResource)(nil)
)

// NewAttachedItemResource returns the chargebee_attached_item resource.
func NewAttachedItemResource() resource.Resource {
	return &attachedItemResource{}
}

type attachedItemResource struct {
	client *chargebee.Client
}

type attachedItemResourceModel struct {
	ID            types.String `tfsdk:"id"`
	ParentItemID  types.String `tfsdk:"parent_item_id"`
	ItemID        types.String `tfsdk:"item_id"`
	Type          types.String `tfsdk:"type"`
	ChargeOnEvent types.String `tfsdk:"charge_on_event"`
	ChargeOnce    types.Bool   `tfsdk:"charge_once"`
	Quantity      types.Int64  `tfsdk:"quantity"`
	BillingCycles types.Int64  `tfsdk:"billing_cycles"`
}

func (r *attachedItemResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_attached_item"
}

func (r *attachedItemResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Attaches a child item (addon or charge) to a parent plan item via Chargebee's Attached Items API, " +
			"expressing plan→charge/addon relationships in Terraform. " +
			"If the child item is already attached to the parent in Chargebee, the attachment is adopted into state " +
			"and its mutable fields are updated to match this configuration.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				Description:   "Chargebee's identifier for the attachment.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"parent_item_id": schema.StringAttribute{
				Required:      true,
				Description:   "Id of the parent plan item the child is attached to. Changing this forces a new resource.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"item_id": schema.StringAttribute{
				Required:      true,
				Description:   "Id of the child item (addon or charge) to attach. Changing this forces a new resource.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"type": schema.StringAttribute{
				Optional:      true,
				Computed:      true,
				Description:   "How the attachment is offered: `recommended`, `mandatory`, or `optional`.",
				Validators:    []validator.String{stringvalidator.OneOf("recommended", "mandatory", "optional")},
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"charge_on_event": schema.StringAttribute{
				Optional: true,
				Computed: true,
				Description: "Event that triggers the charge, e.g. `subscription_creation`. One of: `subscription_creation`, " +
					"`subscription_trial_start`, `plan_activation`, `subscription_activation`, `contract_termination`, `on_demand`.",
				Validators: []validator.String{stringvalidator.OneOf(
					"subscription_creation",
					"subscription_trial_start",
					"plan_activation",
					"subscription_activation",
					"contract_termination",
					"on_demand",
				)},
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"charge_once": schema.BoolAttribute{
				Optional:      true,
				Computed:      true,
				Description:   "Whether the charge is applied only once (rather than every billing cycle).",
				PlanModifiers: []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"quantity": schema.Int64Attribute{
				Optional:      true,
				Computed:      true,
				Description:   "Default quantity of the attached item.",
				PlanModifiers: []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
			},
			"billing_cycles": schema.Int64Attribute{
				Optional:      true,
				Computed:      true,
				Description:   "Number of billing cycles the attached item is charged for.",
				PlanModifiers: []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
			},
		},
	}
}

func (r *attachedItemResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = clientFromProviderData(req.ProviderData, &resp.Diagnostics)
}

func (r *attachedItemResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan attachedItemResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	attached := attachedItemFromModel(plan)
	created, err := r.client.CreateAttachedItem(ctx, attached)
	if chargebee.IsAlreadyExists(err) {
		created = r.adoptAttachedItem(ctx, attached, plan, &resp.Diagnostics)
		if resp.Diagnostics.HasError() {
			return
		}
	} else if err != nil {
		resp.Diagnostics.AddError("Failed to create attached item", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, attachedItemToModel(created))...)
}

// adoptAttachedItem brings a pre-existing attachment of the same child item to
// the same parent under Terraform management and updates its mutable fields to
// match the configuration. Chargebee assigns the attachment the child item's
// id, so it is retrieved by (parent_item_id, item_id). It errors if the update
// cannot reconcile the result with the configuration.
func (r *attachedItemResource) adoptAttachedItem(ctx context.Context, plan chargebee.AttachedItem, planModel attachedItemResourceModel, diags *diag.Diagnostics) *chargebee.AttachedItem {
	existing, err := r.client.GetAttachedItem(ctx, plan.ParentItemID, plan.ItemID)
	if err != nil {
		diags.AddError("Failed to read existing attached item for adoption", err.Error())
		return nil
	}

	plan.ID = existing.ID
	updated, err := r.client.UpdateAttachedItem(ctx, plan)
	if err != nil {
		diags.AddError("Failed to adopt existing attached item", err.Error())
		return nil
	}

	updatedModel := attachedItemToModel(updated)
	conflicts := unreconciledFields([]adoptionField{
		{"type", planModel.Type, updatedModel.Type},
		{"charge_on_event", planModel.ChargeOnEvent, updatedModel.ChargeOnEvent},
		{"charge_once", planModel.ChargeOnce, updatedModel.ChargeOnce},
		{"quantity", planModel.Quantity, updatedModel.Quantity},
		{"billing_cycles", planModel.BillingCycles, updatedModel.BillingCycles},
	})
	if len(conflicts) > 0 {
		diags.Append(adoptionConflictError("attached item", plan.ItemID, conflicts, unreconciledConflictNote))
		return nil
	}

	diags.AddWarning(
		"Adopted existing attached item",
		fmt.Sprintf("Item %q was already attached to parent %q in Chargebee and is now managed by Terraform. "+
			"Its mutable fields were updated to match this configuration.", plan.ItemID, plan.ParentItemID),
	)
	return updated
}

func (r *attachedItemResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state attachedItemResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	attached, err := r.client.GetAttachedItem(ctx, state.ParentItemID.ValueString(), state.ID.ValueString())
	if chargebee.IsNotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Failed to read attached item", err.Error())
		return
	}
	if attached.Status == chargebee.StatusDeleted {
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, attachedItemToModel(attached))...)
}

func (r *attachedItemResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan attachedItemResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	updated, err := r.client.UpdateAttachedItem(ctx, attachedItemFromModel(plan))
	if err != nil {
		resp.Diagnostics.AddError("Failed to update attached item", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, attachedItemToModel(updated))...)
}

func (r *attachedItemResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state attachedItemResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteAttachedItem(ctx, state.ParentItemID.ValueString(), state.ID.ValueString())
	if err != nil && !chargebee.IsNotFound(err) {
		resp.Diagnostics.AddError("Failed to delete attached item", err.Error())
	}
}

func (r *attachedItemResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parentItemID, id, ok := strings.Cut(req.ID, ":")
	if !ok || parentItemID == "" || id == "" {
		resp.Diagnostics.AddError(
			"Invalid import ID",
			fmt.Sprintf("Expected import ID in the format \"parent_item_id:id\", got %q.", req.ID),
		)
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("parent_item_id"), parentItemID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), id)...)
}

func attachedItemFromModel(m attachedItemResourceModel) chargebee.AttachedItem {
	return chargebee.AttachedItem{
		ID:            m.ID.ValueString(),
		ParentItemID:  m.ParentItemID.ValueString(),
		ItemID:        m.ItemID.ValueString(),
		Type:          m.Type.ValueString(),
		ChargeOnEvent: m.ChargeOnEvent.ValueString(),
		ChargeOnce:    m.ChargeOnce.ValueBoolPointer(),
		Quantity:      int32PtrFromInt64(m.Quantity),
		BillingCycles: int32PtrFromInt64(m.BillingCycles),
	}
}

func attachedItemToModel(a *chargebee.AttachedItem) attachedItemResourceModel {
	return attachedItemResourceModel{
		ID:            types.StringValue(a.ID),
		ParentItemID:  types.StringValue(a.ParentItemID),
		ItemID:        types.StringValue(a.ItemID),
		Type:          stringOrNull(a.Type),
		ChargeOnEvent: stringOrNull(a.ChargeOnEvent),
		ChargeOnce:    types.BoolPointerValue(a.ChargeOnce),
		Quantity:      int64FromInt32Ptr(a.Quantity),
		BillingCycles: int64FromInt32Ptr(a.BillingCycles),
	}
}

func int32PtrFromInt64(v types.Int64) *int32 {
	if v.IsNull() || v.IsUnknown() {
		return nil
	}
	n := v.ValueInt64()
	if n < math.MinInt32 {
		n = math.MinInt32
	} else if n > math.MaxInt32 {
		n = math.MaxInt32
	}
	i := int32(n) //nolint:gosec // n is clamped to the int32 range above
	return &i
}

func int64FromInt32Ptr(p *int32) types.Int64 {
	if p == nil {
		return types.Int64Null()
	}
	return types.Int64Value(int64(*p))
}
