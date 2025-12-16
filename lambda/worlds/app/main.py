import os
import json
import boto3
from botocore.exceptions import ClientError

s3 = boto3.client("s3", region_name=os.environ.get("AWS_REGION"))
MAPS_BUCKET = os.environ["MAPS_BUCKET"]
BASE_URL = os.environ.get("BASE_URL", f"https://{MAPS_BUCKET}.s3.{os.environ.get('AWS_REGION')}.amazonaws.com")
MAP_PREFIX = os.environ.get("MAP_PREFIX", "maps/")
CORS_ORIGIN = os.environ.get("CORS_ORIGIN", "*")


def read_json_from_s3(key: str):
    """Read a JSON file from S3 and parse it."""
    try:
        resp = s3.get_object(Bucket=MAPS_BUCKET, Key=key)
        body = resp["Body"].read().decode("utf-8")
        return json.loads(body)
    except ClientError as e:
        if e.response["Error"]["Code"] == "NoSuchKey":
            print(f"Missing key: {key}")
        else:
            print(f"S3 error reading {key}: {e}")
        return None


def make_response(status: int, body: dict):
    return {
        "statusCode": status,
        "headers": {
            "Content-Type": "application/json",
            "Access-Control-Allow-Origin": CORS_ORIGIN,
            "Access-Control-Allow-Methods": "GET,OPTIONS",
        },
        "body": json.dumps(body, indent=2),
    }


def lambda_handler(event, context):
    path = event.get("rawPath") or event.get("path", "/")
    method = event.get("requestContext", {}).get("http", {}).get("method", event.get("httpMethod", "GET"))

    if method == "OPTIONS":
        return make_response(200, {})

    try:
        # /api/worlds
        if path == "/api/worlds":
            data = read_json_from_s3(f"{MAP_PREFIX}world_manifest.json")
            if not data:
                return make_response(404, {"error": "world_manifest.json not found"})

            enriched = []
            for w in data:
                world_name = w.get("world", "")
                # Use preview field if present, otherwise default to {world}/preview.png
                preview_path = w.get("preview", f"{world_name}/preview.png")
                enriched.append({
                    **w,
                    "previewUrl": f"{BASE_URL}/{MAP_PREFIX}{preview_path}",
                    "mapUrl": f"{BASE_URL}/{MAP_PREFIX}{world_name}/"
                })

            return make_response(200, enriched)

        # /api/worlds/{name}
        if path.startswith("/api/worlds/") and path.count("/") == 3:
            name = path.split("/")[3]
            world = read_json_from_s3(f"{MAP_PREFIX}{name}/manifest.json")
            if not world:
                return make_response(404, {"error": f"World '{name}' not found"})

            world["previewUrl"] = f"{BASE_URL}/{MAP_PREFIX}{name}/preview.png"
            enriched_maps = []
            for m in world.get("maps", []):
                # Use map name for path (matches outputSubdir in map config)
                map_name = m.get("name", "")
                enriched_maps.append({
                    **m,
                    "previewUrl": f"{BASE_URL}/{MAP_PREFIX}{name}/{map_name}/preview.png",
                    "mapUrl": f"{BASE_URL}/{MAP_PREFIX}{name}/{map_name}/"
                })
            world["maps"] = enriched_maps

            return make_response(200, world)

        # /api/worlds/{name}/{map}
        parts = path.split("/")
        if len(parts) == 5 and parts[1:3] == ["api", "worlds"]:
            name, map_name = parts[3], parts[4]
            map_data = read_json_from_s3(f"{MAP_PREFIX}{name}/{map_name}/manifest.json")

            if not map_data:
                return make_response(404, {"error": f"Map '{map_name}' not found"})

            map_data["previewUrl"] = f"{BASE_URL}/{MAP_PREFIX}{name}/{map_name}/preview.png"
            map_data["mapUrl"] = f"{BASE_URL}/{MAP_PREFIX}{name}/{map_name}/"
            return make_response(200, map_data)

        return make_response(404, {"error": "Not found"})

    except Exception as e:
        print("Handler error:", e)
        return make_response(500, {"error": "Internal Server Error", "details": str(e)})
