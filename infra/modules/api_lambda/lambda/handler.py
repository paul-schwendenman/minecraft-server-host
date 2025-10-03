import boto3
import os

ec2 = boto3.client("ec2")
INSTANCE_ID = os.environ["INSTANCE_ID"]

def lambda_handler(event, context):
    print("Event:", event)  # debug log

    action = event.get("path", "/").lstrip("/")

    if action == "start":
        ec2.start_instances(InstanceIds=[INSTANCE_ID])
        return {"statusCode": 200, "body": "Starting server"}
    elif action == "stop":
        ec2.stop_instances(InstanceIds=[INSTANCE_ID])
        return {"statusCode": 200, "body": "Stopping server"}
    elif action == "status":
        resp = ec2.describe_instances(InstanceIds=[INSTANCE_ID])
        state = resp["Reservations"][0]["Instances"][0]["State"]["Name"]
        return {"statusCode": 200, "body": f"Server state: {state}"}
    else:
        return {"statusCode": 400, "body": f"Unknown action '{action}'"}
