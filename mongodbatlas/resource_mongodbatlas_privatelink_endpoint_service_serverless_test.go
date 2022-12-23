package mongodbatlas

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccNetworkRSPrivateLinkEndpointServiceServerless_basic(t *testing.T) {
	var (
		resourceName  = "mongodbatlas_privatelink_endpoint_service_serverless.test"
		projectID     = os.Getenv("MONGODB_ATLAS_PROJECT_ID")
		instanceName  = "serverlesssrv"
		commentOrigin = "this is a comment for serverless private link endpoint"
	)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckMongoDBAtlasPrivateLinkEndpointServiceServerlessDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccMongoDBAtlasPrivateLinkEndpointServiceServerlessConfig(projectID, instanceName, commentOrigin),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckMongoDBAtlasPrivateLinkEndpointServiceServerlessExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "provider_name", "AWS"),
					resource.TestCheckResourceAttr(resourceName, "comment", commentOrigin),
				),
			},
		},
	})
}

func TestAccNetworkRSPrivateLinkEndpointServiceServerless_importBasic(t *testing.T) {
	var (
		resourceName  = "mongodbatlas_privatelink_endpoint_service_serverless.test"
		projectID     = os.Getenv("MONGODB_ATLAS_PROJECT_ID")
		instanceName  = "serverlesssrvimport"
		commentOrigin = "this is a comment for serverless private link endpoint"
	)
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckMongoDBAtlasSearchIndexDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccMongoDBAtlasPrivateLinkEndpointServiceServerlessConfig(projectID, instanceName, commentOrigin),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "provider_name", "AWS"),
					resource.TestCheckResourceAttr(resourceName, "comment", commentOrigin),
				),
			},
			{
				Config:            testAccMongoDBAtlasPrivateLinkEndpointServiceServerlessConfig(projectID, instanceName, commentOrigin),
				ResourceName:      resourceName,
				ImportStateIdFunc: testAccCheckMongoDBAtlasPrivateLinkEndpointServiceServerlessImportStateIDFunc(resourceName),
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccCheckMongoDBAtlasPrivateLinkEndpointServiceServerlessDestroy(state *terraform.State) error {
	conn := testAccProvider.Meta().(*MongoDBClient).Atlas

	for _, rs := range state.RootModule().Resources {
		if rs.Type != "mongodbatlas_privatelink_endpoint_service_serverless" {
			continue
		}

		ids := decodeStateID(rs.Primary.ID)

		privateLink, _, err := conn.ServerlessPrivateEndpoints.Get(context.Background(), ids["project_id"], ids["instance_name"], ids["endpoint_id"])
		if err == nil && privateLink != nil {
			return fmt.Errorf("endpoint_id (%s) still exists", ids["endpoint_id"])
		}
	}

	return nil
}

func testAccMongoDBAtlasPrivateLinkEndpointServiceServerlessConfig(projectID, instanceName, comment string) string {
	return fmt.Sprintf(`

	resource "mongodbatlas_privatelink_endpoint_serverless" "test" {
		project_id   = "%[1]s"
		instance_name = mongodbatlas_serverless_instance.test.name
		provider_name = "AWS"
	}


	resource "mongodbatlas_privatelink_endpoint_service_serverless" "test" {
		project_id   = mongodbatlas_privatelink_endpoint_serverless.test.project_id
		instance_name = mongodbatlas_privatelink_endpoint_serverless.test.instance_name
		endpoint_id = mongodbatlas_privatelink_endpoint_serverless.test.endpoint_id
		provider_name = "AWS"
		comment = "%[3]s"
	}

	resource "mongodbatlas_serverless_instance" "test" {
		project_id   = "%[1]s"
		name         = "%[2]s"
		provider_settings_backing_provider_name = "AWS"
		provider_settings_provider_name = "SERVERLESS"
		provider_settings_region_name = "US_EAST_1"
		continuous_backup_enabled = true

		lifecycle {
			ignore_changes = [connection_strings_private_endpoint_srv]
		}
	}

	data "mongodbatlas_serverless_instance" "test" {
		project_id   = mongodbatlas_privatelink_endpoint_service_serverless.test.project_id
		name         = mongodbatlas_serverless_instance.test.name
	}

	`, projectID, instanceName, comment)
}

func testAccCheckMongoDBAtlasPrivateLinkEndpointServiceServerlessExists(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		conn := testAccProvider.Meta().(*MongoDBClient).Atlas

		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("not found: %s", resourceName)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("no ID is set")
		}

		ids := decodeStateID(rs.Primary.ID)

		_, _, err := conn.ServerlessPrivateEndpoints.Get(context.Background(), ids["project_id"], ids["instance_name"], ids["endpoint_id"])
		if err == nil {
			return nil
		}

		return fmt.Errorf("endpoint_id (%s) does not exist", ids["endpoint_id"])
	}
}

func testAccCheckMongoDBAtlasPrivateLinkEndpointServiceServerlessImportStateIDFunc(resourceName string) resource.ImportStateIdFunc {
	return func(s *terraform.State) (string, error) {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return "", fmt.Errorf("not found: %s", resourceName)
		}

		ids := decodeStateID(rs.Primary.ID)

		return fmt.Sprintf("%s--%s--%s", ids["project_id"], ids["instance_name"], ids["endpoint_id"]), nil
	}
}