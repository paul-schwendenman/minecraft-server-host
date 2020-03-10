import boto3
import json
import os

region = 'us-east-2'
ec2 = boto3.client('ec2', region_name=region)

def main_handler(event, context):
    instance_id = os.environ['INSTANCE_ID']
    if event["path"] == '/':
        instance_dict = ec2.describe_instances(InstanceIds=[instance_id])

        instance = instance_dict.get("Reservations")[0].get("Instances")[0]

        ip_address = instance.get("PublicIpAddress")
        state = instance.get('State').get('Name')

        response_message = {
            "ip_address": ip_address,
            "state": state
        }
    elif event["path"] == '/stop':
        ec2.stop_instances(InstanceIds=[instance_id])

        response_message = 'stopped'
    elif event["path"] == '/start':
        ec2.start_instances(InstanceIds=[instance_id])

        response_message = 'started'
    else:
        response_message = None
    return {
        "statusCode": 200,
        "body": json.dumps(response_message)
    }
