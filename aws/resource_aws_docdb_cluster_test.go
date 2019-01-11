package aws

import (
	"errors"
	"fmt"
	"log"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/docdb"
)

func TestAccAWSDocDBCluster_importBasic(t *testing.T) {
	resourceName := "aws_docdb_cluster.default"
	ri := acctest.RandInt()

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckDocDBClusterDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccDocDBClusterConfig(ri),
			},

			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"master_password", "skip_final_snapshot"},
			},
		},
	})
}

func TestAccAWSDocDBCluster_basic(t *testing.T) {
	var dbCluster docdb.DBCluster
	rInt := acctest.RandInt()
	resourceName := "aws_docdb_cluster.default"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckDocDBClusterDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccDocDBClusterConfig(rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDocDBClusterExists(resourceName, &dbCluster),
					resource.TestMatchResourceAttr(resourceName, "arn", regexp.MustCompile(`^arn:[^:]+:docdb:[^:]+:\d{12}:cluster:.+`)),
					resource.TestCheckResourceAttr(resourceName, "storage_encrypted", "false"),
					resource.TestCheckResourceAttr(resourceName, "db_cluster_parameter_group_name", "default.docdb3.6"),
					resource.TestCheckResourceAttrSet(resourceName, "reader_endpoint"),
					resource.TestCheckResourceAttrSet(resourceName, "cluster_resource_id"),
					resource.TestCheckResourceAttr(resourceName, "engine", "docdb"),
					resource.TestCheckResourceAttrSet(resourceName, "engine_version"),
					resource.TestCheckResourceAttrSet(resourceName, "hosted_zone_id"),
					resource.TestCheckResourceAttr(resourceName,
						"enabled_cloudwatch_logs_exports.0", "audit"),
					resource.TestCheckResourceAttr(resourceName,
						"enabled_cloudwatch_logs_exports.1", "error"),
				),
			},
		},
	})
}

func TestAccAWSDocDBCluster_namePrefix(t *testing.T) {
	var v docdb.DBCluster

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckDocDBClusterDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccDocDBClusterConfig_namePrefix(acctest.RandInt()),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDocDBClusterExists("aws_docdb_cluster.test", &v),
					resource.TestMatchResourceAttr(
						"aws_docdb_cluster.test", "cluster_identifier", regexp.MustCompile("^tf-test-")),
				),
			},
		},
	})
}

func TestAccAWSDocDBCluster_generatedName(t *testing.T) {
	var v docdb.DBCluster

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckDocDBClusterDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccDocDBClusterConfig_generatedName(acctest.RandInt()),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDocDBClusterExists("aws_docdb_cluster.test", &v),
					resource.TestMatchResourceAttr(
						"aws_docdb_cluster.test", "cluster_identifier", regexp.MustCompile("^tf-")),
				),
			},
		},
	})
}

func TestAccAWSDocDBCluster_takeFinalSnapshot(t *testing.T) {
	var v docdb.DBCluster
	rInt := acctest.RandInt()

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckDocDBClusterSnapshot(rInt),
		Steps: []resource.TestStep{
			{
				Config: testAccDocDBClusterConfigWithFinalSnapshot(rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDocDBClusterExists("aws_docdb_cluster.default", &v),
				),
			},
		},
	})
}

/// This is a regression test to make sure that we always cover the scenario as hightlighted in
/// https://github.com/hashicorp/terraform/issues/11568
func TestAccAWSDocDBCluster_missingUserNameCausesError(t *testing.T) {
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckDocDBClusterDestroy,
		Steps: []resource.TestStep{
			{
				Config:      testAccDocDBClusterConfigWithoutUserNameAndPassword(acctest.RandInt()),
				ExpectError: regexp.MustCompile(`required field is not set`),
			},
		},
	})
}

func TestAccAWSDocDBCluster_updateTags(t *testing.T) {
	var v docdb.DBCluster
	ri := acctest.RandInt()

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckDocDBClusterDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccDocDBClusterConfig(ri),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDocDBClusterExists("aws_docdb_cluster.default", &v),
					resource.TestCheckResourceAttr(
						"aws_docdb_cluster.default", "tags.%", "1"),
				),
			},
			{
				Config: testAccDocDBClusterConfigUpdatedTags(ri),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDocDBClusterExists("aws_docdb_cluster.default", &v),
					resource.TestCheckResourceAttr(
						"aws_docdb_cluster.default", "tags.%", "2"),
				),
			},
		},
	})
}

func TestAccAWSDocDBCluster_updateCloudwatchLogsExports(t *testing.T) {
	var v docdb.DBCluster
	ri := acctest.RandInt()

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckDocDBClusterDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccDocDBClusterConfig(ri),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDocDBClusterExists("aws_docdb_cluster.default", &v),
					resource.TestCheckResourceAttr("aws_docdb_cluster.default",
						"enabled_cloudwatch_logs_exports.0", "audit"),
					resource.TestCheckResourceAttr("aws_docdb_cluster.default",
						"enabled_cloudwatch_logs_exports.1", "error"),
				),
			},
			{
				Config: testAccDocDBClusterConfigUpdatedCloudwatchLogsExports(ri),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDocDBClusterExists("aws_docdb_cluster.default", &v),
					resource.TestCheckResourceAttr("aws_docdb_cluster.default",
						"enabled_cloudwatch_logs_exports.0", "error"),
					resource.TestCheckResourceAttr("aws_docdb_cluster.default",
						"enabled_cloudwatch_logs_exports.1", "slowquery"),
				),
			},
		},
	})
}

func TestAccAWSDocDBCluster_kmsKey(t *testing.T) {
	var v docdb.DBCluster
	keyRegex := regexp.MustCompile("^arn:aws:kms:")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckDocDBClusterDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccDocDBClusterConfig_kmsKey(acctest.RandInt()),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDocDBClusterExists("aws_docdb_cluster.default", &v),
					resource.TestMatchResourceAttr(
						"aws_docdb_cluster.default", "kms_key_id", keyRegex),
				),
			},
		},
	})
}

func TestAccAWSDocDBCluster_encrypted(t *testing.T) {
	var v docdb.DBCluster

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckDocDBClusterDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccDocDBClusterConfig_encrypted(acctest.RandInt()),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDocDBClusterExists("aws_docdb_cluster.default", &v),
					resource.TestCheckResourceAttr(
						"aws_docdb_cluster.default", "storage_encrypted", "true"),
					resource.TestCheckResourceAttr(
						"aws_docdb_cluster.default", "db_cluster_parameter_group_name", "default.docdb3.6"),
				),
			},
		},
	})
}

func TestAccAWSDocDBCluster_backupsUpdate(t *testing.T) {
	var v docdb.DBCluster

	ri := acctest.RandInt()
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckDocDBClusterDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccDocDBClusterConfig_backups(ri),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDocDBClusterExists("aws_docdb_cluster.default", &v),
					resource.TestCheckResourceAttr(
						"aws_docdb_cluster.default", "preferred_backup_window", "07:00-09:00"),
					resource.TestCheckResourceAttr(
						"aws_docdb_cluster.default", "backup_retention_period", "5"),
					resource.TestCheckResourceAttr(
						"aws_docdb_cluster.default", "preferred_maintenance_window", "tue:04:00-tue:04:30"),
				),
			},

			{
				Config: testAccDocDBClusterConfig_backupsUpdate(ri),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDocDBClusterExists("aws_docdb_cluster.default", &v),
					resource.TestCheckResourceAttr(
						"aws_docdb_cluster.default", "preferred_backup_window", "03:00-09:00"),
					resource.TestCheckResourceAttr(
						"aws_docdb_cluster.default", "backup_retention_period", "10"),
					resource.TestCheckResourceAttr(
						"aws_docdb_cluster.default", "preferred_maintenance_window", "wed:01:00-wed:01:30"),
				),
			},
		},
	})
}

func TestAccAWSDocDBCluster_Port(t *testing.T) {
	var dbCluster1, dbCluster2 docdb.DBCluster
	rInt := acctest.RandInt()
	resourceName := "aws_docdb_cluster.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckDocDBClusterDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccDocDBClusterConfig_Port(rInt, 5432),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDocDBClusterExists(resourceName, &dbCluster1),
					resource.TestCheckResourceAttr(resourceName, "port", "5432"),
				),
			},
			{
				Config: testAccDocDBClusterConfig_Port(rInt, 2345),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDocDBClusterExists(resourceName, &dbCluster2),
					testAccCheckDocDBClusterRecreated(&dbCluster1, &dbCluster2),
					resource.TestCheckResourceAttr(resourceName, "port", "2345"),
				),
			},
		},
	})
}

func TestAccAWSDocDBCluster_SnapshotIdentifier(t *testing.T) {
	var dbCluster, sourceDocDBCluster docdb.DBCluster
	var dbClusterSnapshot docdb.DBClusterSnapshot

	rName := acctest.RandomWithPrefix("tf-acc-test")
	sourceDbResourceName := "aws_docdb_cluster.source"
	snapshotResourceName := "aws_db_cluster_snapshot.test"
	resourceName := "aws_docdb_cluster.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSDBInstanceDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSDocDBClusterConfig_SnapshotIdentifier(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDocDBClusterExists(sourceDbResourceName, &sourceDocDBCluster),
					testAccCheckDocDBClusterSnapshotExists(snapshotResourceName, &dbClusterSnapshot),
					testAccCheckDocDBClusterExists(resourceName, &dbCluster),
				),
			},
		},
	})
}

// Reference: https://github.com/terraform-providers/terraform-provider-aws/issues/6157
func TestAccAWSDocDBCluster_SnapshotIdentifier_EngineVersion_Different(t *testing.T) {
	var dbCluster, sourceDocDBCluster docdb.DBCluster
	var dbClusterSnapshot docdb.DBClusterSnapshot

	rName := acctest.RandomWithPrefix("tf-acc-test")
	sourceDbResourceName := "aws_docdb_cluster.source"
	snapshotResourceName := "aws_db_cluster_snapshot.test"
	resourceName := "aws_docdb_cluster.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSDBInstanceDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSDocDBClusterConfig_SnapshotIdentifier_EngineVersion(rName, "docdb", "3.6.0", "3.6.1"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDocDBClusterExists(sourceDbResourceName, &sourceDocDBCluster),
					testAccCheckDocDBClusterSnapshotExists(snapshotResourceName, &dbClusterSnapshot),
					testAccCheckDocDBClusterExists(resourceName, &dbCluster),
					resource.TestCheckResourceAttr(resourceName, "engine_version", "3.6.1"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"apply_immediately",
					"cluster_identifier_prefix",
					"master_password",
					"skip_final_snapshot",
					"snapshot_identifier",
				},
			},
		},
	})
}

// Reference: https://github.com/terraform-providers/terraform-provider-aws/issues/6157
func TestAccAWSDocDBCluster_SnapshotIdentifier_EngineVersion_Equal(t *testing.T) {
	var dbCluster, sourceDocDBCluster docdb.DBCluster
	var dbClusterSnapshot docdb.DBClusterSnapshot

	rName := acctest.RandomWithPrefix("tf-acc-test")
	sourceDbResourceName := "aws_docdb_cluster.source"
	snapshotResourceName := "aws_db_cluster_snapshot.test"
	resourceName := "aws_docdb_cluster.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSDBInstanceDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSDocDBClusterConfig_SnapshotIdentifier_EngineVersion(rName, "docdb", "3.6.0", "3.6.0"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDocDBClusterExists(sourceDbResourceName, &sourceDocDBCluster),
					testAccCheckDocDBClusterSnapshotExists(snapshotResourceName, &dbClusterSnapshot),
					testAccCheckDocDBClusterExists(resourceName, &dbCluster),
					resource.TestCheckResourceAttr(resourceName, "engine_version", "3.6.0"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"apply_immediately",
					"cluster_identifier_prefix",
					"master_password",
					"skip_final_snapshot",
					"snapshot_identifier",
				},
			},
		},
	})
}

func TestAccAWSDocDBCluster_SnapshotIdentifier_PreferredBackupWindow(t *testing.T) {
	var dbCluster, sourceDocDBCluster docdb.DBCluster
	var dbClusterSnapshot docdb.DBClusterSnapshot

	rName := acctest.RandomWithPrefix("tf-acc-test")
	sourceDbResourceName := "aws_docdb_cluster.source"
	snapshotResourceName := "aws_db_cluster_snapshot.test"
	resourceName := "aws_docdb_cluster.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSDBInstanceDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSDocDBClusterConfig_SnapshotIdentifier_PreferredBackupWindow(rName, "00:00-08:00"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDocDBClusterExists(sourceDbResourceName, &sourceDocDBCluster),
					testAccCheckDocDBClusterSnapshotExists(snapshotResourceName, &dbClusterSnapshot),
					testAccCheckDocDBClusterExists(resourceName, &dbCluster),
					resource.TestCheckResourceAttr(resourceName, "preferred_backup_window", "00:00-08:00"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"apply_immediately",
					"cluster_identifier_prefix",
					"master_password",
					"skip_final_snapshot",
					"snapshot_identifier",
				},
			},
		},
	})
}

func TestAccAWSDocDBCluster_SnapshotIdentifier_PreferredMaintenanceWindow(t *testing.T) {
	var dbCluster, sourceDocDBCluster docdb.DBCluster
	var dbClusterSnapshot docdb.DBClusterSnapshot

	rName := acctest.RandomWithPrefix("tf-acc-test")
	sourceDbResourceName := "aws_docdb_cluster.source"
	snapshotResourceName := "aws_db_cluster_snapshot.test"
	resourceName := "aws_docdb_cluster.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSDBInstanceDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSDocDBClusterConfig_SnapshotIdentifier_PreferredMaintenanceWindow(rName, "sun:01:00-sun:01:30"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDocDBClusterExists(sourceDbResourceName, &sourceDocDBCluster),
					testAccCheckDocDBClusterSnapshotExists(snapshotResourceName, &dbClusterSnapshot),
					testAccCheckDocDBClusterExists(resourceName, &dbCluster),
					resource.TestCheckResourceAttr(resourceName, "preferred_maintenance_window", "sun:01:00-sun:01:30"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"apply_immediately",
					"cluster_identifier_prefix",
					"master_password",
					"skip_final_snapshot",
					"snapshot_identifier",
				},
			},
		},
	})
}

func TestAccAWSDocDBCluster_SnapshotIdentifier_Tags(t *testing.T) {
	var dbCluster, sourceDocDBCluster docdb.DBCluster
	var dbClusterSnapshot docdb.DBClusterSnapshot

	rName := acctest.RandomWithPrefix("tf-acc-test")
	sourceDbResourceName := "aws_docdb_cluster.source"
	snapshotResourceName := "aws_db_cluster_snapshot.test"
	resourceName := "aws_docdb_cluster.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSDBInstanceDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSDocDBClusterConfig_SnapshotIdentifier_Tags(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDocDBClusterExists(sourceDbResourceName, &sourceDocDBCluster),
					testAccCheckDocDBClusterSnapshotExists(snapshotResourceName, &dbClusterSnapshot),
					testAccCheckDocDBClusterExists(resourceName, &dbCluster),
					resource.TestCheckResourceAttr(resourceName, "tags.%", "1"),
					resource.TestCheckResourceAttr(resourceName, "tags.key1", "value1"),
				),
			},
		},
	})
}

func TestAccAWSDocDBCluster_SnapshotIdentifier_VpcSecurityGroupIds(t *testing.T) {
	var dbCluster, sourceDocDBCluster docdb.DBCluster
	var dbClusterSnapshot docdb.DBClusterSnapshot

	rName := acctest.RandomWithPrefix("tf-acc-test")
	sourceDbResourceName := "aws_docdb_cluster.source"
	snapshotResourceName := "aws_db_cluster_snapshot.test"
	resourceName := "aws_docdb_cluster.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSDBInstanceDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSDocDBClusterConfig_SnapshotIdentifier_VpcSecurityGroupIds(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDocDBClusterExists(sourceDbResourceName, &sourceDocDBCluster),
					testAccCheckDocDBClusterSnapshotExists(snapshotResourceName, &dbClusterSnapshot),
					testAccCheckDocDBClusterExists(resourceName, &dbCluster),
				),
			},
		},
	})
}

// Regression reference: https://github.com/terraform-providers/terraform-provider-aws/issues/5450
// This acceptance test explicitly tests when snapshot_identifer is set,
// vpc_security_group_ids is set (which triggered the resource update function),
// and tags is set which was missing its ARN used for tagging
func TestAccAWSDocDBCluster_SnapshotIdentifier_VpcSecurityGroupIds_Tags(t *testing.T) {
	var dbCluster, sourceDocDBCluster docdb.DBCluster
	var dbClusterSnapshot docdb.DBClusterSnapshot

	rName := acctest.RandomWithPrefix("tf-acc-test")
	sourceDbResourceName := "aws_docdb_cluster.source"
	snapshotResourceName := "aws_db_cluster_snapshot.test"
	resourceName := "aws_docdb_cluster.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSDBInstanceDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSDocDBClusterConfig_SnapshotIdentifier_VpcSecurityGroupIds_Tags(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDocDBClusterExists(sourceDbResourceName, &sourceDocDBCluster),
					testAccCheckDocDBClusterSnapshotExists(snapshotResourceName, &dbClusterSnapshot),
					testAccCheckDocDBClusterExists(resourceName, &dbCluster),
					resource.TestCheckResourceAttr(resourceName, "tags.%", "1"),
					resource.TestCheckResourceAttr(resourceName, "tags.key1", "value1"),
				),
			},
		},
	})
}

func TestAccAWSDocDBCluster_SnapshotIdentifier_EncryptedRestore(t *testing.T) {
	var dbCluster, sourceDocDBCluster docdb.DBCluster
	var dbClusterSnapshot docdb.DBClusterSnapshot

	keyRegex := regexp.MustCompile("^arn:aws:kms:")

	rName := acctest.RandomWithPrefix("tf-acc-test")
	sourceDbResourceName := "aws_docdb_cluster.source"
	snapshotResourceName := "aws_db_cluster_snapshot.test"
	resourceName := "aws_docdb_cluster.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSDBInstanceDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSDocDBClusterConfig_SnapshotIdentifier_EncryptedRestore(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDocDBClusterExists(sourceDbResourceName, &sourceDocDBCluster),
					testAccCheckDocDBClusterSnapshotExists(snapshotResourceName, &dbClusterSnapshot),
					testAccCheckDocDBClusterExists(resourceName, &dbCluster),
					resource.TestMatchResourceAttr(
						"aws_docdb_cluster.test", "kms_key_id", keyRegex),
					resource.TestCheckResourceAttr(
						"aws_docdb_cluster.test", "storage_encrypted", "true"),
				),
			},
		},
	})
}

func testAccCheckDocDBClusterDestroy(s *terraform.State) error {
	return testAccCheckDocDBClusterDestroyWithProvider(s, testAccProvider)
}

func testAccCheckDocDBClusterDestroyWithProvider(s *terraform.State, provider *schema.Provider) error {
	conn := provider.Meta().(*AWSClient).docdbconn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_docdb_cluster" {
			continue
		}

		// Try to find the Group
		var err error
		resp, err := conn.DescribeDBClusters(
			&docdb.DescribeDBClustersInput{
				DBClusterIdentifier: aws.String(rs.Primary.ID),
			})

		if err == nil {
			if len(resp.DBClusters) != 0 &&
				*resp.DBClusters[0].DBClusterIdentifier == rs.Primary.ID {
				return fmt.Errorf("DB Cluster %s still exists", rs.Primary.ID)
			}
		}

		// Return nil if the cluster is already destroyed
		if awsErr, ok := err.(awserr.Error); ok {
			if awsErr.Code() == "DBClusterNotFoundFault" {
				return nil
			}
		}

		return err
	}

	return nil
}

func testAccCheckDocDBClusterSnapshot(rInt int) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		for _, rs := range s.RootModule().Resources {
			if rs.Type != "aws_docdb_cluster" {
				continue
			}

			// Try and delete the snapshot before we check for the cluster not found
			snapshot_identifier := fmt.Sprintf("tf-acctest-docdbcluster-snapshot-%d", rInt)

			awsClient := testAccProvider.Meta().(*AWSClient)
			conn := awsClient.docdbconn

			log.Printf("[INFO] Deleting the Snapshot %s", snapshot_identifier)
			_, snapDeleteErr := conn.DeleteDBClusterSnapshot(
				&docdb.DeleteDBClusterSnapshotInput{
					DBClusterSnapshotIdentifier: aws.String(snapshot_identifier),
				})
			if snapDeleteErr != nil {
				return snapDeleteErr
			}

			// Try to find the Group
			var err error
			resp, err := conn.DescribeDBClusters(
				&docdb.DescribeDBClustersInput{
					DBClusterIdentifier: aws.String(rs.Primary.ID),
				})

			if err == nil {
				if len(resp.DBClusters) != 0 &&
					*resp.DBClusters[0].DBClusterIdentifier == rs.Primary.ID {
					return fmt.Errorf("DB Cluster %s still exists", rs.Primary.ID)
				}
			}

			// Return nil if the cluster is already destroyed
			if awsErr, ok := err.(awserr.Error); ok {
				if awsErr.Code() == "DBClusterNotFoundFault" {
					return nil
				}
			}

			return err
		}

		return nil
	}
}

func testAccCheckDocDBClusterExists(n string, v *docdb.DBCluster) resource.TestCheckFunc {
	return testAccCheckDocDBClusterExistsWithProvider(n, v, func() *schema.Provider { return testAccProvider })
}

func testAccCheckDocDBClusterExistsWithProvider(n string, v *docdb.DBCluster, providerF func() *schema.Provider) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No DB Instance ID is set")
		}

		provider := providerF()
		conn := provider.Meta().(*AWSClient).docdbconn
		resp, err := conn.DescribeDBClusters(&docdb.DescribeDBClustersInput{
			DBClusterIdentifier: aws.String(rs.Primary.ID),
		})

		if err != nil {
			return err
		}

		for _, c := range resp.DBClusters {
			if *c.DBClusterIdentifier == rs.Primary.ID {
				*v = *c
				return nil
			}
		}

		return fmt.Errorf("DB Cluster (%s) not found", rs.Primary.ID)
	}
}

func testAccCheckDocDBClusterRecreated(i, j *docdb.DBCluster) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if aws.TimeValue(i.ClusterCreateTime) == aws.TimeValue(j.ClusterCreateTime) {
			return errors.New("DocDB Cluster was not recreated")
		}

		return nil
	}
}

func testAccDocDBClusterConfig(n int) string {
	return fmt.Sprintf(`
resource "aws_docdb_cluster" "default" {
  cluster_identifier = "tf-docdb-cluster-%d"
  availability_zones = ["us-west-2a","us-west-2b","us-west-2c"]
  master_username = "foo"
  master_password = "mustbeeightcharaters"
  db_cluster_parameter_group_name = "default.docdb3.6"
  skip_final_snapshot = true
  tags = {
    Environment = "production"
  }
  enabled_cloudwatch_logs_exports = [
	"audit",
  ]
}`, n)
}

func testAccDocDBClusterConfig_namePrefix(n int) string {
	return fmt.Sprintf(`
resource "aws_docdb_cluster" "test" {
  cluster_identifier_prefix = "tf-test-"
  master_username = "root"
  master_password = "password"
  skip_final_snapshot = true
}
`, n)
}

func testAccDocDBClusterConfigWithFinalSnapshot(n int) string {
	return fmt.Sprintf(`
resource "aws_docdb_cluster" "default" {
  cluster_identifier = "tf-docdb-cluster-%d"
  availability_zones = ["us-west-2a","us-west-2b","us-west-2c"]
  master_username = "foo"
  master_password = "mustbeeightcharaters"
  db_cluster_parameter_group_name = "default.docdb3.6"
  final_snapshot_identifier = "tf-acctest-docdbcluster-snapshot-%d"
  tags = {
    Environment = "production"
  }
}`, n, n)
}

func testAccDocDBClusterConfigWithoutUserNameAndPassword(n int) string {
	return fmt.Sprintf(`
resource "aws_docdb_cluster" "default" {
  cluster_identifier = "tf-docdb-cluster-%d"
  availability_zones = ["us-west-2a","us-west-2b","us-west-2c"]
  skip_final_snapshot = true
}`, n)
}

func testAccDocDBClusterConfigUpdatedTags(n int) string {
	return fmt.Sprintf(`
resource "aws_docdb_cluster" "default" {
  cluster_identifier = "tf-docdb-cluster-%d"
  availability_zones = ["us-west-2a","us-west-2b","us-west-2c"]
  master_username = "foo"
  master_password = "mustbeeightcharaters"
  db_cluster_parameter_group_name = "default.docdb3.6"
  skip_final_snapshot = true
  tags = {
    Environment = "production"
    AnotherTag = "test"
  }
}`, n)
}

func testAccDocDBClusterConfigUpdatedCloudwatchLogsExports(n int) string {
	return fmt.Sprintf(`
resource "aws_docdb_cluster" "default" {
  cluster_identifier = "tf-docdb-cluster-%d"
  availability_zones = ["us-west-2a","us-west-2b","us-west-2c"]
  master_username = "foo"
  master_password = "mustbeeightcharaters"
  db_cluster_parameter_group_name = "default.docdb3.6"
  skip_final_snapshot = true
  enabled_cloudwatch_logs_exports = [
    "error",
    "slowquery"
  ]
}`, n)
}

func testAccDocDBClusterConfig_kmsKey(n int) string {
	return fmt.Sprintf(`

 resource "aws_kms_key" "foo" {
     description = "Terraform acc test %d"
     policy = <<POLICY
 {
   "Version": "2012-10-17",
   "Id": "kms-tf-1",
   "Statement": [
     {
       "Sid": "Enable IAM User Permissions",
       "Effect": "Allow",
       "Principal": {
         "AWS": "*"
       },
       "Action": "kms:*",
       "Resource": "*"
     }
   ]
 }
 POLICY
 }

 resource "aws_docdb_cluster" "default" {
   cluster_identifier = "tf-docdb-cluster-%d"
   availability_zones = ["us-west-2a","us-west-2b","us-west-2c"]
   master_username = "foo"
   master_password = "mustbeeightcharaters"
   db_cluster_parameter_group_name = "default.docdb3.6"
   storage_encrypted = true
   kms_key_id = "${aws_kms_key.foo.arn}"
   skip_final_snapshot = true
 }`, n, n)
}

func testAccDocDBClusterConfig_encrypted(n int) string {
	return fmt.Sprintf(`
resource "aws_docdb_cluster" "default" {
  cluster_identifier = "tf-docdb-cluster-%d"
  availability_zones = ["us-west-2a","us-west-2b","us-west-2c"]
  master_username = "foo"
  master_password = "mustbeeightcharaters"
  storage_encrypted = true
  skip_final_snapshot = true
}`, n)
}

func testAccDocDBClusterConfig_backups(n int) string {
	return fmt.Sprintf(`
resource "aws_docdb_cluster" "default" {
  cluster_identifier = "tf-docdb-cluster-%d"
  availability_zones = ["us-west-2a","us-west-2b","us-west-2c"]
  master_username = "foo"
  master_password = "mustbeeightcharaters"
  backup_retention_period = 5
  preferred_backup_window = "07:00-09:00"
  preferred_maintenance_window = "tue:04:00-tue:04:30"
  skip_final_snapshot = true
}`, n)
}

func testAccDocDBClusterConfig_backupsUpdate(n int) string {
	return fmt.Sprintf(`
resource "aws_docdb_cluster" "default" {
  cluster_identifier = "tf-docdb-cluster-%d"
  availability_zones = ["us-west-2a","us-west-2b","us-west-2c"]
  master_username = "foo"
  master_password = "mustbeeightcharaters"
  backup_retention_period = 10
  preferred_backup_window = "03:00-09:00"
  preferred_maintenance_window = "wed:01:00-wed:01:30"
  apply_immediately = true
  skip_final_snapshot = true
}`, n)
}

func testAccDocDBClusterConfig_Port(rInt, port int) string {
	return fmt.Sprintf(`
data "aws_availability_zones" "available" {}

resource "aws_docdb_cluster" "test" {
  availability_zones              = ["${data.aws_availability_zones.available.names}"]
  cluster_identifier              = "tf-acc-test-%d"
  db_cluster_parameter_group_name = "default.docdb3.6"
  engine                          = "docdb"
  master_password                 = "mustbeeightcharaters"
  master_username                 = "foo"
  port                            = %d
  skip_final_snapshot             = true
}`, rInt, port)
}

func testAccAWSDocDBClusterConfig_SnapshotIdentifier(rName string) string {
	return fmt.Sprintf(`
resource "aws_docdb_cluster" "source" {
  cluster_identifier   = "%s-source"
  master_password      = "barbarbarbar"
  master_username      = "foo"
  skip_final_snapshot  = true
}

resource "aws_db_cluster_snapshot" "test" {
  db_cluster_identifier          = "${aws_docdb_cluster.source.id}"
  db_cluster_snapshot_identifier = %q
}

resource "aws_docdb_cluster" "test" {
  cluster_identifier  = %q
  skip_final_snapshot = true
  snapshot_identifier = "${aws_db_cluster_snapshot.test.id}"
}
`, rName, rName, rName)
}

func testAccAWSDocDBClusterConfig_SnapshotIdentifier_EngineVersion(rName, engine, engineVersionSource, engineVersion string) string {
	return fmt.Sprintf(`
resource "aws_docdb_cluster" "source" {
  cluster_identifier  = "%s-source"
  engine              = %q
  engine_version      = %q
  master_password     = "barbarbarbar"
  master_username     = "foo"
  skip_final_snapshot = true
}

resource "aws_db_cluster_snapshot" "test" {
  db_cluster_identifier          = "${aws_docdb_cluster.source.id}"
  db_cluster_snapshot_identifier = %q
}

resource "aws_docdb_cluster" "test" {
  cluster_identifier  = %q
  engine              = %q
  engine_version      = %q
  skip_final_snapshot = true
  snapshot_identifier = "${aws_db_cluster_snapshot.test.id}"
}
`, rName, engine, engineVersionSource, rName, rName, engine, engineVersion)
}

func testAccAWSDocDBClusterConfig_SnapshotIdentifier_PreferredBackupWindow(rName, preferredBackupWindow string) string {
	return fmt.Sprintf(`
resource "aws_docdb_cluster" "source" {
  cluster_identifier  = "%s-source"
  master_password     = "barbarbarbar"
  master_username     = "foo"
  skip_final_snapshot = true
}

resource "aws_db_cluster_snapshot" "test" {
  db_cluster_identifier          = "${aws_docdb_cluster.source.id}"
  db_cluster_snapshot_identifier = %q
}

resource "aws_docdb_cluster" "test" {
  cluster_identifier      = %q
  preferred_backup_window = %q
  skip_final_snapshot     = true
  snapshot_identifier     = "${aws_db_cluster_snapshot.test.id}"
}
`, rName, rName, rName, preferredBackupWindow)
}

func testAccAWSDocDBClusterConfig_SnapshotIdentifier_PreferredMaintenanceWindow(rName, preferredMaintenanceWindow string) string {
	return fmt.Sprintf(`
resource "aws_docdb_cluster" "source" {
  cluster_identifier  = "%s-source"
  master_password     = "barbarbarbar"
  master_username     = "foo"
  skip_final_snapshot = true
}

resource "aws_db_cluster_snapshot" "test" {
  db_cluster_identifier          = "${aws_docdb_cluster.source.id}"
  db_cluster_snapshot_identifier = %q
}

resource "aws_docdb_cluster" "test" {
  cluster_identifier           = %q
  preferred_maintenance_window = %q
  skip_final_snapshot          = true
  snapshot_identifier          = "${aws_db_cluster_snapshot.test.id}"
}
`, rName, rName, rName, preferredMaintenanceWindow)
}

func testAccAWSDocDBClusterConfig_SnapshotIdentifier_Tags(rName string) string {
	return fmt.Sprintf(`
resource "aws_docdb_cluster" "source" {
  cluster_identifier   = "%s-source"
  master_password      = "barbarbarbar"
  master_username      = "foo"
  skip_final_snapshot  = true
}

resource "aws_db_cluster_snapshot" "test" {
  db_cluster_identifier          = "${aws_docdb_cluster.source.id}"
  db_cluster_snapshot_identifier = %q
}

resource "aws_docdb_cluster" "test" {
  cluster_identifier  = %q
  skip_final_snapshot = true
  snapshot_identifier = "${aws_db_cluster_snapshot.test.id}"

  tags = {
    key1 = "value1"
  }
}
`, rName, rName, rName)
}

func testAccAWSDocDBClusterConfig_SnapshotIdentifier_VpcSecurityGroupIds(rName string) string {
	return fmt.Sprintf(`
data "aws_vpc" "default" {
  default = true
}

data "aws_security_group" "default" {
  name   = "default"
  vpc_id = "${data.aws_vpc.default.id}"
}

resource "aws_docdb_cluster" "source" {
  cluster_identifier   = "%s-source"
  master_password      = "barbarbarbar"
  master_username      = "foo"
  skip_final_snapshot  = true
}

resource "aws_db_cluster_snapshot" "test" {
  db_cluster_identifier          = "${aws_docdb_cluster.source.id}"
  db_cluster_snapshot_identifier = %q
}

resource "aws_docdb_cluster" "test" {
  cluster_identifier     = %q
  skip_final_snapshot    = true
  snapshot_identifier    = "${aws_db_cluster_snapshot.test.id}"
  vpc_security_group_ids = ["${data.aws_security_group.default.id}"]
}
`, rName, rName, rName)
}

func testAccAWSDocDBClusterConfig_SnapshotIdentifier_VpcSecurityGroupIds_Tags(rName string) string {
	return fmt.Sprintf(`
data "aws_vpc" "default" {
  default = true
}

data "aws_security_group" "default" {
  name   = "default"
  vpc_id = "${data.aws_vpc.default.id}"
}

resource "aws_docdb_cluster" "source" {
  cluster_identifier   = "%s-source"
  master_password      = "barbarbarbar"
  master_username      = "foo"
  skip_final_snapshot  = true
}

resource "aws_db_cluster_snapshot" "test" {
  db_cluster_identifier          = "${aws_docdb_cluster.source.id}"
  db_cluster_snapshot_identifier = %q
}

resource "aws_docdb_cluster" "test" {
  cluster_identifier     = %q
  skip_final_snapshot    = true
  snapshot_identifier    = "${aws_db_cluster_snapshot.test.id}"
  vpc_security_group_ids = ["${data.aws_security_group.default.id}"]

  tags = {
    key1 = "value1"
  }
}
`, rName, rName, rName)
}

func testAccAWSDocDBClusterConfig_SnapshotIdentifier_EncryptedRestore(rName string) string {
	return fmt.Sprintf(`
resource "aws_kms_key" "test" {}

resource "aws_docdb_cluster" "source" {
  cluster_identifier   = "%s-source"
  master_password      = "barbarbarbar"
  master_username      = "foo"
  skip_final_snapshot  = true
}

resource "aws_db_cluster_snapshot" "test" {
  db_cluster_identifier          = "${aws_docdb_cluster.source.id}"
  db_cluster_snapshot_identifier = %q
}

resource "aws_docdb_cluster" "test" {
  cluster_identifier  = %q
  skip_final_snapshot = true
  snapshot_identifier = "${aws_db_cluster_snapshot.test.id}"

  storage_encrypted = true
  kms_key_id = "${aws_kms_key.test.arn}"
}
`, rName, rName, rName)
}
