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
    }
