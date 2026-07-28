// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int32planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/jhuiting/terraform-provider-chargebee/internal/chargebee"
)

var (
	_ resource.Resource                   = (*itemPriceResource)(nil)
	_ resource.ResourceWithConfigure      = (*itemPriceResource)(nil)
	_ resource.ResourceWithImportState    = (*itemPriceResource)(nil)
	_ resource.ResourceWithValidateConfig = (*itemPriceResource)(nil)
)

// NewItemPriceResource returns the chargebee_item_price resource.
func NewItemPriceResource() resource.Resource {
	return &itemPriceResource{}
}

type itemPriceResource struct {
	client *chargebee.Client
}

type tierModel struct {
	StartingUnit types.Int32 `tfsdk:"starting_unit"`
	EndingUnit   types.Int32 `tfsdk:"ending_unit"`
	Price        types.Int64 `tfsdk:"price"`
}

var tierObjectType = types.ObjectType{AttrTypes: map[string]attr.Type{
	"starting_unit": types.Int32Type,
	"ending_unit":   types.Int32Type,
	"price":         types.Int64Type,
}}

type itemPriceResourceModel struct {
	ID                        types.String `tfsdk:"id"`
	ItemID                    types.String `tfsdk:"item_id"`
	Name                      types.String `tfsdk:"name"`
	ExternalName              types.String `tfsdk:"external_name"`
	Description               types.String `tfsdk:"description"`
	InvoiceNotes              types.String `tfsdk:"invoice_notes"`
	CurrencyCode              types.String `tfsdk:"currency_code"`
	PricingModel              types.String `tfsdk:"pricing_model"`
	Price                     types.Int64  `tfsdk:"price"`
	Tiers                     types.List   `tfsdk:"tiers"`
	Period                    types.Int32  `tfsdk:"period"`
	PeriodUnit                types.String `tfsdk:"period_unit"`
	TrialPeriod               types.Int32  `tfsdk:"trial_period"`
	TrialPeriodUnit           types.String `tfsdk:"trial_period_unit"`
	BillingCycles             types.Int32  `tfsdk:"billing_cycles"`
	FreeQuantity              types.Int32  `tfsdk:"free_quantity"`
	IsTaxable                 types.Bool   `tfsdk:"is_taxable"`
	ShowDescriptionInInvoices types.Bool   `tfsdk:"show_description_in_invoices"`
	ShowDescriptionInQuotes   types.Bool   `tfsdk:"show_description_in_quotes"`
	Status                    types.String `tfsdk:"status"`
}

func (r *itemPriceResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_item_price"
}

func (r *itemPriceResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "A Chargebee item price (Product Catalog 2.0): the price of an item in one currency and billing period. " +
			"Set item_id, name, currency_code, and pricing_model together to create a new price. Omit all four to " +
			"adopt an existing price by id and track it as-is (every field is read from Chargebee); this is the way " +
			"to bring an archived price under management with just `id` and `status = \"archived\"`. " +
			"If an item price with the given id already exists when creating, it is adopted into state and its " +
			"mutable fields are updated to match this configuration; adoption fails if a create-only field " +
			"(item_id, currency_code) differs from the existing price. " +
			"If deletion is refused because the price is in use, it is archived instead (the id stays reserved in Chargebee).",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Required:      true,
				Description:   "Unique identifier of the item price. Changing this forces a new resource.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"item_id": schema.StringAttribute{
				Optional:      true,
				Computed:      true,
				Description:   "Id of the item this price belongs to. Read from Chargebee when omitted; must be set together with `name`, `currency_code`, and `pricing_model` to create a new price. Changing this forces a new resource.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown(), stringplanmodifier.RequiresReplace()},
			},
			"name": schema.StringAttribute{
				Optional:      true,
				Computed:      true,
				Description:   "Display name of the item price (unique across the site). Read from Chargebee when omitted; must be set together with `item_id`, `currency_code`, and `pricing_model` to create a new price.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"external_name": schema.StringAttribute{
				Optional:      true,
				Computed:      true,
				Description:   "Customer-facing name shown on invoices and checkout (Chargebee defaults it to the price name).",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"description": schema.StringAttribute{
				Optional:      true,
				Computed:      true,
				Description:   "Description of the item price.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"invoice_notes": schema.StringAttribute{
				Optional:      true,
				Computed:      true,
				Description:   "Notes added to invoices for this price.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"currency_code": schema.StringAttribute{
				Optional: true,
				Computed: true,
				Description: "ISO 4217 currency code (e.g. `EUR`). Read from Chargebee when omitted; must be set together with " +
					"`item_id`, `name`, and `pricing_model` to create a new price. Changing this forces a new resource: " +
					"a price in a different currency is a different price point.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown(), stringplanmodifier.RequiresReplace()},
			},
			"pricing_model": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Pricing model: `flat_fee`, `per_unit`, `tiered`, `volume`, or `stairstep`. Read from Chargebee when omitted; must be set together with `item_id`, `name`, and `currency_code` to create a new price.",
				Validators: []validator.String{
					stringvalidator.OneOf(chargebee.PricingModelFlatFee, chargebee.PricingModelPerUnit,
						chargebee.PricingModelTiered, chargebee.PricingModelVolume, chargebee.PricingModelStairstep),
				},
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"price": schema.Int64Attribute{
				Optional:      true,
				Computed:      true,
				Description:   "Price in the currency's smallest unit (e.g. cents). Required for `flat_fee` and `per_unit`; not valid for tier-based models.",
				PlanModifiers: []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
			},
			"period": schema.Int32Attribute{
				Optional:      true,
				Computed:      true,
				Description:   "Length of the billing period. Set together with `period_unit` for recurring prices; omit both for one-time charges.",
				PlanModifiers: []planmodifier.Int32{int32planmodifier.UseStateForUnknown()},
			},
			"period_unit": schema.StringAttribute{
				Optional:      true,
				Computed:      true,
				Description:   "Unit of the billing period: `day`, `week`, `month`, or `year`.",
				Validators:    []validator.String{stringvalidator.OneOf("day", "week", "month", "year")},
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"trial_period": schema.Int32Attribute{
				Optional:      true,
				Computed:      true,
				Description:   "Length of the free trial. Set together with `trial_period_unit`.",
				PlanModifiers: []planmodifier.Int32{int32planmodifier.UseStateForUnknown()},
			},
			"trial_period_unit": schema.StringAttribute{
				Optional:      true,
				Computed:      true,
				Description:   "Unit of the trial period: `day` or `month`.",
				Validators:    []validator.String{stringvalidator.OneOf("day", "month")},
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"billing_cycles": schema.Int32Attribute{
				Optional:      true,
				Computed:      true,
				Description:   "Number of billing cycles after which the subscription item expires. Omit for indefinite billing.",
				PlanModifiers: []planmodifier.Int32{int32planmodifier.UseStateForUnknown()},
			},
			"free_quantity": schema.Int32Attribute{
				Optional:      true,
				Computed:      true,
				Description:   "Quantity included for free (Chargebee defaults to 0).",
				PlanModifiers: []planmodifier.Int32{int32planmodifier.UseStateForUnknown()},
			},
			"is_taxable": schema.BoolAttribute{
				Optional:      true,
				Computed:      true,
				Description:   "Whether the price is taxable (Chargebee defaults to true).",
				PlanModifiers: []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"show_description_in_invoices": schema.BoolAttribute{
				Optional:      true,
				Computed:      true,
				Description:   "Whether the description is shown on invoices.",
				PlanModifiers: []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"show_description_in_quotes": schema.BoolAttribute{
				Optional:      true,
				Computed:      true,
				Description:   "Whether the description is shown on quotes.",
				PlanModifiers: []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"status": schema.StringAttribute{
				Optional:      true,
				Computed:      true,
				Description:   "Status of the item price in Chargebee: `active` or `archived`. Defaults to the price's current status in Chargebee; set `archived` to archive the price under Terraform management.",
				Validators:    []validator.String{stringvalidator.OneOf(chargebee.StatusActive, chargebee.StatusArchived)},
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"tiers": schema.ListNestedAttribute{
				Optional: true,
				Computed: true,
				Description: "Quantity tiers for the `tiered`, `volume`, and `stairstep` pricing models, " +
					"in ascending order of `starting_unit`. Omit `ending_unit` on the last tier for an unbounded tier.",
				PlanModifiers: []planmodifier.List{listplanmodifier.UseStateForUnknown()},
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"starting_unit": schema.Int32Attribute{
							Required:    true,
							Description: "First unit covered by this tier.",
						},
						"ending_unit": schema.Int32Attribute{
							Optional:    true,
							Description: "Last unit covered by this tier; omit for an unbounded final tier.",
						},
						"price": schema.Int64Attribute{
							Required:    true,
							Description: "Price for this tier in the currency's smallest unit.",
						},
					},
				},
			},
		},
	}
}

func (r *itemPriceResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	// Attributes are read individually as framework types: the full model uses
	// a Go slice for tiers, which cannot represent an unknown list (e.g. one
	// derived from an unresolved variable during module expansion).
	var itemID, name, currencyCode, pricingModel, periodUnit types.String
	var price types.Int64
	var period types.Int32
	var tiers types.List
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("item_id"), &itemID)...)
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("name"), &name)...)
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("currency_code"), &currencyCode)...)
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("pricing_model"), &pricingModel)...)
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("price"), &price)...)
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("period"), &period)...)
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("period_unit"), &periodUnit)...)
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("tiers"), &tiers)...)
	if resp.Diagnostics.HasError() {
		return
	}

	for _, field := range createIdentityMissing([]identityField{
		{"item_id", !itemID.IsNull()},
		{"name", !name.IsNull()},
		{"currency_code", !currencyCode.IsNull()},
		{"pricing_model", !pricingModel.IsNull()},
	}) {
		resp.Diagnostics.AddAttributeError(
			path.Root(field),
			"Incomplete item price identity",
			"Set item_id, name, currency_code, and pricing_model together to create a new price, "+
				"or omit all four to adopt an existing price by id.",
		)
	}

	if pricingModel.IsUnknown() || price.IsUnknown() || period.IsUnknown() || periodUnit.IsUnknown() || tiers.IsUnknown() {
		return
	}

	cfg := itemPriceConfig{
		PricingModel:  pricingModel.ValueString(),
		HasPrice:      !price.IsNull(),
		TierCount:     len(tiers.Elements()),
		HasPeriod:     !period.IsNull(),
		HasPeriodUnit: !periodUnit.IsNull(),
	}
	for _, e := range itemPriceConfigErrors(cfg) {
		resp.Diagnostics.AddAttributeError(path.Root(e.Attribute), e.Summary, e.Detail)
	}
}

func (r *itemPriceResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = clientFromProviderData(req.ProviderData, &resp.Diagnostics)
}

func (r *itemPriceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan itemPriceResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var created *chargebee.ItemPrice
	if provided(plan.ItemID) && provided(plan.Name) && provided(plan.CurrencyCode) && provided(plan.PricingModel) {
		created = r.createItemPrice(ctx, plan, &resp.Diagnostics)
	} else {
		created = r.adoptItemPriceByID(ctx, plan, &resp.Diagnostics)
	}
	if resp.Diagnostics.HasError() {
		return
	}

	state, diags := itemPriceToModel(ctx, created)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

// createItemPrice creates a new item price, adopting and reconciling an existing
// price when Chargebee reports the id already exists, then applies the desired
// status.
func (r *itemPriceResource) createItemPrice(ctx context.Context, plan itemPriceResourceModel, diags *diag.Diagnostics) *chargebee.ItemPrice {
	price, priceDiags := itemPriceFromModel(ctx, plan)
	diags.Append(priceDiags...)
	if diags.HasError() {
		return nil
	}
	created, err := r.client.CreateItemPrice(ctx, price)
	if chargebee.IsAlreadyExists(err) {
		created = r.adoptItemPrice(ctx, price, plan, diags)
		if diags.HasError() {
			return nil
		}
	} else if err != nil {
		diags.AddError("Failed to create item price", err.Error())
		return nil
	}

	if price.Status == chargebee.StatusArchived && created.Status != chargebee.StatusArchived {
		created, err = r.client.SetItemPriceStatus(ctx, price.ID, chargebee.StatusArchived)
		if err != nil {
			diags.AddError("Failed to set item price status after create", err.Error())
			return nil
		}
	}
	return created
}

// adoptItemPriceByID brings an existing Chargebee item price under management by
// id alone, reading every field from Chargebee and applying only a status
// transition. It never edits fields, so it is safe on archived prices. It errors
// if no price with the id exists, since without the identity fields there is
// nothing to create.
func (r *itemPriceResource) adoptItemPriceByID(ctx context.Context, plan itemPriceResourceModel, diags *diag.Diagnostics) *chargebee.ItemPrice {
	id := plan.ID.ValueString()
	existing, err := r.client.GetItemPrice(ctx, id)
	if chargebee.IsNotFound(err) {
		diags.AddError(
			"Cannot create item price",
			fmt.Sprintf("To create a new price, set item_id, name, currency_code, and pricing_model. To adopt an "+
				"existing price by id, that price must already exist in Chargebee; no price with id %q was found.", id),
		)
		return nil
	}
	if err != nil {
		diags.AddError("Failed to read existing item price for adoption", err.Error())
		return nil
	}

	created := existing
	if plan.Status.ValueString() == chargebee.StatusArchived && created.Status != chargebee.StatusArchived {
		created, err = r.client.SetItemPriceStatus(ctx, id, chargebee.StatusArchived)
		if err != nil {
			diags.AddError("Failed to set item price status after adoption", err.Error())
			return nil
		}
	}

	diags.AddWarning(
		"Adopted existing item price",
		fmt.Sprintf("Item price %q was adopted by id and is now managed by Terraform. "+
			"All fields were read from Chargebee.", id),
	)
	return created
}

// adoptItemPrice brings a pre-existing Chargebee item price under Terraform
// management: it errors if a create-only field (item_id, currency_code) differs
// from the configuration or if the update cannot reconcile the mutable fields,
// otherwise updates the mutable fields to match.
func (r *itemPriceResource) adoptItemPrice(ctx context.Context, plan chargebee.ItemPrice, planModel itemPriceResourceModel, diags *diag.Diagnostics) *chargebee.ItemPrice {
	existing, err := r.client.GetItemPrice(ctx, plan.ID)
	if err != nil {
		diags.AddError("Failed to read existing item price for adoption", err.Error())
		return nil
	}
	if conflicts := itemPriceAdoptionConflicts(plan, *existing); len(conflicts) > 0 {
		diags.Append(adoptionConflictError("item price", plan.ID, conflicts, createOnlyConflictNote))
		return nil
	}

	updated, err := r.client.UpdateItemPrice(ctx, plan)
	if err != nil {
		diags.AddError("Failed to adopt existing item price", err.Error())
		return nil
	}

	updatedModel, toDiags := itemPriceToModel(ctx, updated)
	diags.Append(toDiags...)
	if diags.HasError() {
		return nil
	}
	conflicts := unreconciledFields([]adoptionField{
		{"name", planModel.Name, updatedModel.Name},
		{"external_name", planModel.ExternalName, updatedModel.ExternalName},
		{"description", planModel.Description, updatedModel.Description},
		{"invoice_notes", planModel.InvoiceNotes, updatedModel.InvoiceNotes},
		{"pricing_model", planModel.PricingModel, updatedModel.PricingModel},
		{"price", planModel.Price, updatedModel.Price},
		{"period", planModel.Period, updatedModel.Period},
		{"period_unit", planModel.PeriodUnit, updatedModel.PeriodUnit},
		{"trial_period", planModel.TrialPeriod, updatedModel.TrialPeriod},
		{"trial_period_unit", planModel.TrialPeriodUnit, updatedModel.TrialPeriodUnit},
		{"billing_cycles", planModel.BillingCycles, updatedModel.BillingCycles},
		{"free_quantity", planModel.FreeQuantity, updatedModel.FreeQuantity},
		{"is_taxable", planModel.IsTaxable, updatedModel.IsTaxable},
		{"show_description_in_invoices", planModel.ShowDescriptionInInvoices, updatedModel.ShowDescriptionInInvoices},
		{"show_description_in_quotes", planModel.ShowDescriptionInQuotes, updatedModel.ShowDescriptionInQuotes},
	})
	if len(conflicts) > 0 {
		diags.Append(adoptionConflictError("item price", plan.ID, conflicts, unreconciledConflictNote))
		return nil
	}

	diags.AddWarning(
		"Adopted existing item price",
		fmt.Sprintf("Item price %q already existed in Chargebee and is now managed by Terraform. "+
			"Its mutable fields were updated to match this configuration.", plan.ID),
	)
	return updated
}

func (r *itemPriceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state itemPriceResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	price, err := r.client.GetItemPrice(ctx, state.ID.ValueString())
	if chargebee.IsNotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Failed to read item price", err.Error())
		return
	}
	if price.Status == chargebee.StatusDeleted {
		resp.State.RemoveResource(ctx)
		return
	}

	state, diags := itemPriceToModel(ctx, price)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *itemPriceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan itemPriceResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	price, fromDiags := itemPriceFromModel(ctx, plan)
	resp.Diagnostics.Append(fromDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	updated, err := r.applyItemPriceUpdate(ctx, price)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to update item price",
			err.Error()+"\n\nNote: Chargebee rejects changes to pricing attributes of item prices that are in use by subscriptions.",
		)
		return
	}

	state, toDiags := itemPriceToModel(ctx, updated)
	resp.Diagnostics.Append(toDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

// applyItemPriceUpdate reconciles field and status changes. Chargebee forbids
// field edits on archived prices and cascades archival from an item to its
// prices, so a status transition is issued on its own; when the price is already
// at the requested status the call is skipped so re-applies stay a no-op.
func (r *itemPriceResource) applyItemPriceUpdate(ctx context.Context, price chargebee.ItemPrice) (*chargebee.ItemPrice, error) {
	desired := price.Status
	if desired == "" {
		desired = chargebee.StatusActive
	}

	current, err := r.client.GetItemPrice(ctx, price.ID)
	if err != nil {
		return nil, err
	}

	if desired == chargebee.StatusArchived {
		if current.Status == chargebee.StatusArchived {
			return current, nil
		}
		return r.client.SetItemPriceStatus(ctx, price.ID, chargebee.StatusArchived)
	}

	if current.Status == chargebee.StatusArchived {
		if _, err := r.client.SetItemPriceStatus(ctx, price.ID, chargebee.StatusActive); err != nil {
			return nil, err
		}
	}
	return r.client.UpdateItemPrice(ctx, price)
}

func (r *itemPriceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state itemPriceResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := state.ID.ValueString()
	outcome, err := chargebee.DeleteWithArchiveFallback(ctx,
		func(ctx context.Context) error { return r.client.DeleteItemPrice(ctx, id) },
		func(ctx context.Context) error { return r.client.ArchiveItemPrice(ctx, id) },
	)
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete item price", err.Error())
		return
	}
	if outcome == chargebee.Archived {
		resp.Diagnostics.AddWarning(
			"Item price archived instead of deleted",
			fmt.Sprintf("Chargebee refused to delete item price %q because it is in use; it was archived instead. "+
				"The id remains reserved in Chargebee.", id),
		)
	}
}

func (r *itemPriceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func itemPriceFromModel(ctx context.Context, m itemPriceResourceModel) (chargebee.ItemPrice, diag.Diagnostics) {
	price := chargebee.ItemPrice{
		ID:                        m.ID.ValueString(),
		ItemID:                    m.ItemID.ValueString(),
		Name:                      m.Name.ValueString(),
		ExternalName:              m.ExternalName.ValueString(),
		Description:               m.Description.ValueString(),
		InvoiceNotes:              m.InvoiceNotes.ValueString(),
		CurrencyCode:              m.CurrencyCode.ValueString(),
		PricingModel:              m.PricingModel.ValueString(),
		Price:                     int64PtrOrNil(m.Price),
		Period:                    int32PtrOrNil(m.Period),
		PeriodUnit:                m.PeriodUnit.ValueString(),
		TrialPeriod:               int32PtrOrNil(m.TrialPeriod),
		TrialPeriodUnit:           m.TrialPeriodUnit.ValueString(),
		BillingCycles:             int32PtrOrNil(m.BillingCycles),
		FreeQuantity:              int32PtrOrNil(m.FreeQuantity),
		IsTaxable:                 boolPtrOrNil(m.IsTaxable),
		ShowDescriptionInInvoices: boolPtrOrNil(m.ShowDescriptionInInvoices),
		ShowDescriptionInQuotes:   boolPtrOrNil(m.ShowDescriptionInQuotes),
		Status:                    m.Status.ValueString(),
	}

	var diags diag.Diagnostics
	if !m.Tiers.IsNull() && !m.Tiers.IsUnknown() {
		var tiers []tierModel
		diags = m.Tiers.ElementsAs(ctx, &tiers, false)
		price.Tiers = make([]chargebee.Tier, 0, len(tiers))
		for _, t := range tiers {
			price.Tiers = append(price.Tiers, chargebee.Tier{
				StartingUnit: t.StartingUnit.ValueInt32(),
				EndingUnit:   t.EndingUnit.ValueInt32Pointer(),
				Price:        t.Price.ValueInt64(),
			})
		}
	}
	return price, diags
}

func itemPriceToModel(ctx context.Context, p *chargebee.ItemPrice) (itemPriceResourceModel, diag.Diagnostics) {
	m := itemPriceResourceModel{
		ID:                        types.StringValue(p.ID),
		ItemID:                    types.StringValue(p.ItemID),
		Name:                      types.StringValue(p.Name),
		ExternalName:              stringOrNull(p.ExternalName),
		Description:               stringOrNull(p.Description),
		InvoiceNotes:              stringOrNull(p.InvoiceNotes),
		CurrencyCode:              types.StringValue(p.CurrencyCode),
		PricingModel:              types.StringValue(p.PricingModel),
		Price:                     types.Int64PointerValue(p.Price),
		Period:                    types.Int32PointerValue(p.Period),
		PeriodUnit:                stringOrNull(p.PeriodUnit),
		TrialPeriod:               types.Int32PointerValue(p.TrialPeriod),
		TrialPeriodUnit:           stringOrNull(p.TrialPeriodUnit),
		BillingCycles:             types.Int32PointerValue(p.BillingCycles),
		FreeQuantity:              types.Int32PointerValue(p.FreeQuantity),
		IsTaxable:                 types.BoolPointerValue(p.IsTaxable),
		ShowDescriptionInInvoices: types.BoolPointerValue(p.ShowDescriptionInInvoices),
		ShowDescriptionInQuotes:   types.BoolPointerValue(p.ShowDescriptionInQuotes),
		Status:                    types.StringValue(p.Status),
	}

	if len(p.Tiers) == 0 {
		m.Tiers = types.ListNull(tierObjectType)
		return m, nil
	}
	tiers := make([]tierModel, 0, len(p.Tiers))
	for _, t := range p.Tiers {
		tiers = append(tiers, tierModel{
			StartingUnit: types.Int32Value(t.StartingUnit),
			EndingUnit:   types.Int32PointerValue(t.EndingUnit),
			Price:        types.Int64Value(t.Price),
		})
	}
	list, diags := types.ListValueFrom(ctx, tierObjectType, tiers)
	m.Tiers = list
	return m, diags
}

func int32PtrOrNil(v types.Int32) *int32 {
	if v.IsNull() || v.IsUnknown() {
		return nil
	}
	return v.ValueInt32Pointer()
}

func int64PtrOrNil(v types.Int64) *int64 {
	if v.IsNull() || v.IsUnknown() {
		return nil
	}
	return v.ValueInt64Pointer()
}

func boolPtrOrNil(v types.Bool) *bool {
	if v.IsNull() || v.IsUnknown() {
		return nil
	}
	return v.ValueBoolPointer()
}
