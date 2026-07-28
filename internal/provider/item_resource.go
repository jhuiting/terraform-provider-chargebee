// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/jhuiting/terraform-provider-chargebee/internal/chargebee"
)

var (
	_ resource.Resource                   = (*itemResource)(nil)
	_ resource.ResourceWithConfigure      = (*itemResource)(nil)
	_ resource.ResourceWithImportState    = (*itemResource)(nil)
	_ resource.ResourceWithValidateConfig = (*itemResource)(nil)
)

// NewItemResource returns the chargebee_item resource.
func NewItemResource() resource.Resource {
	return &itemResource{}
}

type itemResource struct {
	client *chargebee.Client
}

type itemResourceModel struct {
	ID                 types.String `tfsdk:"id"`
	ItemFamilyID       types.String `tfsdk:"item_family_id"`
	Type               types.String `tfsdk:"type"`
	Name               types.String `tfsdk:"name"`
	ExternalName       types.String `tfsdk:"external_name"`
	Description        types.String `tfsdk:"description"`
	Unit               types.String `tfsdk:"unit"`
	RedirectURL        types.String `tfsdk:"redirect_url"`
	Metered            types.Bool   `tfsdk:"metered"`
	UsageCalculation   types.String `tfsdk:"usage_calculation"`
	EnabledForCheckout types.Bool   `tfsdk:"enabled_for_checkout"`
	EnabledInPortal    types.Bool   `tfsdk:"enabled_in_portal"`
	IsShippable        types.Bool   `tfsdk:"is_shippable"`
	IsGiftable         types.Bool   `tfsdk:"is_giftable"`
	Metadata           types.Map    `tfsdk:"metadata"`
	Status             types.String `tfsdk:"status"`
}

func (r *itemResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_item"
}

func (r *itemResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "A Chargebee item (Product Catalog 2.0): a plan, addon, or charge. " +
			"Set name, type, and item_family_id together to create a new item. Omit all three to adopt an " +
			"existing item by id and track it as-is (every field is read from Chargebee); this is the way to " +
			"bring an archived item under management with just `id` and `status = \"archived\"`. " +
			"If an item with the given id already exists when creating, it is adopted into state and its " +
			"mutable fields are updated to match this configuration; adoption fails if a create-only field " +
			"(item_family_id, type, metered, usage_calculation, is_giftable) differs from the existing item. " +
			"If deletion is refused because the item is in use, it is archived instead (the id stays reserved in Chargebee).",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Required:      true,
				Description:   "Unique identifier of the item. Changing this forces a new resource.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"item_family_id": schema.StringAttribute{
				Optional:      true,
				Computed:      true,
				Description:   "Id of the product family this item belongs to. Read from Chargebee when omitted; must be set together with `name` and `type` to create a new item. Changing this forces a new resource.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown(), stringplanmodifier.RequiresReplace()},
			},
			"type": schema.StringAttribute{
				Optional:      true,
				Computed:      true,
				Description:   "Item type: `plan`, `addon`, or `charge`. Read from Chargebee when omitted; must be set together with `name` and `item_family_id` to create a new item. Changing this forces a new resource.",
				Validators:    []validator.String{stringvalidator.OneOf("plan", "addon", "charge")},
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown(), stringplanmodifier.RequiresReplace()},
			},
			"name": schema.StringAttribute{
				Optional:      true,
				Computed:      true,
				Description:   "Display name of the item. Read from Chargebee when omitted; must be set together with `type` and `item_family_id` to create a new item.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"external_name": schema.StringAttribute{
				Optional:      true,
				Computed:      true,
				Description:   "Customer-facing name shown on invoices and checkout (Chargebee defaults it to the item name).",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"description": schema.StringAttribute{
				Optional:      true,
				Computed:      true,
				Description:   "Description of the item.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"unit": schema.StringAttribute{
				Optional:      true,
				Computed:      true,
				Description:   "Display unit for quantities (e.g. `seat`, `GB`).",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"redirect_url": schema.StringAttribute{
				Optional:      true,
				Computed:      true,
				Description:   "URL to redirect to after checkout.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"metered": schema.BoolAttribute{
				Optional:      true,
				Computed:      true,
				Description:   "Whether usage for this item is metered (billed in arrears). Chargebee defaults this to false. Changing this forces a new resource.",
				PlanModifiers: []planmodifier.Bool{boolplanmodifier.UseStateForUnknown(), boolplanmodifier.RequiresReplace()},
			},
			"usage_calculation": schema.StringAttribute{
				Optional:      true,
				Computed:      true,
				Description:   "How usage is aggregated for metered items: `sum_of_usages`, `last_usage`, or `max_usage`. Requires `metered = true`. Changing this forces a new resource.",
				Validators:    []validator.String{stringvalidator.OneOf("sum_of_usages", "last_usage", "max_usage")},
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown(), stringplanmodifier.RequiresReplace()},
			},
			"enabled_for_checkout": schema.BoolAttribute{
				Optional:      true,
				Computed:      true,
				Description:   "Whether the item can be purchased via hosted checkout pages (Chargebee defaults to true).",
				PlanModifiers: []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"enabled_in_portal": schema.BoolAttribute{
				Optional:      true,
				Computed:      true,
				Description:   "Whether the item is visible in the customer portal (Chargebee defaults to true).",
				PlanModifiers: []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"is_shippable": schema.BoolAttribute{
				Optional:      true,
				Computed:      true,
				Description:   "Whether orders are created for this item (shippable products).",
				PlanModifiers: []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"is_giftable": schema.BoolAttribute{
				Optional:      true,
				Computed:      true,
				Description:   "Whether the item can be gifted. Chargebee defaults this to false. Changing this forces a new resource.",
				PlanModifiers: []planmodifier.Bool{boolplanmodifier.UseStateForUnknown(), boolplanmodifier.RequiresReplace()},
			},
			"metadata": schema.MapAttribute{
				ElementType:   types.StringType,
				Optional:      true,
				Computed:      true,
				Description:   "Arbitrary string key/value metadata stored on the item.",
				PlanModifiers: []planmodifier.Map{mapplanmodifier.UseStateForUnknown()},
			},
			"status": schema.StringAttribute{
				Optional:      true,
				Computed:      true,
				Description:   "Status of the item in Chargebee: `active` or `archived`. Defaults to the item's current status in Chargebee; set `archived` to archive the item under Terraform management.",
				Validators:    []validator.String{stringvalidator.OneOf(chargebee.StatusActive, chargebee.StatusArchived)},
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
		},
	}
}

func (r *itemResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var config itemResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	for _, field := range createIdentityMissing([]identityField{
		{"name", !config.Name.IsNull()},
		{"type", !config.Type.IsNull()},
		{"item_family_id", !config.ItemFamilyID.IsNull()},
	}) {
		resp.Diagnostics.AddAttributeError(
			path.Root(field),
			"Incomplete item identity",
			"Set name, type, and item_family_id together to create a new item, "+
				"or omit all three to adopt an existing item by id.",
		)
	}

	if config.UsageCalculation.IsUnknown() || config.Metered.IsUnknown() {
		return
	}
	if !config.UsageCalculation.IsNull() && !config.Metered.ValueBool() {
		resp.Diagnostics.AddAttributeError(
			path.Root("usage_calculation"),
			"usage_calculation requires metered = true",
			"The usage_calculation attribute is only valid for metered items.",
		)
	}
}

func (r *itemResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = clientFromProviderData(req.ProviderData, &resp.Diagnostics)
}

func (r *itemResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan itemResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var created *chargebee.Item
	if provided(plan.Name) && provided(plan.Type) && provided(plan.ItemFamilyID) {
		created = r.createItem(ctx, plan, &resp.Diagnostics)
	} else {
		created = r.adoptItemByID(ctx, plan, &resp.Diagnostics)
	}
	if resp.Diagnostics.HasError() {
		return
	}

	state, diags := itemToModel(ctx, created)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

// createItem creates a new item, adopting and reconciling an existing item when
// Chargebee reports the id already exists, then applies the desired status.
func (r *itemResource) createItem(ctx context.Context, plan itemResourceModel, diags *diag.Diagnostics) *chargebee.Item {
	item, modelDiags := itemFromModel(ctx, plan)
	diags.Append(modelDiags...)
	if diags.HasError() {
		return nil
	}

	created, err := r.client.CreateItem(ctx, item)
	if chargebee.IsAlreadyExists(err) {
		created = r.adoptItem(ctx, item, plan, diags)
		if diags.HasError() {
			return nil
		}
	} else if err != nil {
		diags.AddError("Failed to create item", err.Error())
		return nil
	}

	if item.Status == chargebee.StatusArchived && created.Status != chargebee.StatusArchived {
		created, err = r.client.SetItemStatus(ctx, item.ID, chargebee.StatusArchived)
		if err != nil {
			diags.AddError("Failed to set item status after create", err.Error())
			return nil
		}
	}
	return created
}

// adoptItemByID brings an existing Chargebee item under management by id alone,
// reading every field from Chargebee and applying only a status transition. It
// never edits fields, so it is safe on archived items. It errors if no item with
// the id exists, since without the identity fields there is nothing to create.
func (r *itemResource) adoptItemByID(ctx context.Context, plan itemResourceModel, diags *diag.Diagnostics) *chargebee.Item {
	id := plan.ID.ValueString()
	existing, err := r.client.GetItem(ctx, id)
	if chargebee.IsNotFound(err) {
		diags.AddError(
			"Cannot create item",
			fmt.Sprintf("To create a new item, set name, type, and item_family_id. To adopt an existing "+
				"item by id, that item must already exist in Chargebee; no item with id %q was found.", id),
		)
		return nil
	}
	if err != nil {
		diags.AddError("Failed to read existing item for adoption", err.Error())
		return nil
	}

	created := existing
	if plan.Status.ValueString() == chargebee.StatusArchived && created.Status != chargebee.StatusArchived {
		created, err = r.client.SetItemStatus(ctx, id, chargebee.StatusArchived)
		if err != nil {
			diags.AddError("Failed to set item status after adoption", err.Error())
			return nil
		}
	}

	diags.AddWarning(
		"Adopted existing item",
		fmt.Sprintf("Item %q was adopted by id and is now managed by Terraform. "+
			"All fields were read from Chargebee.", id),
	)
	return created
}

// provided reports whether a configuration supplied a non-empty value for a
// string attribute (as opposed to omitting it or leaving it computed-unknown).
func provided(v types.String) bool {
	return !v.IsNull() && !v.IsUnknown() && v.ValueString() != ""
}

// adoptItem brings a pre-existing Chargebee item under Terraform management:
// it errors if any create-only field differs from the configuration, otherwise
// updates the item's mutable fields to match and returns the reconciled item.
func (r *itemResource) adoptItem(ctx context.Context, plan chargebee.Item, planModel itemResourceModel, diags *diag.Diagnostics) *chargebee.Item {
	existing, err := r.client.GetItem(ctx, plan.ID)
	if err != nil {
		diags.AddError("Failed to read existing item for adoption", err.Error())
		return nil
	}
	if conflicts := itemAdoptionConflicts(plan, *existing); len(conflicts) > 0 {
		diags.Append(adoptionConflictError("item", plan.ID, conflicts, createOnlyConflictNote))
		return nil
	}

	// Chargebee rejects field edits on archived items, so an archived item is
	// reactivated before its fields are reconciled; createItem restores the
	// archived status afterwards when the configuration asks for it.
	if existing.Status == chargebee.StatusArchived {
		if _, err := r.client.SetItemStatus(ctx, plan.ID, chargebee.StatusActive); err != nil {
			diags.AddError("Failed to reactivate item for adoption", err.Error())
			return nil
		}
	}

	updated, err := r.client.UpdateItem(ctx, plan)
	if err != nil {
		diags.AddError("Failed to adopt existing item", err.Error())
		return nil
	}

	updatedModel, modelDiags := itemToModel(ctx, updated)
	diags.Append(modelDiags...)
	if diags.HasError() {
		return nil
	}
	conflicts := unreconciledFields([]adoptionField{
		{"name", planModel.Name, updatedModel.Name},
		{"description", planModel.Description, updatedModel.Description},
		{"external_name", planModel.ExternalName, updatedModel.ExternalName},
		{"unit", planModel.Unit, updatedModel.Unit},
		{"redirect_url", planModel.RedirectURL, updatedModel.RedirectURL},
		{"enabled_for_checkout", planModel.EnabledForCheckout, updatedModel.EnabledForCheckout},
		{"enabled_in_portal", planModel.EnabledInPortal, updatedModel.EnabledInPortal},
		{"is_shippable", planModel.IsShippable, updatedModel.IsShippable},
		{"metadata", planModel.Metadata, updatedModel.Metadata},
	})
	if len(conflicts) > 0 {
		diags.Append(adoptionConflictError("item", plan.ID, conflicts, unreconciledConflictNote))
		return nil
	}

	diags.AddWarning(
		"Adopted existing item",
		fmt.Sprintf("Item %q already existed in Chargebee and is now managed by Terraform. "+
			"Its mutable fields were updated to match this configuration.", plan.ID),
	)
	return updated
}

func (r *itemResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state itemResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	item, err := r.client.GetItem(ctx, state.ID.ValueString())
	if chargebee.IsNotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Failed to read item", err.Error())
		return
	}
	if item.Status == chargebee.StatusDeleted {
		resp.State.RemoveResource(ctx)
		return
	}

	newState, diags := itemToModel(ctx, item)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, newState)...)
}

func (r *itemResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan itemResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	item, diags := itemFromModel(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	updated, err := r.applyItemUpdate(ctx, item)
	if err != nil {
		resp.Diagnostics.AddError("Failed to update item", err.Error())
		return
	}

	state, diags := itemToModel(ctx, updated)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

// applyItemUpdate reconciles field and status changes. Chargebee forbids field
// edits on archived items and cascades archival from an item to its prices, so a
// status transition is issued on its own; when the item is already at the
// requested status the call is skipped so re-applies stay a no-op.
func (r *itemResource) applyItemUpdate(ctx context.Context, item chargebee.Item) (*chargebee.Item, error) {
	desired := item.Status
	if desired == "" {
		desired = chargebee.StatusActive
	}

	current, err := r.client.GetItem(ctx, item.ID)
	if err != nil {
		return nil, err
	}

	if desired == chargebee.StatusArchived {
		if current.Status == chargebee.StatusArchived {
			return current, nil
		}
		return r.client.SetItemStatus(ctx, item.ID, chargebee.StatusArchived)
	}

	if current.Status == chargebee.StatusArchived {
		if _, err := r.client.SetItemStatus(ctx, item.ID, chargebee.StatusActive); err != nil {
			return nil, err
		}
	}
	return r.client.UpdateItem(ctx, item)
}

func (r *itemResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state itemResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := state.ID.ValueString()
	outcome, err := chargebee.DeleteWithArchiveFallback(ctx,
		func(ctx context.Context) error { return r.client.DeleteItem(ctx, id) },
		func(ctx context.Context) error { return r.client.ArchiveItem(ctx, id) },
	)
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete item", err.Error())
		return
	}
	if outcome == chargebee.Archived {
		resp.Diagnostics.AddWarning(
			"Item archived instead of deleted",
			fmt.Sprintf("Chargebee refused to delete item %q because it is in use; it was archived instead. "+
				"The id remains reserved in Chargebee.", id),
		)
	}
}

func (r *itemResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func itemFromModel(ctx context.Context, m itemResourceModel) (chargebee.Item, diag.Diagnostics) {
	item := chargebee.Item{
		ID:                 m.ID.ValueString(),
		ItemFamilyID:       m.ItemFamilyID.ValueString(),
		Type:               m.Type.ValueString(),
		Name:               m.Name.ValueString(),
		ExternalName:       m.ExternalName.ValueString(),
		Description:        m.Description.ValueString(),
		Unit:               m.Unit.ValueString(),
		RedirectURL:        m.RedirectURL.ValueString(),
		Metered:            m.Metered.ValueBool(),
		UsageCalculation:   m.UsageCalculation.ValueString(),
		EnabledForCheckout: m.EnabledForCheckout.ValueBoolPointer(),
		EnabledInPortal:    m.EnabledInPortal.ValueBoolPointer(),
		IsShippable:        m.IsShippable.ValueBoolPointer(),
		IsGiftable:         m.IsGiftable.ValueBoolPointer(),
		Status:             m.Status.ValueString(),
	}
	var diags diag.Diagnostics
	if !m.Metadata.IsNull() && !m.Metadata.IsUnknown() {
		metadata := map[string]string{}
		diags = m.Metadata.ElementsAs(ctx, &metadata, false)
		item.Metadata = metadata
	}
	return item, diags
}

func itemToModel(ctx context.Context, i *chargebee.Item) (itemResourceModel, diag.Diagnostics) {
	m := itemResourceModel{
		ID:                 types.StringValue(i.ID),
		ItemFamilyID:       types.StringValue(i.ItemFamilyID),
		Type:               types.StringValue(i.Type),
		Name:               types.StringValue(i.Name),
		ExternalName:       stringOrNull(i.ExternalName),
		Description:        stringOrNull(i.Description),
		Unit:               stringOrNull(i.Unit),
		RedirectURL:        stringOrNull(i.RedirectURL),
		Metered:            types.BoolValue(i.Metered),
		UsageCalculation:   stringOrNull(i.UsageCalculation),
		EnabledForCheckout: types.BoolPointerValue(i.EnabledForCheckout),
		EnabledInPortal:    types.BoolPointerValue(i.EnabledInPortal),
		IsShippable:        types.BoolPointerValue(i.IsShippable),
		IsGiftable:         types.BoolPointerValue(i.IsGiftable),
		Status:             types.StringValue(i.Status),
	}
	if len(i.Metadata) == 0 {
		m.Metadata = types.MapNull(types.StringType)
		return m, nil
	}
	metadata, diags := types.MapValueFrom(ctx, types.StringType, i.Metadata)
	m.Metadata = metadata
	return m, diags
}
