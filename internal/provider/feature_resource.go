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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/jhuiting/terraform-provider-chargebee/internal/chargebee"
)

var (
	_ resource.Resource                   = (*featureResource)(nil)
	_ resource.ResourceWithConfigure      = (*featureResource)(nil)
	_ resource.ResourceWithImportState    = (*featureResource)(nil)
	_ resource.ResourceWithValidateConfig = (*featureResource)(nil)
)

// NewFeatureResource returns the chargebee_feature resource.
func NewFeatureResource() resource.Resource {
	return &featureResource{}
}

type featureResource struct {
	client *chargebee.Client
}

type featureResourceModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	Type        types.String `tfsdk:"type"`
	Unit        types.String `tfsdk:"unit"`
	Status      types.String `tfsdk:"status"`
}

func (r *featureResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_feature"
}

func (r *featureResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "A Chargebee entitlement feature. Created features are activated automatically so items can be " +
			"granted them via chargebee_item_entitlement. " +
			"If a feature with the given id already exists in Chargebee, it is adopted into state and its mutable " +
			"fields are updated to match this configuration.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Required:      true,
				Description:   "Unique identifier of the feature. Changing this forces a new resource.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Display name of the feature.",
			},
			"description": schema.StringAttribute{
				Optional:    true,
				Description: "Description of the feature.",
			},
			"type": schema.StringAttribute{
				Optional: true,
				Computed: true,
				Description: "Feature type: `switch` (the default), `custom`, `quantity`, or `range`. " +
					"Changing this forces a new resource.",
				Default:       stringdefault.StaticString(chargebee.FeatureTypeSwitch),
				Validators:    []validator.String{stringvalidator.OneOf("switch", "custom", "quantity", "range")},
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"unit": schema.StringAttribute{
				Optional:    true,
				Description: "Unit label for quantity or range features.",
			},
			"status": schema.StringAttribute{
				Computed:      true,
				Description:   "Status of the feature in Chargebee (e.g. `active`).",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
		},
	}
}

func (r *featureResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var config featureResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if config.Unit.IsNull() || config.Unit.IsUnknown() || config.Type.IsUnknown() {
		return
	}
	featureType := config.Type.ValueString()
	if config.Type.IsNull() {
		featureType = chargebee.FeatureTypeSwitch
	}
	if featureType != chargebee.FeatureTypeQuantity && featureType != chargebee.FeatureTypeRange {
		resp.Diagnostics.AddAttributeError(
			path.Root("unit"),
			"unit requires a quantity or range feature",
			"The unit attribute is only valid for features of type quantity or range.",
		)
	}
}

func (r *featureResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = clientFromProviderData(req.ProviderData, &resp.Diagnostics)
}

func (r *featureResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan featureResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	feature := featureFromModel(plan)
	created, err := r.client.CreateFeature(ctx, feature)
	if chargebee.IsAlreadyExists(err) {
		created = r.adoptFeature(ctx, feature, plan, &resp.Diagnostics)
		if resp.Diagnostics.HasError() {
			return
		}
	} else if err != nil {
		resp.Diagnostics.AddError("Failed to create feature", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, featureToModel(created))...)
}

// adoptFeature brings a pre-existing Chargebee feature under Terraform
// management. It updates the feature's mutable fields only when they differ from
// the configuration: adopting a feature that already matches must not re-send
// its name, which Chargebee's unique-name check rejects even for the feature's
// own name. It errors if the result still differs from the plan.
func (r *featureResource) adoptFeature(ctx context.Context, plan chargebee.Feature, planModel featureResourceModel, diags *diag.Diagnostics) *chargebee.Feature {
	existing, err := r.client.GetFeature(ctx, plan.ID)
	if err != nil {
		diags.AddError("Failed to read existing feature for adoption", err.Error())
		return nil
	}
	if conflicts := featureAdoptionConflicts(plan, *existing); len(conflicts) > 0 {
		diags.Append(adoptionConflictError("feature", plan.ID, conflicts, createOnlyConflictNote))
		return nil
	}

	updated := existing
	if existing.Name != plan.Name || existing.Description != plan.Description || existing.Unit != plan.Unit {
		updated, err = r.client.UpdateFeature(ctx, plan)
		if err != nil {
			diags.AddError("Failed to adopt existing feature", err.Error())
			return nil
		}
	}

	updatedModel := featureToModel(updated)
	conflicts := unreconciledFields([]adoptionField{
		{"name", planModel.Name, updatedModel.Name},
		{"description", planModel.Description, updatedModel.Description},
		{"unit", planModel.Unit, updatedModel.Unit},
	})
	if len(conflicts) > 0 {
		diags.Append(adoptionConflictError("feature", plan.ID, conflicts, unreconciledConflictNote))
		return nil
	}

	diags.AddWarning(
		"Adopted existing feature",
		fmt.Sprintf("Feature %q already existed in Chargebee and is now managed by Terraform. "+
			"Its mutable fields were updated to match this configuration.", plan.ID),
	)
	return updated
}

func (r *featureResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state featureResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	feature, err := r.client.GetFeature(ctx, state.ID.ValueString())
	if chargebee.IsNotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Failed to read feature", err.Error())
		return
	}
	if feature.Status == chargebee.StatusDeleted {
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, featureToModel(feature))...)
}

func (r *featureResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan featureResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	updated, err := r.client.UpdateFeature(ctx, featureFromModel(plan))
	if err != nil {
		resp.Diagnostics.AddError("Failed to update feature", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, featureToModel(updated))...)
}

func (r *featureResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state featureResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := state.ID.ValueString()

	// Chargebee only deletes features that are draft or archived, so an active
	// feature is archived first. An already-archived feature makes this a no-op,
	// and a draft feature is deletable as-is, so its archive rejection is ignored.
	err := r.client.ArchiveFeature(ctx, id)
	if err != nil && !chargebee.IsNotFound(err) && !chargebee.IsAlreadyArchived(err) && !chargebee.IsDraftState(err) {
		resp.Diagnostics.AddError("Failed to archive feature before delete", err.Error())
		return
	}

	// An in-use refusal is retried with backoff: entitlement removals from the
	// same destroy may still be propagating in Chargebee. The nil archive
	// fallback is deliberate — the feature is already archived at this point,
	// so a feature that stays in use is simply left that way.
	outcome, err := chargebee.DeleteWithArchiveFallback(ctx,
		func(ctx context.Context) error { return r.client.DeleteFeature(ctx, id) },
		nil,
	)
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete feature", err.Error())
		return
	}
	if outcome == chargebee.LeftInPlace {
		resp.Diagnostics.AddWarning(
			"Feature archived instead of deleted",
			fmt.Sprintf("Chargebee refused to delete feature %q because it is in use, so it was left archived.", id),
		)
	}
}

func (r *featureResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func featureFromModel(m featureResourceModel) chargebee.Feature {
	return chargebee.Feature{
		ID:          m.ID.ValueString(),
		Name:        m.Name.ValueString(),
		Description: m.Description.ValueString(),
		Type:        m.Type.ValueString(),
		Unit:        m.Unit.ValueString(),
	}
}

func featureToModel(f *chargebee.Feature) featureResourceModel {
	return featureResourceModel{
		ID:          types.StringValue(f.ID),
		Name:        types.StringValue(f.Name),
		Description: stringOrNull(f.Description),
		Type:        types.StringValue(f.Type),
		Unit:        stringOrNull(f.Unit),
		Status:      types.StringValue(f.Status),
	}
}
