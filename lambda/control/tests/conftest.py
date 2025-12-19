"""Pytest fixtures for control lambda tests."""

import os
import pytest
import boto3
from moto import mock_aws


@pytest.fixture(scope="function")
def aws_credentials():
    """Mock AWS credentials for moto."""
    os.environ["AWS_ACCESS_KEY_ID"] = "testing"
    os.environ["AWS_SECRET_ACCESS_KEY"] = "testing"
    os.environ["AWS_SECURITY_TOKEN"] = "testing"
    os.environ["AWS_SESSION_TOKEN"] = "testing"
    os.environ["AWS_DEFAULT_REGION"] = "us-east-2"


@pytest.fixture(scope="function")
def env_vars():
    """Set required environment variables."""
    os.environ["INSTANCE_ID"] = "i-1234567890abcdef0"
    os.environ["REGION"] = "us-east-2"
    os.environ["DNS_NAME"] = "minecraft.example.com"
    os.environ["ZONE_ID"] = ""
    os.environ["CORS_ORIGIN"] = "*"
    os.environ["AWS_LOCAL"] = ""
    yield
    # Cleanup is handled by test isolation


@pytest.fixture(scope="function")
def mock_ec2(aws_credentials):
    """Create a mock EC2 client with a test instance."""
    with mock_aws():
        ec2 = boto3.client("ec2", region_name="us-east-2")

        # Create VPC and subnet (required for EC2)
        vpc = ec2.create_vpc(CidrBlock="10.0.0.0/16")
        vpc_id = vpc["Vpc"]["VpcId"]

        subnet = ec2.create_subnet(VpcId=vpc_id, CidrBlock="10.0.1.0/24")
        subnet_id = subnet["Subnet"]["SubnetId"]

        # Create a test instance
        instances = ec2.run_instances(
            ImageId="ami-12345678",
            MinCount=1,
            MaxCount=1,
            InstanceType="t2.micro",
            SubnetId=subnet_id,
        )
        instance_id = instances["Instances"][0]["InstanceId"]

        # Update env var with actual instance ID
        os.environ["INSTANCE_ID"] = instance_id

        yield ec2, instance_id


@pytest.fixture(scope="function")
def mock_route53(aws_credentials):
    """Create a mock Route53 client with a test hosted zone."""
    with mock_aws():
        r53 = boto3.client("route53")

        # Create a hosted zone
        zone = r53.create_hosted_zone(
            Name="example.com",
            CallerReference="test-ref-123",
        )
        zone_id = zone["HostedZone"]["Id"]

        os.environ["ZONE_ID"] = zone_id

        yield r53, zone_id


@pytest.fixture(scope="function")
def mock_aws_full(aws_credentials, env_vars):
    """Combined mock for EC2 and Route53."""
    with mock_aws():
        # Set up EC2
        ec2 = boto3.client("ec2", region_name="us-east-2")

        vpc = ec2.create_vpc(CidrBlock="10.0.0.0/16")
        vpc_id = vpc["Vpc"]["VpcId"]

        subnet = ec2.create_subnet(VpcId=vpc_id, CidrBlock="10.0.1.0/24")
        subnet_id = subnet["Subnet"]["SubnetId"]

        instances = ec2.run_instances(
            ImageId="ami-12345678",
            MinCount=1,
            MaxCount=1,
            InstanceType="t2.micro",
            SubnetId=subnet_id,
        )
        instance_id = instances["Instances"][0]["InstanceId"]
        os.environ["INSTANCE_ID"] = instance_id

        # Set up Route53
        r53 = boto3.client("route53")
        zone = r53.create_hosted_zone(
            Name="example.com",
            CallerReference="test-ref-123",
        )
        zone_id = zone["HostedZone"]["Id"]
        os.environ["ZONE_ID"] = zone_id

        yield {
            "ec2": ec2,
            "r53": r53,
            "instance_id": instance_id,
            "zone_id": zone_id,
        }
