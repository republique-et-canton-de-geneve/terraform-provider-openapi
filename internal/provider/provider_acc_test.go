package provider

import (
	"fmt"
	"net/http"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// TestMain guards acceptance tests: when TF_ACC=1 and the server is unreachable it exits
// with code 1 and a helpful message rather than letting New() crash mid-test.
// In normal usage 'make testacc' starts the server before running tests, so this guard
// only fires when go test is invoked directly without a server running.
func TestMain(m *testing.M) {
	if os.Getenv("TF_ACC") == "1" {
		baseURL := envOr("OPENAPI_URL", "http://localhost:8000")
		resp, err := http.Get(baseURL + "/health/")
		if err != nil || resp.StatusCode != http.StatusOK {
			fmt.Fprintf(os.Stderr,
				"testacc: server not reachable at %s/health/ — run 'make testacc' or start the server first\n",
				baseURL)
			os.Exit(1)
		}
	}
	os.Exit(m.Run())
}

// testAccProviderFactories is lazily evaluated: New reads OPENAPI_SPEC and calls os.Exit(1)
// if it is missing, so we must not call it until the factory is actually invoked
// (i.e. inside resource.Test, after TestMain has confirmed the server is up).
var testAccProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"openapi": func() (tfprotov6.ProviderServer, error) {
		return providerserver.NewProtocol6WithError(New("test")())()
	},
}

func widgetConfig(name string, size int) string {
	return fmt.Sprintf(`
provider "openapi" {
  url = %q
}

resource "openapi_widget" "test" {
  name = %q
  size = %d
}
`, os.Getenv("OPENAPI_URL"), name, size)
}

func widgetConfigNoSize(name string) string {
	return fmt.Sprintf(`
provider "openapi" {
  url = %q
}

resource "openapi_widget" "test" {
  name = %q
}
`, os.Getenv("OPENAPI_URL"), name)
}

// TestAccWidget_basic creates a widget and verifies all fields including server-computed ones.
func TestAccWidget_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: widgetConfig("alpha", 3),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("openapi_widget.test", "name", "alpha"),
					resource.TestCheckResourceAttr("openapi_widget.test", "size", "3"),
					resource.TestCheckResourceAttrSet("openapi_widget.test", "id"),
					resource.TestCheckResourceAttrSet("openapi_widget.test", "created_at"),
				),
			},
		},
	})
}

// TestAccWidget_update creates a widget then patches name and size in a second step.
func TestAccWidget_update(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: widgetConfig("alpha", 3),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("openapi_widget.test", "name", "alpha"),
					resource.TestCheckResourceAttr("openapi_widget.test", "size", "3"),
				),
			},
			{
				Config: widgetConfig("beta", 7),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("openapi_widget.test", "name", "beta"),
					resource.TestCheckResourceAttr("openapi_widget.test", "size", "7"),
				),
			},
		},
	})
}

// TestAccWidget_disappears deletes the resource out-of-band and verifies the plan is non-empty.
func TestAccWidget_disappears(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:             widgetConfig("alpha", 3),
				Check:              testAccDeleteWidget("openapi_widget.test"),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func testAccDeleteWidget(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource %s not found in state", resourceName)
		}
		url := fmt.Sprintf("%s/api/v1/widgets/%s/", os.Getenv("OPENAPI_URL"), rs.Primary.ID)
		req, err := http.NewRequest(http.MethodDelete, url, nil)
		if err != nil {
			return err
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusNoContent {
			return fmt.Errorf("DELETE %s returned %d", url, resp.StatusCode)
		}
		return nil
	}
}

// TestAccWidget_importState verifies that an existing widget can be imported by ID.
func TestAccWidget_importState(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: widgetConfig("alpha", 3),
			},
			{
				ResourceName:      "openapi_widget.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

// TestAccWidget_nullSize creates a widget without size and verifies the attribute stays null.
func TestAccWidget_nullSize(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: widgetConfigNoSize("gamma"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("openapi_widget.test", "name", "gamma"),
					resource.TestCheckNoResourceAttr("openapi_widget.test", "size"),
				),
			},
		},
	})
}
