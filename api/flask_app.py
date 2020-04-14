from flask import Flask, request, abort
from mcstatus import MinecraftServer
import socket
import werkzeug
import boto3
import json
import os

region = 'us-east-2'
ec2 = boto3.client('ec2', region_name=region)
r53 = boto3.client('route53')

instance_id = os.environ['INSTANCE_ID']
dns_name = os.environ['DNS_NAME']
cors_origin = os.environ['CORS_ORIGIN']

app = Flask(__name__)

@app.route("/")
def hello():
    return "Minecraft Server API"

@app.route("/status")
def status():
    hosted_zone_id = get_hosted_zone_id()
    dns_record = get_dns_record(hosted_zone_id)
    instance = get_instance(instance_id)

    instance_state = instance.get('State').get('Name')
    instance_ip = instance.get("PublicIpAddress")

    dns_record_name = dns_record.get("Name")
    dns_record_value = dns_record.get("ResourceRecords", [{}])[0].get("Value")
    dns_record_type = dns_record.get("Type")

    return {
        "instance": {
            "state": instance_state,
            "ip_address": instance_ip,
        },
        "dns_record": {
            "name": dns_record_name,
            "value": dns_record_value,
            "type": dns_record_type,
        },
    }

@app.route("/stop")
def stop():
    ec2.stop_instances(InstanceIds=[instance_id])

    return {"message": "Success"}

@app.route("/start")
def start():
    ec2.start_instances(InstanceIds=[instance_id])

    return {"message": "Success"}

@app.route("/syncdns")
def sync_dns():
    hosted_zone_id = get_hosted_zone_id()
    instance = get_instance(instance_id)
    ip_address = instance.get("PublicIpAddress")

    if not ip_address:
        return {"message": "DNS Sync Failed"}

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

    return {"message": "Success"}

@app.route("/details")
def details():
    hostname = request.args.get('hostname', default="minecraft.paulandsierra.com", type=str)
    server = MinecraftServer(hostname)
    try:
        status = server.status()
    except socket.timeout:
        abort(503)

    return status.raw

@app.errorhandler(werkzeug.exceptions.ServiceUnavailable)
def handle_bad_request(e):
    return 'Service Unavailable', 503


def get_instance(instance_id):
    instance_dict = ec2.describe_instances(InstanceIds=[instance_id])

    return instance_dict.get("Reservations")[0].get("Instances")[0]


def get_dns_record(hosted_zone_id, record_type = "A"):
    return list(filter(lambda item: item.get("Type") == record_type,
        r53.list_resource_record_sets(HostedZoneId=hosted_zone_id).get("ResourceRecordSets")))[0]

def get_hosted_zone_id():
    return r53.list_hosted_zones().get('HostedZones')[0].get('Id')
