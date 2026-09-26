"""Minecraft Server control API – FastAPI + Mangum Lambda handler."""

import logging
from typing import Optional
from fastapi import FastAPI, HTTPException
from mangum import Mangum
from .config import get_settings
from . import aws_utils as aws

settings = get_settings()
logger = logging.getLogger()
logger.setLevel(logging.INFO)

app = FastAPI(title="Minecraft Server API", version="1.0")
handler = Mangum(app, api_gateway_base_path="/api")


@app.get("/")
def root():
    return {"message": "Minecraft Server API"}


@app.get("/status")
def status():
    return aws.describe_state(settings["INSTANCE_ID"], settings["DNS_NAME"])


@app.post("/start")
def start(world: Optional[str] = None):
    """Start the instance, optionally choosing which world it boots into."""
    if world is None:
        return aws.start_instance(settings["INSTANCE_ID"])

    try:
        worlds = aws.list_worlds()
    except Exception as e:
        logger.error("Could not read the world list: %s", e)
        raise HTTPException(503, "World list unavailable")
    if world not in worlds:
        raise HTTPException(400, f"Unknown world: {world}")

    instance = aws.get_instance(settings["INSTANCE_ID"])
    state = instance["State"]["Name"]
    active = aws.get_active_world(instance)

    if state in ("pending", "running"):
        if world != active:
            raise HTTPException(
                409, f"Server is running {active}; stop it before switching worlds"
            )
        return {"message": "Success", "world": world}
    if state != "stopped":
        raise HTTPException(409, f"Server is {state}; try again once it has stopped")

    aws.set_active_world(settings["INSTANCE_ID"], world)
    aws.start_instance(settings["INSTANCE_ID"])
    return {"message": "Success", "world": world}


@app.post("/stop")
def stop():
    return aws.stop_instance(settings["INSTANCE_ID"])


@app.post("/syncdns")
def syncdns():
    if settings["DNS_NAME"]:
        return aws.update_dns(settings["INSTANCE_ID"], settings["DNS_NAME"])
    return {"message": "DNS not configured"}
