package okta

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
)

func deleteMfaPolicies(client *testClient) error {
	return deletePolicyByType(mfaPolicyType, client)
}

func TestAccOktaMfaPolicy(t *testing.T) {
	ri := acctest.RandInt()
	config := testOktaMfaPolicy(ri)
	updatedConfig := testOktaMfaPolicyUpdated(ri)
	resourceName := buildResourceFQN(mfaPolicy, ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: createPolicyCheckDestroy(mfaPolicy),
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					ensurePolicyExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", buildResourceName(ri)),
					resource.TestCheckResourceAttr(resourceName, "status", "ACTIVE"),
					resource.TestCheckResourceAttr(resourceName, "description", "Terraform Acceptance Test MFA Policy"),
					resource.TestCheckResourceAttr(resourceName, "google_otp.enroll", "REQUIRED"),
				),
			},
			{
				Config: updatedConfig,
				Check: resource.ComposeTestCheckFunc(
					ensurePolicyExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", buildResourceName(ri)),
					resource.TestCheckResourceAttr(resourceName, "status", "INACTIVE"),
					resource.TestCheckResourceAttr(resourceName, "description", "Terraform Acceptance Test MFA Policy Updated"),
					resource.TestCheckResourceAttr(resourceName, "fido_u2f.enroll", "OPTIONAL"),
					resource.TestCheckResourceAttr(resourceName, "fido_webauthn.enroll", "OPTIONAL"),
					resource.TestCheckResourceAttr(resourceName, "google_otp.enroll", "OPTIONAL"),
					resource.TestCheckResourceAttr(resourceName, "okta_otp.enroll", "OPTIONAL"),
					resource.TestCheckResourceAttr(resourceName, "okta_sms.enroll", "OPTIONAL"),
				),
			},
		},
	})
}

func testOktaMfaPolicy(rInt int) string {
	name := buildResourceName(rInt)

	return fmt.Sprintf(`
resource "okta_factor" "google" {
	provider_id = "google_otp" 
}
resource "%s" "%s" {
  name        		= "%s"
  status      		= "ACTIVE"
  description 		= "Terraform Acceptance Test MFA Policy"
  google_otp = {
	  enroll = "REQUIRED"
  }
  depends_on = [
	"okta_factor.google",
  ]	
}
`, mfaPolicy, name, name)
}

func testOktaMfaPolicyUpdated(rInt int) string {
	name := buildResourceName(rInt)

	return fmt.Sprintf(`
data "okta_everyone_group" "everyone-%d" {}
resource "okta_factor" "google" {
	provider_id = "google_otp" 
}
resource "okta_factor" "sms" {
	provider_id = "okta_sms" 
}
resource "okta_factor" "otp" {
	provider_id = "okta_otp" 
}
resource "okta_factor" "fido_u2f" {
	provider_id = "fido_u2f" 
}
resource "okta_factor" "fido_webauthn" {
	provider_id = "fido_webauthn" 
}

resource "%s" "%s" {
	name        = "%s"
	status      = "INACTIVE"
	description = "Terraform Acceptance Test MFA Policy Updated"
	groups_included = [ "${data.okta_everyone_group.everyone-%d.id}" ]
	fido_u2f 		= {
		enroll = "OPTIONAL"
	}
	fido_webauthn 	= {
		enroll = "OPTIONAL"
	}
	google_otp	 	= {
		enroll = "OPTIONAL"
	}
	okta_otp 		= {
		enroll = "OPTIONAL"
	}
	okta_sms 		= {
		enroll = "OPTIONAL"
	}
	depends_on = [
		"okta_factor.google",
		"okta_factor.sms",
		"okta_factor.otp",
		"okta_factor.fido_u2f",
		"okta_factor.fido_webauthn",
	]
}
`, rInt, mfaPolicy, name, name, rInt)
}
