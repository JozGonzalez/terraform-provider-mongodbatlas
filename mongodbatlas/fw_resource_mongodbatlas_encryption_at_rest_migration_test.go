package mongodbatlas

import (
	"os"
	"testing"

	matlas "go.mongodb.org/atlas/mongodbatlas"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/mwielbut/pointy"
)

func TestAccAdvRS_Migration_EncryptionAtRest_basicAWS(t *testing.T) {
	SkipTestExtCred(t)
	var (
		resourceName = "mongodbatlas_encryption_at_rest.test"
		projectID    = os.Getenv("MONGODB_ATLAS_PROJECT_ID")

		awsKms = matlas.AwsKms{
			Enabled:             pointy.Bool(true),
			CustomerMasterKeyID: os.Getenv("AWS_CUSTOMER_MASTER_KEY_ID"),
			Region:              os.Getenv("AWS_REGION"),
			RoleID:              os.Getenv("AWS_ROLE_ID"),
		}
	)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t); testCheckAwsEnv(t) },
		CheckDestroy: testAccCheckMongoDBAtlasEncryptionAtRestDestroy,
		Steps: []resource.TestStep{
			{
				ExternalProviders: map[string]resource.ExternalProvider{
					"mongodbatlas": {
						VersionConstraint: "1.11.0",
						Source:            "mongodb/mongodbatlas",
					},
				},
				Config: testAccMongoDBAtlasEncryptionAtRestConfigAwsKms(projectID, &awsKms),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckMongoDBAtlasEncryptionAtRestExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "project_id", projectID),
					resource.TestCheckResourceAttr(resourceName, "aws_kms_config.0.enabled", "true"),
					resource.TestCheckResourceAttr(resourceName, "aws_kms_config.0.region", awsKms.Region),
					resource.TestCheckResourceAttr(resourceName, "aws_kms_config.0.role_id", awsKms.RoleID),
				),
			},
			{
				ProtoV6ProviderFactories: testAccProviderV6Factories,
				Config:                   testAccMongoDBAtlasEncryptionAtRestConfigAwsKms(projectID, &awsKms),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPreRefresh: []plancheck.PlanCheck{
						DebugPlan(),
					},
				},
				PlanOnly: true,
			},
		},
	})
}

func TestAccAdvRS_Migration_EncryptionAtRest_WithRole_basicAWS(t *testing.T) {
	SkipTest(t)
	SkipTestExtCred(t)
	var (
		resourceName = "mongodbatlas_encryption_at_rest.test"
		projectID    = os.Getenv("MONGODB_ATLAS_PROJECT_ID")
		accessKeyID  = os.Getenv("AWS_ACCESS_KEY_ID")
		secretKey    = os.Getenv("AWS_SECRET_ACCESS_KEY")
		policyName   = acctest.RandomWithPrefix("test-aws-policy")
		roleName     = acctest.RandomWithPrefix("test-aws-role")

		awsKms = matlas.AwsKms{
			Enabled:             pointy.Bool(true),
			CustomerMasterKeyID: os.Getenv("AWS_CUSTOMER_MASTER_KEY_ID"),
			Region:              os.Getenv("AWS_REGION"),
		}
	)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t); testCheckAwsEnv(t) },
		CheckDestroy: testAccCheckMongoDBAtlasEncryptionAtRestDestroy,
		Steps: []resource.TestStep{
			{
				ExternalProviders: map[string]resource.ExternalProvider{
					"mongodbatlas": {
						VersionConstraint: "1.11.0",
						Source:            "mongodb/mongodbatlas",
					},
					"aws": {
						VersionConstraint: "5.1.0",
						Source:            "hashicorp/aws",
					},
				},
				Config: testAccMongoDBAtlasEncryptionAtRestConfigAwsKmsWithRole(awsKms.Region, accessKeyID, secretKey, projectID, policyName, roleName, false, &awsKms),
			},
			{
				ExternalProviders: map[string]resource.ExternalProvider{
					"aws": {
						VersionConstraint: "5.1.0",
						Source:            "hashicorp/aws",
					},
				},
				ProtoV6ProviderFactories: testAccProviderV6Factories,
				Config:                   testAccMongoDBAtlasEncryptionAtRestConfigAwsKmsWithRole(awsKms.Region, accessKeyID, secretKey, projectID, policyName, roleName, false, &awsKms),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckMongoDBAtlasEncryptionAtRestExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "project_id", projectID),
					resource.TestCheckResourceAttr(resourceName, "aws_kms_config.0.enabled", "true"),
					resource.TestCheckResourceAttr(resourceName, "aws_kms_config.0.region", awsKms.Region),
					resource.TestCheckResourceAttr(resourceName, "aws_kms_config.0.role_id", awsKms.RoleID),
				),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPreRefresh: []plancheck.PlanCheck{
						DebugPlan(),
					},
				},
				PlanOnly: true,
			},
		},
	})
}

func TestAccAdvRS_Migration_EncryptionAtRest_basicAzure(t *testing.T) {
	SkipTestExtCred(t)
	var (
		resourceName = "mongodbatlas_encryption_at_rest.test"
		projectID    = os.Getenv("MONGODB_ATLAS_PROJECT_ID")

		azureKeyVault = matlas.AzureKeyVault{
			Enabled:           pointy.Bool(true),
			ClientID:          os.Getenv("AZURE_CLIENT_ID"),
			AzureEnvironment:  "AZURE",
			SubscriptionID:    os.Getenv("AZURE_SUBSCRIPTION_ID"),
			ResourceGroupName: os.Getenv("AZURE_RESOURCE_GROUP_NAME"),
			KeyVaultName:      os.Getenv("AZURE_KEY_VAULT_NAME"),
			KeyIdentifier:     os.Getenv("AZURE_KEY_IDENTIFIER"),
			Secret:            os.Getenv("AZURE_SECRET"),
			TenantID:          os.Getenv("AZURE_TENANT_ID"),
		}
	)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t); testCheckEncryptionAtRestEnvAzure(t) },
		CheckDestroy: testAccCheckMongoDBAtlasEncryptionAtRestDestroy,
		Steps: []resource.TestStep{
			{
				ExternalProviders: map[string]resource.ExternalProvider{
					"mongodbatlas": {
						VersionConstraint: "1.11.0",
						Source:            "mongodb/mongodbatlas",
					},
				},
				Config: testAccMongoDBAtlasEncryptionAtRestConfigAzureKeyVault(projectID, &azureKeyVault),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckMongoDBAtlasEncryptionAtRestExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "project_id", projectID),
					resource.TestCheckResourceAttr(resourceName, "azure_key_vault_config.0.enabled", "true"),
					resource.TestCheckResourceAttr(resourceName, "azure_key_vault_config.0.azure_environment", azureKeyVault.AzureEnvironment),
					resource.TestCheckResourceAttr(resourceName, "azure_key_vault_config.0.resource_group_name", azureKeyVault.ResourceGroupName),
					resource.TestCheckResourceAttr(resourceName, "azure_key_vault_config.0.key_vault_name", azureKeyVault.KeyVaultName),
				),
			},
			{
				ProtoV6ProviderFactories: testAccProviderV6Factories,
				Config:                   testAccMongoDBAtlasEncryptionAtRestConfigAzureKeyVault(projectID, &azureKeyVault),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPreRefresh: []plancheck.PlanCheck{
						DebugPlan(),
					},
				},
				PlanOnly: true,
			},
		},
	})
}

func TestAccAdvRS_Migration_EncryptionAtRest_basicGCP(t *testing.T) {
	SkipTestExtCred(t)
	var (
		resourceName = "mongodbatlas_encryption_at_rest.test"
		projectID    = os.Getenv("MONGODB_ATLAS_PROJECT_ID")

		googleCloudKms = matlas.GoogleCloudKms{
			Enabled:              pointy.Bool(true),
			ServiceAccountKey:    os.Getenv("GCP_SERVICE_ACCOUNT_KEY"),
			KeyVersionResourceID: os.Getenv("GCP_KEY_VERSION_RESOURCE_ID"),
		}
	)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t); testAccPreCheckGPCEnv(t) },
		CheckDestroy: testAccCheckMongoDBAtlasEncryptionAtRestDestroy,
		Steps: []resource.TestStep{
			{
				ExternalProviders: map[string]resource.ExternalProvider{
					"mongodbatlas": {
						VersionConstraint: "1.11.0",
						Source:            "mongodb/mongodbatlas",
					},
				},
				Config: testAccMongoDBAtlasEncryptionAtRestConfigGoogleCloudKms(projectID, &googleCloudKms),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckMongoDBAtlasEncryptionAtRestExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "project_id", projectID),
					resource.TestCheckResourceAttr(resourceName, "google_cloud_kms_config.0.enabled", "true"),
				),
			},
			{
				ProtoV6ProviderFactories: testAccProviderV6Factories,
				Config:                   testAccMongoDBAtlasEncryptionAtRestConfigGoogleCloudKms(projectID, &googleCloudKms),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPreRefresh: []plancheck.PlanCheck{
						DebugPlan(),
					},
				},
				PlanOnly: true,
			},
		},
	})
}
