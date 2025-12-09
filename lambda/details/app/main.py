import json
import socket
import os
from mcstatus import JavaServer

def lambda_handler(event, context):
    cors_origin = os.environ.get("CORS_ORIGIN", "*")

    params = event.get("queryStringParameters") or {}
    hostname = params.get("hostname")

    if not hostname:
        return resp(400, {"message": "Missing ?hostname"}, cors_origin)

    try:
        server = JavaServer.lookup(hostname)
        status = server.status()
        return resp(200, status.raw, cors_origin)
    except socket.timeout:
        return resp(503, {"message": "Server Timeout"}, cors_origin)
    except Exception as e:
        return resp(500, {"message": f"Error: {e}"}, cors_origin)


def resp(code, body, origin):
    return {
        "statusCode": code,
        "headers": {
            "Access-Control-Allow-Origin": origin,
            "Access-Control-Allow-Headers": "content-type,x-api-key",
            "Access-Control-Allow-Methods": "GET,OPTIONS",
        },
        "body": json.dumps(body),
    }
