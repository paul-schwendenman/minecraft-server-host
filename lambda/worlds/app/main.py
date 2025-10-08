import os, json, boto3

S3 = boto3.client("s3")
BUCKET = os.environ["MAPS_BUCKET"]
PREFIX = os.environ.get("MAP_PREFIX", "maps/")
CORS = os.environ.get("CORS_ORIGIN", "*")
BASE_PATH = os.environ.get("BASE_PATH", "")

def _list_prefixes(prefix):
    paginator = S3.get_paginator("list_objects_v2")
    prefixes = []
    for page in paginator.paginate(Bucket=BUCKET, Prefix=prefix, Delimiter="/"):
        prefixes += [p["Prefix"] for p in page.get("CommonPrefixes", [])]
    return prefixes

def _dimensions(world):
    dim_prefixes = _list_prefixes(f"{PREFIX}{world}/")
    dims = []
    for dp in dim_prefixes:
        dim = dp.split("/")[-2]
        dims.append({
            "id": dim,
            "mapUrl": f"{BASE_PATH}/maps/{world}/{dim}/",
            "previewUrl": f"{BASE_PATH}/maps/{world}/{dim}/preview.png"
        })
    return dims

def handler(event, _ctx):
    headers = {
        "Access-Control-Allow-Origin": CORS,
        "Access-Control-Allow-Methods": "GET,OPTIONS",
        "Access-Control-Allow-Headers": "Content-Type",
    }

    method = event.get("requestContext", {}).get("http", {}).get("method")
    path = event.get("rawPath", "")
    if method == "OPTIONS":
        return {"statusCode": 200, "headers": headers, "body": ""}

    parts = [p for p in path.split("/") if p]
    if len(parts) == 2:  # /api/worlds
        worlds = [p.split("/")[-2] for p in _list_prefixes(PREFIX)]
        body = {"worlds": worlds}
    elif len(parts) >= 3:  # /api/worlds/<world>/maps
        world = parts[2]
        body = {"world": world, "dimensions": _dimensions(world)}
    else:
        body = {"error": "invalid path"}

    return {"statusCode": 200, "headers": headers, "body": json.dumps(body)}
