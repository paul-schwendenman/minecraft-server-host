"""Tests for aws_utils module with moto mocks."""

import os
import pytest
import boto3
from moto import mock_aws


@pytest.fixture(autouse=True)
def setup_env():
    """Set up environment variables for each test."""
    os.environ["INSTANCE_ID"] = "i-test123"
    os.environ["REGION"] = "us-east-2"
    os.environ["DNS_NAME"] = "minecraft.example.com"
    os.environ["ZONE_ID"] = ""
    os.environ["AWS_LOCAL"] = ""
    os.environ["AWS_ACCESS_KEY_ID"] = "testing"
    os.environ["AWS_SECRET_ACCESS_KEY"] = "testing"
    os.environ["AWS_DEFAULT_REGION"] = "us-east-2"

    # Clear config cache
    from app.config import get_settings
    get_settings.cache_clear()


@mock_aws
def test_get_instance():
    """Test get_instance returns instance data."""
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

    # Clear cache and reimport
    from app.config import get_settings
    get_settings.cache_clear()

    # Need to reimport aws_utils after setting up mocks
    import importlib
    import app.aws_utils as aws_utils
    importlib.reload(aws_utils)

    instance = aws_utils.get_instance(instance_id)

    assert instance["InstanceId"] == instance_id
    assert instance["State"]["Name"] in ["pending", "running"]


@mock_aws
def test_start_instance():
    """Test start_instance returns success."""
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

    # Stop the instance first
    ec2.stop_instances(InstanceIds=[instance_id])
    os.environ["INSTANCE_ID"] = instance_id

    from app.config import get_settings
    get_settings.cache_clear()

    import importlib
    import app.aws_utils as aws_utils
    importlib.reload(aws_utils)

    result = aws_utils.start_instance(instance_id)

    assert result == {"message": "Success"}


@mock_aws
def test_stop_instance():
    """Test stop_instance returns success."""
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

    from app.config import get_settings
    get_settings.cache_clear()

    import importlib
    import app.aws_utils as aws_utils
    importlib.reload(aws_utils)

    result = aws_utils.stop_instance(instance_id)

    assert result == {"message": "Success"}


@mock_aws
def test_get_hosted_zone_id_from_env():
    """Test get_hosted_zone_id returns zone from env."""
    os.environ["ZONE_ID"] = "Z123456"

    from app.config import get_settings
    get_settings.cache_clear()

    import importlib
    import app.aws_utils as aws_utils
    importlib.reload(aws_utils)

    zone_id = aws_utils.get_hosted_zone_id()

    assert zone_id == "Z123456"


@mock_aws
def test_get_hosted_zone_id_from_api():
    """Test get_hosted_zone_id finds first zone when not in env."""
    os.environ["ZONE_ID"] = ""

    r53 = boto3.client("route53")
    zone = r53.create_hosted_zone(
        Name="example.com",
        CallerReference="test-ref-123",
    )
    expected_zone_id = zone["HostedZone"]["Id"]

    from app.config import get_settings
    get_settings.cache_clear()

    import importlib
    import app.aws_utils as aws_utils
    importlib.reload(aws_utils)

    zone_id = aws_utils.get_hosted_zone_id()

    assert zone_id == expected_zone_id


@mock_aws
def test_describe_state():
    """Test describe_state returns combined EC2 and DNS state."""
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
    os.environ["ZONE_ID"] = ""
    os.environ["DNS_NAME"] = ""

    from app.config import get_settings
    get_settings.cache_clear()

    import importlib
    import app.aws_utils as aws_utils
    importlib.reload(aws_utils)

    result = aws_utils.describe_state(instance_id, "")

    assert "instance" in result
    assert "dns_record" in result
    assert result["instance"]["state"] in ["pending", "running"]


@mock_aws
def test_get_dns_record():
    """Test get_dns_record fetches existing record."""
    r53 = boto3.client("route53")

    zone = r53.create_hosted_zone(
        Name="example.com",
        CallerReference="test-ref-123",
    )
    zone_id = zone["HostedZone"]["Id"]

    # Create a record
    r53.change_resource_record_sets(
        HostedZoneId=zone_id,
        ChangeBatch={
            "Changes": [
                {
                    "Action": "CREATE",
                    "ResourceRecordSet": {
                        "Name": "minecraft.example.com",
                        "Type": "A",
                        "TTL": 300,
                        "ResourceRecords": [{"Value": "1.2.3.4"}],
                    },
                }
            ]
        },
    )

    os.environ["ZONE_ID"] = zone_id

    from app.config import get_settings
    get_settings.cache_clear()

    import importlib
    import app.aws_utils as aws_utils
    importlib.reload(aws_utils)

    record = aws_utils.get_dns_record(zone_id, "minecraft.example.com")

    assert record["Name"] == "minecraft.example.com."
    assert record["Type"] == "A"


@mock_aws
def test_get_dns_record_not_found():
    """Test get_dns_record returns empty dict when not found."""
    r53 = boto3.client("route53")

    zone = r53.create_hosted_zone(
        Name="example.com",
        CallerReference="test-ref-123",
    )
    zone_id = zone["HostedZone"]["Id"]

    os.environ["ZONE_ID"] = zone_id

    from app.config import get_settings
    get_settings.cache_clear()

    import importlib
    import app.aws_utils as aws_utils
    importlib.reload(aws_utils)

    record = aws_utils.get_dns_record(zone_id, "nonexistent.example.com")

    assert record == {}
