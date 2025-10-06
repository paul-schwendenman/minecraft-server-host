"""Helper functions wrapping EC2 and Route53 operations."""

import boto3
import logging
from typing import Dict, Optional
from .config import get_settings

settings = get_settings()
logger = logging.getLogger(__name__)

# Allow LocalStack endpoint if enabled
endpoint = "http://localhost:4566" if settings["AWS_LOCAL"] else None

ec2 = boto3.client("ec2", region_name=settings["REGION"], endpoint_url=endpoint)
r53 = boto3.client("route53", endpoint_url=endpoint)


def get_instance(instance_id: str) -> Dict:
    """Return the EC2 instance description."""
    resp = ec2.describe_instances(InstanceIds=[instance_id])
    return resp["Reservations"][0]["Instances"][0]


def get_hosted_zone_id() -> Optional[str]:
    """Return zone id from env or the first hosted zone."""
    if settings["ZONE_ID"]:
        return settings["ZONE_ID"]

    zones = r53.list_hosted_zones().get("HostedZones", [])
    return zones[0]["Id"] if zones else None


def get_dns_record(hosted_zone_id: str, record_name: str, record_type: str = "A") -> Dict:
    """Fetch an existing Route53 record, if any."""
    result = r53.list_resource_record_sets(
        HostedZoneId=hosted_zone_id,
        StartRecordName=record_name,
        StartRecordType=record_type,
        MaxItems="1",
    )
    for item in result["ResourceRecordSets"]:
        if (
            item["Type"] == record_type
            and item["Name"].rstrip(".") == record_name.rstrip(".")
        ):
            return item
    return {}


def start_instance(instance_id: str) -> Dict:
    ec2.start_instances(InstanceIds=[instance_id])
    return {"message": "Success"}


def stop_instance(instance_id: str) -> Dict:
    ec2.stop_instances(InstanceIds=[instance_id])
    return {"message": "Success"}


def describe_state(instance_id: str, dns_name: str) -> Dict:
    """Return combined EC2 and DNS state."""
    instance = get_instance(instance_id)
    state = instance["State"]["Name"]
    ip = instance.get("PublicIpAddress")

    dns_record = {"name": None, "value": None, "type": None}
    try:
        hosted_zone_id = get_hosted_zone_id()
        if hosted_zone_id and dns_name:
            record = get_dns_record(hosted_zone_id, dns_name)
            dns_record = {
                "name": record.get("Name"),
                "value": record.get("ResourceRecords", [{}])[0].get("Value"),
                "type": record.get("Type"),
            }
    except Exception as e:
        logger.warning("DNS lookup skipped: %s", e)

    return {"instance": {"state": state, "ip_address": ip}, "dns_record": dns_record}


def update_dns(instance_id: str, dns_name: str, record_type: str = "A") -> Dict:
    """Update Route53 A record to current instance IP."""
    instance = get_instance(instance_id)
    ip_address = instance.get("PublicIpAddress")

    if not ip_address:
        return {"message": "DNS Sync Failed: No public IP"}

    hosted_zone_id = get_hosted_zone_id()
    if not hosted_zone_id:
        return {"message": "DNS Sync Failed: No hosted zone found"}

    r53.change_resource_record_sets(
        HostedZoneId=hosted_zone_id,
        ChangeBatch={
            "Comment": "Update Minecraft server IP",
            "Changes": [
                {
                    "Action": "UPSERT",
                    "ResourceRecordSet": {
                        "TTL": 300,
                        "Name": dns_name,
                        "Type": record_type,
                        "ResourceRecords": [{"Value": ip_address}],
                    },
                }
            ],
        },
    )
    return {"message": "Success"}
