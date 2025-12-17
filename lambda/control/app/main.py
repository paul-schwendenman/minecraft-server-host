"""Minecraft Server control API – FastAPI + Mangum Lambda handler."""

import logging
from fastapi import FastAPI
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
def start():
    return aws.start_instance(settings["INSTANCE_ID"])


@app.post("/stop")
def stop():
    return aws.stop_instance(settings["INSTANCE_ID"])


@app.post("/syncdns")
def syncdns():
    if settings["DNS_NAME"]:
        return aws.update_dns(settings["INSTANCE_ID"], settings["DNS_NAME"])
    return {"message": "DNS not configured"}
