import os
import json
import boto3
from botocore.exceptions import ClientError

s3 = boto3.client("s3", region_name=os.environ.get("AWS_REGION"))
BUCKET = os.environ["WORLD_BUCKET"]
BASE_URL = os.environ.get("BASE_URL", f"https://{BUCKET}.s3.{os.environ.get('AWS_REGION')}.amazonaws.com")
CORS_ORIGIN = os.environ.get("CORS_ORIGIN", "*")


def read_json_from_s3(key: str):
    """Read a JSON file from S3 and parse it."""
    try:
        resp = s3.get_object(Bucket=BUCKET, Key=key)
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


def handler(event, context):
    path = event.get("rawPath") or event.get("path", "/")
    method = event.get("requestContext", {}).get("http", {}).get("method", event.get("httpMethod", "GET"))

    if method == "OPTIONS":
        return make_response(200, {})

    try:
        # /api/worlds
        if path == "/api/worlds":
            data = read_json_from_s3("world_manifest.json")
            if not data:
                return make_response(404, {"error": "world_manifest.json not found"})

            enriched = []
            for w in data:
                enriched.append({
                    **w,
                    "previewUrl": f"{BASE_URL}/{w.get('preview')}",
                    "mapUrl": f"{BASE_URL}/worlds/{w['world']}/"
                })

            return make_response(200, enriched)

        # /api/worlds/{name}
        if path.startswith("/api/worlds/") and path.count("/") == 3:
            name = path.split("/")[3]
            world = read_json_from_s3(f"worlds/{name}/manifest.json")
            if not world:
                return make_response(404, {"error": f"World '{name}' not found"})

            world["previewUrl"] = f"{BASE_URL}/worlds/{name}/preview.png"
            dims = []
            for d in world.get("dimensions", []):
                dims.append({
                    **d,
                    "previewUrl": f"{BASE_URL}/worlds/{name}/{d['name']}/preview.png",
                    "mapUrl": f"{BASE_URL}/worlds/{name}/{d['name']}/"
                })
            world["dimensions"] = dims

            return make_response(200, world)

        # /api/worlds/{name}/{dimension}
        parts = path.split("/")
        if len(parts) == 5 and parts[1:3] == ["api", "worlds"]:
            name, dim = parts[3], parts[4]
            dim_data = read_json_from_s3(f"worlds/{name}/{dim}/manifest.json")
            if not dim_data:
                return make_response(404, {"error": f"Dimension '{dim}' not found"})

            dim_data["previewUrl"] = f"{BASE_URL}/worlds/{name}/{dim}/preview.png"
            dim_data["mapUrl"] = f"{BASE_URL}/worlds/{name}/{dim}/"
            return make_response(200, dim_data)

        return make_response(404, {"error": "Not found"})

    except Exception as e:
        print("Handler error:", e)
        return make_response(500, {"error": "Internal Server Error", "details": str(e)})
