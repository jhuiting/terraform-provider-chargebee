// SPDX-License-Identifier: MPL-2.0

// Package chargebee wraps the chargebee-go SDK behind a small, testable
// surface with the provider's own domain types.
package chargebee

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand/v2"
	"net/http"
	"time"

	cb "github.com/chargebee/chargebee-go/v4"
)

// Client provides access to the Chargebee Product Catalog 2.0 API.
type Client struct {
	sdk *cb.Client
}

// requestTimeout bounds every API call. The SDK cannot attach a context to
// Retrieve/Delete calls (see the Get*/Delete* method comments), so a hard
// HTTP-client timeout is the only cancellation those requests get.
const requestTimeout = 60 * time.Second

// NewClient creates a Client for the given Chargebee site and API key.
func NewClient(site, apiKey string) *Client {
	cfg := cb.NewClientConfig(site, apiKey)
	cfg.HTTPClient = &http.Client{Timeout: requestTimeout}
	return &Client{sdk: cb.NewClient(cfg)}
}

// errEmptyResponse signals a 2xx response whose body lacked the expected
// object (e.g. 204 No Content); the SDK returns a nil object and a nil error.
var errEmptyResponse = errors.New("chargebee returned an empty response object")

func wrapErr(op string, err error) error {
	var cbErr *cb.Error
	if errors.As(err, &cbErr) {
		return fmt.Errorf("%s: %w", op, &APIError{
			HTTPStatus: cbErr.HTTPStatusCode,
			Code:       cbErr.APIErrorCode,
			Message:    cbErr.Msg,
		})
	}
	return fmt.Errorf("%s: %w", op, err)
}

// --- Item families ---

// CreateItemFamily creates a product family.
func (c *Client) CreateItemFamily(ctx context.Context, f ItemFamily) (*ItemFamily, error) {
	req := &cb.ItemFamilyCreateRequest{Id: f.ID, Name: f.Name, Description: f.Description}
	req.SetContext(ctx)
	res, err := c.sdk.ItemFamily.Create(req)
	if err != nil {
		return nil, wrapErr("create item family", err)
	}
	if res.ItemFamily == nil {
		return nil, fmt.Errorf("create item family: %w", errEmptyResponse)
	}
	return itemFamilyFromSDK(res.ItemFamily), nil
}

// GetItemFamily retrieves a product family by id. The SDK exposes no way to
// attach ctx to Retrieve/Delete calls; it is accepted for signature uniformity.
func (c *Client) GetItemFamily(ctx context.Context, id string) (*ItemFamily, error) {
	_ = ctx
	res, err := c.sdk.ItemFamily.Retrieve(id)
	if err != nil {
		return nil, wrapErr("get item family", err)
	}
	if res.ItemFamily == nil {
		return nil, fmt.Errorf("get item family: %w", errEmptyResponse)
	}
	return itemFamilyFromSDK(res.ItemFamily), nil
}

// UpdateItemFamily updates a product family's mutable fields.
func (c *Client) UpdateItemFamily(ctx context.Context, f ItemFamily) (*ItemFamily, error) {
	req := &cb.ItemFamilyUpdateRequest{Name: f.Name, Description: f.Description}
	req.SetContext(ctx)
	res, err := c.sdk.ItemFamily.Update(f.ID, req)
	if err != nil {
		return nil, wrapErr("update item family", err)
	}
	if res.ItemFamily == nil {
		return nil, fmt.Errorf("update item family: %w", errEmptyResponse)
	}
	return itemFamilyFromSDK(res.ItemFamily), nil
}

// DeleteItemFamily deletes a product family. The SDK exposes no way to
// attach ctx to Retrieve/Delete calls; it is accepted for signature uniformity.
func (c *Client) DeleteItemFamily(ctx context.Context, id string) error {
	_ = ctx
	if _, err := c.sdk.ItemFamily.Delete(id); err != nil {
		return wrapErr("delete item family", err)
	}
	return nil
}

func itemFamilyFromSDK(f *cb.ItemFamily) *ItemFamily {
	return &ItemFamily{
		ID:          f.Id,
		Name:        f.Name,
		Description: f.Description,
		Status:      string(f.Status),
	}
}

// --- Items ---

// CreateItem creates an item (plan, addon, or charge).
func (c *Client) CreateItem(ctx context.Context, i Item) (*Item, error) {
	req := &cb.ItemCreateRequest{
		Id:                 i.ID,
		Name:               i.Name,
		Type:               cb.ItemType(i.Type),
		ItemFamilyId:       i.ItemFamilyID,
		Description:        i.Description,
		ExternalName:       i.ExternalName,
		Unit:               i.Unit,
		RedirectUrl:        i.RedirectURL,
		UsageCalculation:   cb.ItemUsageCalculation(i.UsageCalculation),
		EnabledForCheckout: i.EnabledForCheckout,
		EnabledInPortal:    i.EnabledInPortal,
		IsShippable:        i.IsShippable,
		IsGiftable:         i.IsGiftable,
		Metadata:           metadataToSDK(i.Metadata),
	}
	if i.Metered {
		req.Metered = &i.Metered
	}
	req.SetContext(ctx)
	res, err := c.sdk.Item.Create(req)
	if err != nil {
		return nil, wrapErr("create item", err)
	}
	if res.Item == nil {
		return nil, fmt.Errorf("create item: %w", errEmptyResponse)
	}
	return itemFromSDK(res.Item)
}

// GetItem retrieves an item by id. The SDK exposes no way to
// attach ctx to Retrieve/Delete calls; it is accepted for signature uniformity.
func (c *Client) GetItem(ctx context.Context, id string) (*Item, error) {
	_ = ctx
	res, err := c.sdk.Item.Retrieve(id)
	if err != nil {
		return nil, wrapErr("get item", err)
	}
	if res.Item == nil {
		return nil, fmt.Errorf("get item: %w", errEmptyResponse)
	}
	return itemFromSDK(res.Item)
}

// UpdateItem updates an item's mutable fields.
func (c *Client) UpdateItem(ctx context.Context, i Item) (*Item, error) {
	req := &cb.ItemUpdateRequest{
		Name:               i.Name,
		Description:        i.Description,
		ExternalName:       i.ExternalName,
		Unit:               i.Unit,
		RedirectUrl:        i.RedirectURL,
		EnabledForCheckout: i.EnabledForCheckout,
		EnabledInPortal:    i.EnabledInPortal,
		IsShippable:        i.IsShippable,
		Metadata:           metadataToSDK(i.Metadata),
	}
	req.SetContext(ctx)
	res, err := c.sdk.Item.Update(i.ID, req)
	if err != nil {
		return nil, wrapErr("update item", err)
	}
	if res.Item == nil {
		return nil, fmt.Errorf("update item: %w", errEmptyResponse)
	}
	return itemFromSDK(res.Item)
}

// DeleteItem deletes an item. The SDK exposes no way to
// attach ctx to Retrieve/Delete calls; it is accepted for signature uniformity.
func (c *Client) DeleteItem(ctx context.Context, id string) error {
	_ = ctx
	if _, err := c.sdk.Item.Delete(id); err != nil {
		return wrapErr("delete item", err)
	}
	return nil
}

// SetItemStatus changes only an item's status (active or archived). Chargebee
// rejects field updates on archived items, so status transitions are sent on
// their own, without any other mutable fields.
func (c *Client) SetItemStatus(ctx context.Context, id, status string) (*Item, error) {
	req := &cb.ItemUpdateRequest{Status: cb.ItemStatus(status)}
	req.SetContext(ctx)
	res, err := c.sdk.Item.Update(id, req)
	if err != nil {
		return nil, wrapErr("set item status", err)
	}
	if res.Item == nil {
		return nil, fmt.Errorf("set item status: %w", errEmptyResponse)
	}
	return itemFromSDK(res.Item)
}

// ArchiveItem sets an item's status to archived.
func (c *Client) ArchiveItem(ctx context.Context, id string) error {
	_, err := c.SetItemStatus(ctx, id, string(cb.ItemStatusArchived))
	return err
}

func itemFromSDK(i *cb.Item) (*Item, error) {
	metadata, err := metadataFromSDK(i.Metadata)
	if err != nil {
		return nil, fmt.Errorf("item %s: %w", i.Id, err)
	}
	return &Item{
		ID:                 i.Id,
		ItemFamilyID:       i.ItemFamilyId,
		Type:               string(i.Type),
		Name:               i.Name,
		ExternalName:       i.ExternalName,
		Description:        i.Description,
		Unit:               i.Unit,
		RedirectURL:        i.RedirectUrl,
		Metered:            i.Metered,
		UsageCalculation:   string(i.UsageCalculation),
		EnabledForCheckout: ptr(i.EnabledForCheckout),
		EnabledInPortal:    ptr(i.EnabledInPortal),
		IsShippable:        ptr(i.IsShippable),
		IsGiftable:         ptr(i.IsGiftable),
		Metadata:           metadata,
		Status:             string(i.Status),
	}, nil
}

func metadataToSDK(m map[string]string) map[string]any {
	if len(m) == 0 {
		return nil
	}
	out := make(map[string]any, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

func metadataFromSDK(raw json.RawMessage) (map[string]string, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	var m map[string]string
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, fmt.Errorf("metadata contains non-string values, which this provider does not support: %w", err)
	}
	if len(m) == 0 {
		return nil, nil
	}
	return m, nil
}

// --- Item prices ---

// CreateItemPrice creates an item price.
func (c *Client) CreateItemPrice(ctx context.Context, p ItemPrice) (*ItemPrice, error) {
	req := &cb.ItemPriceCreateRequest{
		Id:                        p.ID,
		ItemId:                    p.ItemID,
		Name:                      p.Name,
		ExternalName:              p.ExternalName,
		Description:               p.Description,
		InvoiceNotes:              p.InvoiceNotes,
		CurrencyCode:              p.CurrencyCode,
		PricingModel:              cb.PricingModel(p.PricingModel),
		Price:                     p.Price,
		Tiers:                     tiersToSDKCreate(p.Tiers),
		Period:                    p.Period,
		PeriodUnit:                cb.ItemPricePeriodUnit(p.PeriodUnit),
		TrialPeriod:               p.TrialPeriod,
		TrialPeriodUnit:           cb.ItemPriceTrialPeriodUnit(p.TrialPeriodUnit),
		BillingCycles:             p.BillingCycles,
		FreeQuantity:              p.FreeQuantity,
		IsTaxable:                 p.IsTaxable,
		ShowDescriptionInInvoices: p.ShowDescriptionInInvoices,
		ShowDescriptionInQuotes:   p.ShowDescriptionInQuotes,
	}
	req.SetContext(ctx)
	res, err := c.sdk.ItemPrice.Create(req)
	if err != nil {
		return nil, wrapErr("create item price", err)
	}
	if res.ItemPrice == nil {
		return nil, fmt.Errorf("create item price: %w", errEmptyResponse)
	}
	return itemPriceFromSDK(res.ItemPrice), nil
}

// GetItemPrice retrieves an item price by id. The SDK exposes no way to
// attach ctx to Retrieve/Delete calls; it is accepted for signature uniformity.
func (c *Client) GetItemPrice(ctx context.Context, id string) (*ItemPrice, error) {
	_ = ctx
	res, err := c.sdk.ItemPrice.Retrieve(id)
	if err != nil {
		return nil, wrapErr("get item price", err)
	}
	if res.ItemPrice == nil {
		return nil, fmt.Errorf("get item price: %w", errEmptyResponse)
	}
	return itemPriceFromSDK(res.ItemPrice), nil
}

// UpdateItemPrice updates an item price's mutable fields.
func (c *Client) UpdateItemPrice(ctx context.Context, p ItemPrice) (*ItemPrice, error) {
	req := &cb.ItemPriceUpdateRequest{
		Name:                      p.Name,
		ExternalName:              p.ExternalName,
		Description:               p.Description,
		InvoiceNotes:              p.InvoiceNotes,
		PricingModel:              cb.PricingModel(p.PricingModel),
		Price:                     p.Price,
		Tiers:                     tiersToSDKUpdate(p.Tiers),
		Period:                    p.Period,
		PeriodUnit:                cb.ItemPricePeriodUnit(p.PeriodUnit),
		TrialPeriod:               p.TrialPeriod,
		TrialPeriodUnit:           cb.ItemPriceTrialPeriodUnit(p.TrialPeriodUnit),
		BillingCycles:             p.BillingCycles,
		FreeQuantity:              p.FreeQuantity,
		IsTaxable:                 p.IsTaxable,
		ShowDescriptionInInvoices: p.ShowDescriptionInInvoices,
		ShowDescriptionInQuotes:   p.ShowDescriptionInQuotes,
	}
	req.SetContext(ctx)
	res, err := c.sdk.ItemPrice.Update(p.ID, req)
	if err != nil {
		return nil, wrapErr("update item price", err)
	}
	if res.ItemPrice == nil {
		return nil, fmt.Errorf("update item price: %w", errEmptyResponse)
	}
	return itemPriceFromSDK(res.ItemPrice), nil
}

// DeleteItemPrice deletes an item price. The SDK exposes no way to
// attach ctx to Retrieve/Delete calls; it is accepted for signature uniformity.
func (c *Client) DeleteItemPrice(ctx context.Context, id string) error {
	_ = ctx
	if _, err := c.sdk.ItemPrice.Delete(id); err != nil {
		return wrapErr("delete item price", err)
	}
	return nil
}

// SetItemPriceStatus changes only an item price's status (active or archived).
// Chargebee rejects field updates on archived prices, so status transitions are
// sent on their own, without any other mutable fields.
func (c *Client) SetItemPriceStatus(ctx context.Context, id, status string) (*ItemPrice, error) {
	req := &cb.ItemPriceUpdateRequest{Status: cb.ItemPriceStatus(status)}
	req.SetContext(ctx)
	res, err := c.sdk.ItemPrice.Update(id, req)
	if err != nil {
		return nil, wrapErr("set item price status", err)
	}
	if res.ItemPrice == nil {
		return nil, fmt.Errorf("set item price status: %w", errEmptyResponse)
	}
	return itemPriceFromSDK(res.ItemPrice), nil
}

// ArchiveItemPrice sets an item price's status to archived.
func (c *Client) ArchiveItemPrice(ctx context.Context, id string) error {
	_, err := c.SetItemPriceStatus(ctx, id, string(cb.ItemPriceStatusArchived))
	return err
}

func itemPriceFromSDK(p *cb.ItemPrice) *ItemPrice {
	out := &ItemPrice{
		ID:                        p.Id,
		ItemID:                    p.ItemId,
		Name:                      p.Name,
		ExternalName:              p.ExternalName,
		Description:               p.Description,
		InvoiceNotes:              p.InvoiceNotes,
		CurrencyCode:              p.CurrencyCode,
		PricingModel:              string(p.PricingModel),
		Tiers:                     tiersFromSDK(p.Tiers),
		Period:                    fromInt32(p.Period),
		PeriodUnit:                string(p.PeriodUnit),
		TrialPeriod:               fromInt32(p.TrialPeriod),
		TrialPeriodUnit:           string(p.TrialPeriodUnit),
		BillingCycles:             fromInt32(p.BillingCycles),
		FreeQuantity:              ptr(p.FreeQuantity),
		IsTaxable:                 ptr(p.IsTaxable),
		ShowDescriptionInInvoices: ptr(p.ShowDescriptionInInvoices),
		ShowDescriptionInQuotes:   ptr(p.ShowDescriptionInQuotes),
		Status:                    string(p.Status),
	}
	switch out.PricingModel {
	case PricingModelTiered, PricingModelVolume, PricingModelStairstep:
		// price is not meaningful for tier-based models
	default:
		out.Price = ptr(p.Price)
	}
	return out
}

func tiersToSDKCreate(tiers []Tier) []*cb.ItemPriceCreateTier {
	if len(tiers) == 0 {
		return nil
	}
	out := make([]*cb.ItemPriceCreateTier, len(tiers))
	for i, t := range tiers {
		out[i] = &cb.ItemPriceCreateTier{
			StartingUnit: ptr(t.StartingUnit),
			Price:        ptr(t.Price),
		}
		if t.EndingUnit != nil {
			out[i].EndingUnit = t.EndingUnit
		}
	}
	return out
}

func tiersToSDKUpdate(tiers []Tier) []*cb.ItemPriceUpdateTier {
	if len(tiers) == 0 {
		return nil
	}
	out := make([]*cb.ItemPriceUpdateTier, len(tiers))
	for i, t := range tiers {
		out[i] = &cb.ItemPriceUpdateTier{
			StartingUnit: ptr(t.StartingUnit),
			Price:        ptr(t.Price),
		}
		if t.EndingUnit != nil {
			out[i].EndingUnit = t.EndingUnit
		}
	}
	return out
}

func tiersFromSDK(tiers []*cb.ItemPriceTier) []Tier {
	if len(tiers) == 0 {
		return nil
	}
	out := make([]Tier, len(tiers))
	for i, t := range tiers {
		out[i] = Tier{StartingUnit: t.StartingUnit, Price: t.Price}
		if t.EndingUnit != 0 {
			out[i].EndingUnit = ptr(t.EndingUnit)
		}
	}
	return out
}

// --- Attached items ---

// CreateAttachedItem attaches a child item to a parent plan item.
func (c *Client) CreateAttachedItem(ctx context.Context, a AttachedItem) (*AttachedItem, error) {
	req := &cb.AttachedItemCreateRequest{
		ItemId:        a.ItemID,
		Type:          cb.AttachedItemType(a.Type),
		ChargeOnEvent: cb.ChargeOnEvent(a.ChargeOnEvent),
		ChargeOnce:    a.ChargeOnce,
		Quantity:      a.Quantity,
		BillingCycles: a.BillingCycles,
	}
	req.SetContext(ctx)
	res, err := c.sdk.AttachedItem.Create(a.ParentItemID, req)
	if err != nil {
		return nil, wrapErr("create attached item", err)
	}
	if res.AttachedItem == nil {
		return nil, fmt.Errorf("create attached item: %w", errEmptyResponse)
	}
	return attachedItemFromSDK(res.AttachedItem), nil
}

// GetAttachedItem retrieves an attached item by its parent item id and id. The
// SDK exposes no way to attach ctx to Retrieve calls; it is accepted for
// signature uniformity.
func (c *Client) GetAttachedItem(ctx context.Context, parentItemID, id string) (*AttachedItem, error) {
	_ = ctx
	res, err := c.sdk.AttachedItem.Retrieve(id, &cb.AttachedItemRetrieveRequest{ParentItemId: parentItemID})
	if err != nil {
		return nil, wrapErr("get attached item", err)
	}
	if res.AttachedItem == nil {
		return nil, fmt.Errorf("get attached item: %w", errEmptyResponse)
	}
	return attachedItemFromSDK(res.AttachedItem), nil
}

// UpdateAttachedItem updates an attached item's mutable fields.
func (c *Client) UpdateAttachedItem(ctx context.Context, a AttachedItem) (*AttachedItem, error) {
	req := &cb.AttachedItemUpdateRequest{
		ParentItemId:  a.ParentItemID,
		Type:          cb.AttachedItemType(a.Type),
		ChargeOnEvent: cb.ChargeOnEvent(a.ChargeOnEvent),
		ChargeOnce:    a.ChargeOnce,
		Quantity:      a.Quantity,
		BillingCycles: a.BillingCycles,
	}
	req.SetContext(ctx)
	res, err := c.sdk.AttachedItem.Update(a.ID, req)
	if err != nil {
		return nil, wrapErr("update attached item", err)
	}
	if res.AttachedItem == nil {
		return nil, fmt.Errorf("update attached item: %w", errEmptyResponse)
	}
	return attachedItemFromSDK(res.AttachedItem), nil
}

// DeleteAttachedItem detaches an attached item. The SDK exposes no way to
// attach ctx to Delete calls; it is accepted for signature uniformity.
func (c *Client) DeleteAttachedItem(ctx context.Context, parentItemID, id string) error {
	_ = ctx
	if _, err := c.sdk.AttachedItem.Delete(id, &cb.AttachedItemDeleteRequest{ParentItemId: parentItemID}); err != nil {
		return wrapErr("delete attached item", err)
	}
	return nil
}

func attachedItemFromSDK(a *cb.AttachedItem) *AttachedItem {
	return &AttachedItem{
		ID:            a.Id,
		ParentItemID:  a.ParentItemId,
		ItemID:        a.ItemId,
		Type:          string(a.Type),
		ChargeOnEvent: string(a.ChargeOnEvent),
		ChargeOnce:    ptr(a.ChargeOnce),
		Quantity:      fromInt32(a.Quantity),
		BillingCycles: fromInt32(a.BillingCycles),
		Status:        string(a.Status),
	}
}

// --- Features ---

// CreateFeature creates a feature and activates it. Chargebee creates features
// in draft; an item cannot be granted a draft feature, so the feature is
// activated before it is returned.
func (c *Client) CreateFeature(ctx context.Context, f Feature) (*Feature, error) {
	req := &cb.FeatureCreateRequest{
		Id:          f.ID,
		Name:        f.Name,
		Description: f.Description,
		Type:        cb.FeatureType(f.Type),
		Unit:        f.Unit,
	}
	req.SetContext(ctx)
	res, err := c.sdk.Feature.Create(req)
	if err != nil {
		return nil, wrapErr("create feature", err)
	}
	if res.Feature == nil {
		return nil, fmt.Errorf("create feature: %w", errEmptyResponse)
	}
	return c.activateFeature(res.Feature)
}

// activateFeature moves a feature to active unless it already is. The SDK's
// activate call takes no request object and so cannot carry a context.
func (c *Client) activateFeature(f *cb.Feature) (*Feature, error) {
	if f.Status == cb.FeatureStatusActive {
		return featureFromSDK(f), nil
	}
	res, err := c.sdk.Feature.Activate(f.Id)
	if err != nil {
		return nil, wrapErr("activate feature", err)
	}
	if res.Feature == nil {
		return nil, fmt.Errorf("activate feature: %w", errEmptyResponse)
	}
	return featureFromSDK(res.Feature), nil
}

// GetFeature retrieves a feature by id. The SDK exposes no way to attach ctx to
// Retrieve calls; it is accepted for signature uniformity.
func (c *Client) GetFeature(ctx context.Context, id string) (*Feature, error) {
	_ = ctx
	res, err := c.sdk.Feature.Retrieve(id)
	if err != nil {
		return nil, wrapErr("get feature", err)
	}
	if res.Feature == nil {
		return nil, fmt.Errorf("get feature: %w", errEmptyResponse)
	}
	return featureFromSDK(res.Feature), nil
}

// UpdateFeature updates a feature's mutable fields.
func (c *Client) UpdateFeature(ctx context.Context, f Feature) (*Feature, error) {
	req := &cb.FeatureUpdateRequest{
		Name:        f.Name,
		Description: f.Description,
		Unit:        f.Unit,
	}
	req.SetContext(ctx)
	res, err := c.sdk.Feature.Update(f.ID, req)
	if err != nil {
		return nil, wrapErr("update feature", err)
	}
	if res.Feature == nil {
		return nil, fmt.Errorf("update feature: %w", errEmptyResponse)
	}
	return featureFromSDK(res.Feature), nil
}

// DeleteFeature deletes a feature. The SDK exposes no way to attach ctx to
// Delete calls; it is accepted for signature uniformity.
func (c *Client) DeleteFeature(ctx context.Context, id string) error {
	_ = ctx
	if _, err := c.sdk.Feature.Delete(id); err != nil {
		return wrapErr("delete feature", err)
	}
	return nil
}

// ArchiveFeature archives a feature, the fallback when Chargebee refuses to
// delete one still referenced by item entitlements.
func (c *Client) ArchiveFeature(ctx context.Context, id string) error {
	_ = ctx
	if _, err := c.sdk.Feature.Archive(id); err != nil {
		return wrapErr("archive feature", err)
	}
	return nil
}

func featureFromSDK(f *cb.Feature) *Feature {
	return &Feature{
		ID:          f.Id,
		Name:        f.Name,
		Description: f.Description,
		Type:        string(f.Type),
		Unit:        f.Unit,
		Status:      string(f.Status),
	}
}

// --- Item entitlements ---

// itemEntitlementPageLimit bounds a single list page when scanning an item's
// entitlements for a feature.
const itemEntitlementPageLimit int32 = 100

// entitlementConflictAttempts bounds how many extra times an entitlement upsert
// is retried after a transient conflict. Chargebee's Entitlements API serializes
// writes per entity, so two grants applied to the same item concurrently (as
// Terraform does within one apply) can return http 409 asking the caller to
// retry. These package vars let tests make retries instant.
var (
	entitlementConflictAttempts = 8
	entitlementConflictWait     = func(ctx context.Context, attempt int) error {
		// Exponential backoff plus random jitter: two grants racing on the same
		// entity would otherwise retry in lockstep and keep colliding.
		jitter := time.Duration(rand.N(250 * time.Millisecond)) //nolint:gosec // backoff jitter, not security-sensitive
		d := min(250*time.Millisecond<<attempt, 4*time.Second) + jitter
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(d):
			return nil
		}
	}
)

// UpsertItemEntitlement grants a feature to an item, or updates its value if the
// grant already exists. It uses Chargebee's Entitlements API (POST
// /entitlements), which grandfather-enabled sites require in place of the
// item-scoped upsert endpoint. The upsert is idempotent, so this doubles as the
// adoption path: an entitlement created outside Terraform is brought to the
// configured value on the first apply.
func (c *Client) UpsertItemEntitlement(ctx context.Context, e ItemEntitlement) (*ItemEntitlement, error) {
	req := &cb.EntitlementCreateRequest{
		Action: cb.ActionUpsert,
		Entitlements: []*cb.EntitlementCreateEntitlement{
			{
				EntityId:   e.ItemID,
				EntityType: cb.EntitlementEntityType(e.ItemType),
				FeatureId:  e.FeatureID,
				Value:      e.Value,
			},
		},
	}
	req.SetContext(ctx)
	res, err := c.sdk.Entitlement.Create(req)
	for attempt := 0; attempt < entitlementConflictAttempts && IsConflict(err); attempt++ {
		if waitErr := entitlementConflictWait(ctx, attempt); waitErr != nil {
			return nil, waitErr
		}
		res, err = c.sdk.Entitlement.Create(req)
	}
	if err != nil {
		return nil, wrapErr("upsert item entitlement", err)
	}
	for _, entry := range res.List {
		if entry.Entitlement != nil && entry.Entitlement.FeatureId == e.FeatureID {
			return itemEntitlementFromSDK(entry.Entitlement), nil
		}
	}
	return nil, fmt.Errorf("upsert item entitlement: %w", errEmptyResponse)
}

// GetItemEntitlement retrieves a single feature grant on an item. It returns a
// not-found APIError (see IsNotFound) when the item has no grant for the
// feature, so callers can treat a removed grant like any other missing object.
func (c *Client) GetItemEntitlement(ctx context.Context, itemID, featureID string) (*ItemEntitlement, error) {
	offset := ""
	for {
		req := &cb.EntitlementListRequest{
			Limit:     ptr(itemEntitlementPageLimit),
			Offset:    offset,
			EntityId:  &cb.StringFilter{Is: itemID},
			FeatureId: &cb.StringFilter{Is: featureID},
		}
		req.SetContext(ctx)
		res, err := c.sdk.Entitlement.List(req)
		if err != nil {
			return nil, wrapErr("get item entitlement", err)
		}
		for _, entry := range res.List {
			if entry.Entitlement != nil && entry.Entitlement.FeatureId == featureID {
				return itemEntitlementFromSDK(entry.Entitlement), nil
			}
		}
		if res.NextOffset == "" {
			return nil, &APIError{HTTPStatus: http.StatusNotFound, Code: "resource_not_found",
				Message: fmt.Sprintf("item %q has no entitlement for feature %q", itemID, featureID)}
		}
		offset = res.NextOffset
	}
}

// RemoveItemEntitlement revokes a feature grant from an item. Removing a grant
// that is already absent is a no-op on Chargebee's side.
func (c *Client) RemoveItemEntitlement(ctx context.Context, e ItemEntitlement) error {
	req := &cb.EntitlementCreateRequest{
		Action: cb.ActionRemove,
		Entitlements: []*cb.EntitlementCreateEntitlement{
			{
				EntityId:   e.ItemID,
				EntityType: cb.EntitlementEntityType(e.ItemType),
				FeatureId:  e.FeatureID,
			},
		},
	}
	req.SetContext(ctx)
	_, err := c.sdk.Entitlement.Create(req)
	for attempt := 0; attempt < entitlementConflictAttempts && IsConflict(err); attempt++ {
		if waitErr := entitlementConflictWait(ctx, attempt); waitErr != nil {
			return waitErr
		}
		_, err = c.sdk.Entitlement.Create(req)
	}
	if err != nil {
		return wrapErr("remove item entitlement", err)
	}
	return nil
}

func itemEntitlementFromSDK(e *cb.Entitlement) *ItemEntitlement {
	return &ItemEntitlement{
		ItemID:    e.EntityId,
		ItemType:  string(e.EntityType),
		FeatureID: e.FeatureId,
		Value:     e.Value,
	}
}

func ptr[T any](v T) *T { return &v }

// fromInt32 maps Chargebee's 0 for unset optional integers (period,
// trial_period, billing_cycles) to nil. free_quantity is NOT routed through
// this: its 0 is a real server default that must round-trip.
func fromInt32(v int32) *int32 {
	if v == 0 {
		return nil
	}
	return ptr(v)
}
