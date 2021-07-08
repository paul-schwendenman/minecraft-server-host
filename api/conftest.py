import pytest
import os
import boto3
from moto import mock_ec2, mock_route53


@pytest.fixture
def lambda_environment():
    os.environ['CORS_ORIGIN'] = "www.minecraft-server.test"

@pytest.fixture
def test_client(lambda_environment):
    import flask_app
    return flask_app.app.test_client()


@pytest.fixture(scope='function')
def aws_credentials():
    """Mocked AWS Credentials for moto."""
    os.environ['AWS_ACCESS_KEY_ID'] = 'testing'
    os.environ['AWS_SECRET_ACCESS_KEY'] = 'testing'
    os.environ['AWS_SECURITY_TOKEN'] = 'testing'
    os.environ['AWS_SESSION_TOKEN'] = 'testing'


@pytest.fixture(scope='function')
def route_53(aws_credentials):
    with mock_route53():
        yield boto3.client('route53')


@pytest.fixture(scope='function')
def ec2_client(aws_credentials):
    with mock_ec2():
        yield boto3.client('ec2', region_name='us-east-2')


@pytest.fixture(scope='function')
def ec2_instance(ec2_client):
    response = ec2_client.run_instances(ImageId='ami-785db401', MaxCount=1, MinCount=1)
    instance = response["Instances"][0]
    instance_id = instance["InstanceId"]
    os.environ['INSTANCE_ID'] = instance_id
    yield instance


@pytest.fixture(scope='function')
def hosted_zone(route_53):
    hosted_zone = route_53.create_hosted_zone(Name="example", CallerReference="uuid")
    yield hosted_zone["HostedZone"]


@pytest.fixture(scope='function')
def dns_entry(route_53, hosted_zone):
    dns_name = "example"
    record_type = "A"
    ip_address = "10.0.0.1"

    route_53.change_resource_record_sets(
        HostedZoneId=hosted_zone["Id"],
        ChangeBatch={
           'Comment': 'Update IP',
           'Changes': [{
                'Action': 'UPSERT',
                'ResourceRecordSet': {
                    'TTL': 300,
                    'Name': dns_name,
                    'Type': record_type,
                    'ResourceRecords': [{'Value': ip_address}, ],
                }
            }]
        }
    )
