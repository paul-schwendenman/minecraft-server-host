import pytest
import boto3
import os
import socket
import json
from http import HTTPStatus
from unittest.mock import patch


def test_hello_route(test_client):
    response = test_client.get('/')
    assert response.status_code == HTTPStatus.OK
    assert response.get_data() == b"Minecraft Server API"


def test_get_status(test_client, dns_entry, ec2_instance):
    response = test_client.get('/status')

    assert response.status_code == HTTPStatus.OK
    assert parse_response_to_json(response) == {
        'dns_record': {
            'name': "example.",
            'type': 'A',
            'value': '10.0.0.1'
        },
        'instance': {
            "ip_address": ec2_instance.get("PublicIpAddress"),
            "state": "running"
        }
    }


def test_stop_instance(test_client, ec2_client, ec2_instance):
    instances = ec2_client.describe_instances()['Reservations'][0]['Instances']
    assert len(instances) == 1
    assert instances[0]["State"]["Name"] == "running"

    response = test_client.get('/stop')

    assert response.status_code == HTTPStatus.OK
    instances = ec2_client.describe_instances()['Reservations'][0]['Instances']
    assert len(instances) == 1
    assert instances[0]["State"]["Name"] == "stopped"



def test_start_instance(test_client, ec2_client, ec2_instance):
    ec2 = boto3.resource('ec2', region_name='us-east-2')
    ec2.instances.filter(InstanceIds=[ec2_instance["InstanceId"]]).stop()
    instances = ec2_client.describe_instances()['Reservations'][0]['Instances']
    assert len(instances) == 1
    assert instances[0]["State"]["Name"] == "stopped"

    response = test_client.get('/start')

    assert response.status_code == HTTPStatus.OK
    instances = ec2_client.describe_instances()['Reservations'][0]['Instances']
    assert len(instances) == 1
    assert instances[0]["State"]["Name"] == "running"


def test_update_dns_creates_missing_dns_record(test_client, route_53, hosted_zone, ec2_instance):
    os.environ['DNS_NAME'] = "example"
    assert route_53.list_resource_record_sets(HostedZoneId=hosted_zone["Id"])["ResourceRecordSets"] == []

    response = test_client.get('/syncdns')

    assert response.status_code == HTTPStatus.OK
    assert route_53.list_resource_record_sets(HostedZoneId=hosted_zone["Id"])["ResourceRecordSets"] == [
        {
            'Name': 'example.',
            'ResourceRecords': [{'Value': ec2_instance["PublicIpAddress"]}],
            'TTL': 300,
            'Type': 'A'
        }
    ]


def test_update_dns_creates_updates_existing_record(test_client, route_53, hosted_zone, dns_entry, ec2_instance):
    os.environ['DNS_NAME'] = "example"
    assert route_53.list_resource_record_sets(HostedZoneId=hosted_zone["Id"])["ResourceRecordSets"] == [
        {
            'Name': 'example.',
            'ResourceRecords': [{'Value': '10.0.0.1'}],
            'TTL': 300,
            'Type': 'A'
        }
    ]

    response = test_client.get('/syncdns')

    assert response.status_code == HTTPStatus.OK
    assert route_53.list_resource_record_sets(HostedZoneId=hosted_zone["Id"])["ResourceRecordSets"] == [
        {
            'Name': 'example.',
            'ResourceRecords': [{'Value': ec2_instance["PublicIpAddress"]}],
            'TTL': 300,
            'Type': 'A'
        }
    ]


def test_server_details_bad_request(test_client):
    response = test_client.get('/details')

    assert response.status_code == HTTPStatus.BAD_REQUEST


@patch('mcstatus.MinecraftServer')
def test_server_details(mock_mcstatus, test_client):
    mock_mcstatus.return_value.status.return_value.raw = {
        "description": {
            "text": "A Minecraft Server"
        },
        "players": {
            "max": 20,
            "online": 0
        },
        "version": {
            "name": "1.15.2",
            "protocol": 578
        }
    }

    response = test_client.get('/details?hostname=10.0.0.1')

    assert response.status_code == HTTPStatus.OK
    mock_mcstatus.assert_called_once_with("10.0.0.1")
    mock_mcstatus.return_value.status.assert_called_once_with()


@patch('mcstatus.MinecraftServer')
def test_server_details_returns_service_unavailable_on_timeout(mock_mcstatus, test_client):
    mock_mcstatus.return_value.status.side_effect = socket.timeout
    response = test_client.get('/details?hostname=10.0.0.2')

    assert response.status_code == HTTPStatus.SERVICE_UNAVAILABLE
    mock_mcstatus.assert_called_once_with("10.0.0.2")
    mock_mcstatus.return_value.status.assert_called_once_with()


def parse_response_to_json(response):
    return json.loads(response.get_data().decode())
