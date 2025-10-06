import boto3
import logging
import os

from fastapi import FastAPI
from mangum import Mangum

logger = logging.getLogger()
logger.setLevel(logging.INFO)

instance_id = os.environ['INSTANCE_ID']
region = os.environ.get('REGION', 'us-east-2')
dns_name = os.environ.get("DNS_NAME", "")
cors_origin = os.environ.get("CORS_ORIGIN", "*")

ec2 = boto3.client('ec2', region_name=region)
r53 = boto3.client('route53')


app = FastAPI()
handler = Mangum(app)

@app.get("/")
def root():
    return "Minecraft Server API"

@app.get("/status")
def status():
    return describe_state(instance_id, dns_name)

@app.post("/stop")
def stop():
    return stop_instance(instance_id)

@app.post("/start")
def start():
    return start_instance(instance_id)

@app.post("/syncdns")
def sync_dns():
    if dns_name:
        return update_dns(instance_id, dns_name)
    return {"message": "DNS not configured"}


def stop_instance(instance_id):
    ec2.stop_instances(InstanceIds=[instance_id])
    return {"message": "Success"}


def start_instance(instance_id):
    ec2.start_instances(InstanceIds=[instance_id])
    return {"message": "Success"}


def get_instance(instance_id):
    return ec2.describe_instances(InstanceIds=[instance_id])["Reservations"][0]["Instances"][0]


def get_dns_record(hosted_zone_id, record_name, record_type="A"):
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


def get_hosted_zone_id():
    zones = r53.list_hosted_zones().get("HostedZones", [])
    return zones[0]["Id"] if zones else None


def describe_state(instance_id, dns_name):
    instance = get_instance(instance_id)

    state = instance["State"]["Name"]
    ip = instance.get("PublicIpAddress")

    dns_record = {"name": None, "value": None, "type": None}
    try:
        hosted_zone_id = os.environ.get("ZONE_ID", get_hosted_zone_id())

        if hosted_zone_id:
            record = get_dns_record(hosted_zone_id, dns_name)
            dns_record = {
                "name": record.get("Name"),
                "value": record.get("ResourceRecords", [{}])[0].get("Value"),
                "type": record.get("Type"),
            }
    except Exception as e:
        print(f"DNS lookup skipped: {e}")

    return {"instance": {"state": state, "ip_address": ip}, "dns_record": dns_record}


def update_dns(instance_id, dns_name, record_type="A"):
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
                    }
                }
            ],
        },
    )
    return {"message": "Success"}
