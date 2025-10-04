import boto3
import json
import os

region = "us-east-2"
ec2 = boto3.client("ec2", region_name=region)
r53 = boto3.client("route53")

def lambda_handler(event, context):
    instance_id = os.environ["INSTANCE_ID"]
    dns_name = os.environ.get("DNS_NAME", "")
    cors_origin = os.environ.get("CORS_ORIGIN", "*")

    path = event.get("path", "/")

    if path in ("/status", "/api/status"):
        response_message = describe_state(instance_id)
    elif path in ("/stop", "/api/stop"):
        response_message = stop_instance(instance_id)
    elif path in ("/start", "/api/start"):
        response_message = start_instance(instance_id)
    elif path in ("/syncdns", "/api/syncdns"):
        if dns_name:
            response_message = update_dns(instance_id, dns_name)
        else:
            response_message = {"message": "DNS not configured"}
    else:
        return {"statusCode": 404}

    return {
        "statusCode": 200,
        "headers": {"Access-Control-Allow-Origin": cors_origin},
        "body": json.dumps(response_message),
    }


def stop_instance(instance_id):
    ec2.stop_instances(InstanceIds=[instance_id])
    return {"message": "Success"}


def start_instance(instance_id):
    ec2.start_instances(InstanceIds=[instance_id])
    return {"message": "Success"}


def get_instance(instance_id):
    return ec2.describe_instances(InstanceIds=[instance_id])["Reservations"][0]["Instances"][0]


def get_dns_record(hosted_zone_id, record_type="A"):
    sets = r53.list_resource_record_sets(HostedZoneId=hosted_zone_id)["ResourceRecordSets"]
    return next((item for item in sets if item.get("Type") == record_type), {})


def get_hosted_zone_id():
    zones = r53.list_hosted_zones().get("HostedZones", [])
    return zones[0]["Id"] if zones else None


def describe_state(instance_id):
    instance = get_instance(instance_id)

    state = instance["State"]["Name"]
    ip = instance.get("PublicIpAddress")

    dns_record = {"name": None, "value": None, "type": None}
    try:
        hosted_zone_id = os.environ.get("ZONE_ID", get_hosted_zone_id())

        if hosted_zone_id:
            record = get_dns_record(hosted_zone_id)
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
