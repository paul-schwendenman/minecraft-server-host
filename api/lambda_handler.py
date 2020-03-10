import boto3
import json
import os

region = 'us-east-2'
ec2 = boto3.client('ec2', region_name=region)
r53 = boto3.client('route53')

def main_handler(event, context):
    instance_id = os.environ['INSTANCE_ID']
    dns_name = os.environ['DNS_NAME']

    if event["path"] == '/':
        response_message = list_details(instance_id)
    elif event["path"] == '/stop':
        response_message = stop_instance(instance_id)
    elif event["path"] == '/start':
        response_message = start_instance(instance_id)
    elif event["path"] == '/dns':
        response_message = update_dns(instance_id, dns_name)
    else:
        response_message = None
    return {
        "statusCode": 200,
        "body": json.dumps(response_message)
    }


def stop_instance(instance_id):
    ec2.stop_instances(InstanceIds=[instance_id])

    return {"state": "stopping"}


def start_instance(instance_id):
    ec2.start_instances(InstanceIds=[instance_id])

    return {"state": "starting"}


def get_instance(instance_id):
    instance_dict = ec2.describe_instances(InstanceIds=[instance_id])

    return  instance_dict.get("Reservations")[0].get("Instances")[0]


def get_dns_record(hosted_zone_id, record_type = "A"):
    return list(filter(lambda item: item.get("Type") == record_type,
        r53.list_resource_record_sets(HostedZoneId=hosted_zone_id).get("ResourceRecordSets")))[0]


def get_hosted_zone_id():
    return r53.list_hosted_zones().get('HostedZones')[0].get('Id')


def list_details(instance_id):
    hosted_zone_id = get_hosted_zone_id()
    dns_record = get_dns_record(hosted_zone_id)

    instance = get_instance(instance_id)
    ip_address = instance.get("PublicIpAddress")
    state = instance.get('State').get('Name')

    return {
        "ip_address": ip_address,
        "state": state,
        "dns": {
            "name": dns_record.get("Name"),
            "value": dns_record.get("ResourceRecords")[0].get("Value"),
            "type": dns_record.get("Type")
        }
    }


def update_dns(instance_id, dns_name, record_type='A'):
    hosted_zone_id = get_hosted_zone_id()
    instance = get_instance(instance_id)
    ip_address = instance.get("PublicIpAddress")

    r53.change_resource_record_sets(
        HostedZoneId=hosted_zone_id,
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

    return { "dns": {
        "value": ip_address,
        "name": dns_name,
        "type": record_type
    }}
