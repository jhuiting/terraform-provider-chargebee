// SPDX-License-Identifier: MPL-2.0

package chargebee

import (
	"context"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	cb "github.com/chargebee/chargebee-go/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// rewriteTransport sends every SDK request to the test server, regardless of
// the chargebee.com host the SDK builds.
type rewriteTransport struct {
	target *url.URL
}

func (t rewriteTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.URL.Scheme = t.target.Scheme
	req.URL.Host = t.target.Host
	return http.DefaultTransport.RoundTrip(req)
}

func newTestClient(t *testing.T, handler http.Handler) *Client {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	target, err := url.Parse(srv.URL)
	require.NoError(t, err)

	cfg := cb.NewClientConfig("test-site", "test-key")
	cfg.HTTPClient = &http.Client{Transport: rewriteTransport{target: target}}
	return &Client{sdk: cb.NewClient(cfg)}
}

func TestClientErrorMapping(t *testing.T) {
	tests := []struct {
		name       string
		status     int
		body       string
		call       func(c *Client) error
		wantInUse  bool
		wantIsGone bool
	}{
		{
			name:   "delete in use",
			status: http.StatusBadRequest,
			body: `{"message":"Cannot delete an item that has subscriptions.",
				"type":"invalid_request","api_error_code":"invalid_state_for_request","http_status_code":400}`,
			call: func(c *Client) error {
				return c.DeleteItem(context.Background(), "standard")
			},
			wantInUse: true,
		},
		{
			name:   "retrieve not found",
			status: http.StatusNotFound,
			body:   `{"message":"item_family not found","type":"invalid_request","api_error_code":"resource_not_found","http_status_code":404}`,
			call: func(c *Client) error {
				_, err := c.GetItemFamily(context.Background(), "missing")
				return err
			},
			wantIsGone: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.status)
				_, _ = w.Write([]byte(tt.body))
			}))

			err := tt.call(client)
			require.Error(t, err)
			assert.Equal(t, tt.wantInUse, IsInUse(err), "IsInUse")
			assert.Equal(t, tt.wantIsGone, IsNotFound(err), "IsNotFound")
		})
	}
}

func TestClientEmptyResponseObject(t *testing.T) {
	// The SDK returns a nil inner object with a nil error for a 2xx response
	// whose body lacks the expected key; the client must error, not panic.
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{}`))
	}))

	tests := []struct {
		name string
		call func() error
	}{
		{"get item family", func() error { _, err := client.GetItemFamily(context.Background(), "x"); return err }},
		{"create item", func() error { _, err := client.CreateItem(context.Background(), Item{ID: "x"}); return err }},
		{"update item price", func() error { _, err := client.UpdateItemPrice(context.Background(), ItemPrice{ID: "x"}); return err }},
		{"create attached item", func() error {
			_, err := client.CreateAttachedItem(context.Background(), AttachedItem{ParentItemID: "p", ItemID: "x"})
			return err
		}},
		{"get attached item", func() error { _, err := client.GetAttachedItem(context.Background(), "p", "x"); return err }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.call()
			require.Error(t, err)
			assert.ErrorIs(t, err, errEmptyResponse)
		})
	}
}

func TestClientNonStringMetadata(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"item":{"id":"x","name":"X","metadata":{"count":5}}}`))
	}))

	_, err := client.GetItem(context.Background(), "x")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "metadata contains non-string values")
}

func TestClientCreateItemPriceRoundTrip(t *testing.T) {
	var gotPath, gotAuthHeader string
	var gotForm url.Values

	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotAuthHeader = r.Header.Get("Authorization")
		require.NoError(t, r.ParseForm())
		gotForm = r.PostForm

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"item_price":{
			"id":"api-calls-eur","item_id":"api-calls","name":"api-calls-eur",
			"currency_code":"EUR","pricing_model":"tiered","status":"active",
			"period":1,"period_unit":"month","free_quantity":0,"is_taxable":true,
			"tiers":[
				{"starting_unit":1,"ending_unit":100000,"price":0},
				{"starting_unit":100001,"price":1}
			]}}`))
	}))

	endingUnit := int32(100000)
	created, err := client.CreateItemPrice(context.Background(), ItemPrice{
		ID:           "api-calls-eur",
		ItemID:       "api-calls",
		Name:         "api-calls-eur",
		CurrencyCode: "EUR",
		PricingModel: "tiered",
		Period:       ptr(int32(1)),
		PeriodUnit:   "month",
		Tiers: []Tier{
			{StartingUnit: 1, EndingUnit: &endingUnit, Price: 0},
			{StartingUnit: 100001, Price: 1},
		},
	})
	require.NoError(t, err)

	// Request assertions: form-encoded params reached the API.
	assert.Equal(t, "/api/v2/item_prices", gotPath)
	// The SDK sends "Basic base64(<api key>)" (no username:password colon).
	assert.Equal(t, "Basic "+base64.StdEncoding.EncodeToString([]byte("test-key")), gotAuthHeader)
	assert.Equal(t, "api-calls-eur", gotForm.Get("id"))
	assert.Equal(t, "api-calls", gotForm.Get("item_id"))
	assert.Equal(t, "tiered", gotForm.Get("pricing_model"))
	hasTierParam := false
	for key := range gotForm {
		if strings.HasPrefix(key, "tiers[") {
			hasTierParam = true
			break
		}
	}
	assert.True(t, hasTierParam, "expected tier params in form body, got keys: %v", gotForm)

	// Response assertions: domain conversion.
	assert.Nil(t, created.Price, "price must be nil for tier-based models")
	require.Len(t, created.Tiers, 2)
	require.NotNil(t, created.Tiers[0].EndingUnit)
	assert.Equal(t, int32(100000), *created.Tiers[0].EndingUnit)
	assert.Nil(t, created.Tiers[1].EndingUnit, "unbounded final tier")
	require.NotNil(t, created.Period)
	assert.Equal(t, int32(1), *created.Period)
	require.NotNil(t, created.FreeQuantity)
	assert.Equal(t, int32(0), *created.FreeQuantity)
	assert.Equal(t, "active", created.Status)
}

func TestClientCreateAttachedItemRoundTrip(t *testing.T) {
	var gotPath string
	var gotForm url.Values

	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		require.NoError(t, r.ParseForm())
		gotForm = r.PostForm

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"attached_item":{
			"id":"attached-123","parent_item_id":"eyecyberguard-v3","item_id":"EYE-IMPLEMENTATION-v3",
			"type":"recommended","charge_on_event":"subscription_creation","charge_once":true,
			"quantity":1,"status":"active"}}`))
	}))

	created, err := client.CreateAttachedItem(context.Background(), AttachedItem{
		ParentItemID:  "eyecyberguard-v3",
		ItemID:        "EYE-IMPLEMENTATION-v3",
		ChargeOnEvent: "subscription_creation",
	})
	require.NoError(t, err)

	assert.Equal(t, "/api/v2/items/eyecyberguard-v3/attached_items", gotPath)
	assert.Equal(t, "EYE-IMPLEMENTATION-v3", gotForm.Get("item_id"))
	assert.Equal(t, "subscription_creation", gotForm.Get("charge_on_event"))

	assert.Equal(t, "attached-123", created.ID)
	assert.Equal(t, "eyecyberguard-v3", created.ParentItemID)
	assert.Equal(t, "EYE-IMPLEMENTATION-v3", created.ItemID)
	assert.Equal(t, "recommended", created.Type)
	assert.Equal(t, "subscription_creation", created.ChargeOnEvent)
	require.NotNil(t, created.ChargeOnce)
	assert.True(t, *created.ChargeOnce)
	require.NotNil(t, created.Quantity)
	assert.Equal(t, int32(1), *created.Quantity)
	assert.Equal(t, "active", created.Status)
}
