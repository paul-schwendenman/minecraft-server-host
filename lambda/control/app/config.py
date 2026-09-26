"""Centralized configuration for the Minecraft control Lambda."""

import os
from functools import lru_cache


@lru_cache
def get_settings():
    """Read environment variables once and expose as a simple object."""
    return {
        "INSTANCE_ID": os.environ["INSTANCE_ID"],
        "REGION": os.environ.get("REGION", "us-east-2"),
        "DNS_NAME": os.environ.get("DNS_NAME", ""),
        "ZONE_ID": os.environ.get("ZONE_ID"),
        "CORS_ORIGIN": os.environ.get("CORS_ORIGIN", "*"),
        "AWS_LOCAL": os.environ.get("AWS_LOCAL") == "1",
        "MAPS_BUCKET": os.environ.get("MAPS_BUCKET", ""),
        "MAP_PREFIX": os.environ.get("MAP_PREFIX", "maps/"),
        # What the instance starts when ActiveWorld is unset (MC_DEFAULT_WORLD there)
        "DEFAULT_WORLD": os.environ.get("DEFAULT_WORLD", "default"),
    }
