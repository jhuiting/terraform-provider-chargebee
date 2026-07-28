// SPDX-License-Identifier: MPL-2.0

package provider_test

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"

	"github.com/jhuiting/terraform-provider-chargebee/internal/provider"
)

// testAccProtoV6ProviderFactories instantiates the provider under test.
var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"chargebee": providerserver.NewProtocol6WithError(provider.New("test")()),
}

// testAccPreCheck verifies the environment is set up for acceptance tests.
func testAccPreCheck(t *testing.T) {
	t.Helper()
	for _, v := range []string{"CHARGEBEE_SITE", "CHARGEBEE_API_KEY"} {
		if os.Getenv(v) == "" {
			t.Fatalf("%s must be set for acceptance tests (use a Chargebee TEST site)", v)
		}
	}
}

// testAccRandomID returns a unique lowercase id with the given prefix, so
// parallel or repeated test runs do not collide on Chargebee ids.
func testAccRandomID(prefix string) string {
	return prefix + "-" + acctest.RandStringFromCharSet(8, acctest.CharSetAlpha)
}
