package provider

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"hindclaw": providerserver.NewProtocol6WithError(New("test")()),
}

// testAccPreCheck validates that required env vars are set before running
// acceptance tests. Called in every test's PreCheck to fail fast with a
// clear message instead of bubbling up provider-config errors.
func testAccPreCheck(t *testing.T) {
	t.Helper()
	if os.Getenv("HINDCLAW_API_URL") == "" {
		t.Fatal("HINDCLAW_API_URL must be set for acceptance tests")
	}
	if os.Getenv("HINDCLAW_API_KEY") == "" {
		t.Fatal("HINDCLAW_API_KEY must be set for acceptance tests")
	}
}
