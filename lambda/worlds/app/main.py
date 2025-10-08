import os
import json
import boto3
from datetime import datetime, timezone

S3 = boto3.client("s3")
BUCKET = os.environ["MAPS_BUCKET"]
PREFIX = os.environ.get("MAP_PREFIX", "maps/")
CORS = os.environ.get("CORS_ORIGIN", "*")
BASE_PATH = os.environ.get("BASE_PATH", "")


def lambda_handler(event, _ctx):
    headers = {
        "Access-Control-Allow-Origin": CORS,
        "Access-Control-Allow-Methods": "GET,OPTIONS",
        "Access-Control-Allow-Headers": "Content-Type",
    }

    try:
        method = event.get("requestContext", {}).get("http", {}).get("method")
        path = event.get("rawPath", "")
        if method == "OPTIONS":
            return {"statusCode": 200, "headers": headers, "body": ""}

        parts = [p for p in path.split("/") if p]

        if len(parts) == 2:  # /api/worlds
            body = {"worlds": list_worlds()}

        elif len(parts) >= 3:
            world = parts[2]
            body = world_details(world)

        else:
            body = {"error": "invalid path", "path": path, "parts": parts}

        return {
            "statusCode": 200,
            "headers": headers,
            "body": json.dumps(body),
        }

    except Exception as e:
        print(f"[ERROR] {e}")
        return {
            "statusCode": 500,
            "headers": headers,
            "body": json.dumps({"message": "Internal Server Error", "error": str(e)}),
        }


# ---------------------------------------------------------------------
# Helper functions
# ---------------------------------------------------------------------

def list_worlds():
    """Return a list of worlds under maps/, with metadata for each."""
    worlds = []
    paginator = S3.get_paginator("list_objects_v2")

    for page in paginator.paginate(Bucket=BUCKET, Prefix=PREFIX, Delimiter="/"):
        for p in page.get("CommonPrefixes", []):
            world_prefix = p["Prefix"]  # e.g. "maps/default/"
            world_name = world_prefix[len(PREFIX):-1]
            info = get_world_info(world_name)
            worlds.append(info)
    return worlds


def get_world_info(world_name):
    """Summarize a single world directory."""
    map_url = f"{BASE_PATH}/maps/{world_name}/"
    preview_url = f"{map_url}overworld/preview.png"
    last_updated = get_last_modified(f"{PREFIX}{world_name}/overworld/index.html")

    return {
        "name": world_name,
        "id": world_name,
        "map_url": map_url,
        "preview_url": preview_url,
        "last_updated": last_updated,
    }


def world_details(world_name):
    """Return details for /api/worlds/<world> or /api/worlds/<world>/maps."""
    dims = []
    paginator = S3.get_paginator("list_objects_v2")
    dim_prefix = f"{PREFIX}{world_name}/"
    try:
        for page in paginator.paginate(Bucket=BUCKET, Prefix=dim_prefix, Delimiter="/"):
            for p in page.get("CommonPrefixes", []):
                dim_id = p["Prefix"].split("/")[-2]
                dims.append({
                    "id": dim_id,
                    "map_url": f"{BASE_PATH}/maps/{world_name}/{dim_id}/"
                })
    except Exception as e:
        print(f"[WARN] listing dimensions for {world_name}: {e}")

    return {"world": world_name, "dimensions": dims}


def get_last_modified(key):
    """Return last modified timestamp of key, or None."""
    try:
        resp = S3.head_object(Bucket=BUCKET, Key=key)
        ts = resp["LastModified"]
        return ts.astimezone(timezone.utc).isoformat()
    except S3.exceptions.ClientError:
        return None
