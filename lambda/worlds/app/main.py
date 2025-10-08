import os, json, boto3, botocore

S3 = boto3.client("s3")
BUCKET = os.environ["MAPS_BUCKET"]
PREFIX = os.environ.get("MAP_PREFIX", "maps/")
CORS = os.environ.get("CORS_ORIGIN", "*")
BASE_PATH = os.environ.get("BASE_PATH", "")


def _list_prefixes(prefix):
    """Return unique first-level folders under prefix without crawling everything."""
    try:
        paginator = S3.get_paginator("list_objects_v2")
        worlds = set()
        for page in paginator.paginate(
            Bucket=BUCKET,
            Prefix=prefix,
            Delimiter="/",  # <- critical for shallow listing
        ):
            for p in page.get("CommonPrefixes", []):
                worlds.add(p["Prefix"])
        return sorted(worlds)
    except Exception as e:
        print(f"[ERROR] listing prefixes: {e}")
        return []



def _dimensions(world):
    """Return list of dimensions for a given world name."""
    dim_prefixes = _list_prefixes(f"{PREFIX}{world}/")
    dims = []
    for dp in dim_prefixes:
        dim = dp.split("/")[-2]  # get folder name
        dims.append({
            "id": dim,
            "mapUrl": f"{BASE_PATH}/maps/{world}/{dim}/",
            "previewUrl": f"{BASE_PATH}/maps/{world}/{dim}/preview.png",
        })
    return dims


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
            world_prefixes = _list_prefixes(PREFIX)
            worlds = [p.split("/")[-2] for p in world_prefixes]
            body = {"worlds": worlds}
        elif len(parts) >= 3:  # /api/worlds/<world>/maps
            world = parts[2]
            body = {"world": world, "dimensions": _dimensions(world)}
        else:
            body = {"error": "invalid path"}

        return {"statusCode": 200, "headers": headers, "body": json.dumps(body)}

    except Exception as e:
        print(f"[ERROR] Unhandled exception: {e}")
        return {
            "statusCode": 500,
            "headers": headers,
            "body": json.dumps({"message": "Internal Server Error", "error": str(e)}),
        }
