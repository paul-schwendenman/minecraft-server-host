"""Minecraft Server control API – FastAPI + Mangum Lambda handler."""

import logging
from typing import Optional
from fastapi import FastAPI, HTTPException
from fastapi.responses import JSONResponse
from mangum import Mangum
from .config import get_settings
from . import aws_utils as aws
from . import server

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
    """Start the instance, optionally choosing which world it boots into.

    Only for a stopped server: a different world on a running one is 409 (use
    /switch). Keeping the two apart means a page that saw the server stopped
    can't switch the world under someone who started it meanwhile.
    """
    if world is None:
        return aws.start_instance(settings["INSTANCE_ID"])

    check_known_world(world)
    instance = aws.get_instance(settings["INSTANCE_ID"])
    state = instance["State"]["Name"]
    active = aws.get_active_world(instance)

    if state in ("pending", "running"):
        if world != active:
            raise HTTPException(
                409, f"Server is running {active}; switch worlds instead of starting"
            )
        return {"message": "Success", "world": world}
    if state != "stopped":
        raise HTTPException(409, f"Server is {state}; try again once it has stopped")

    aws.set_active_world(settings["INSTANCE_ID"], world)
    aws.start_instance(settings["INSTANCE_ID"])
    return {"message": "Success", "world": world}


@app.post("/switch")
def switch(world: str):
    """Switch a running server to another world.

    Sets the tag; the instance's watcher (minecraft-world-watch) does the
    switch within a minute, so the response is 202. Refused (409) while anyone's
    online, while the world is loading, and when the server isn't running (use
    /start).
    """
    check_known_world(world)
    instance = aws.get_instance(settings["INSTANCE_ID"])
    state = instance["State"]["Name"]
    active = aws.get_active_world(instance)

    if state == "pending":
        raise HTTPException(409, "Server is starting; try again once it's running")
    if state != "running":
        raise HTTPException(409, f"Server is {state}; start it instead of switching")
    if world == active:
        return {"message": "Success", "world": world}

    check_server_empty(instance)
    aws.set_active_world(settings["INSTANCE_ID"], world)
    return JSONResponse(status_code=202, content={"message": "Switching", "world": world})


def check_known_world(world: str) -> None:
    """Refuse a world that isn't in the published world manifest."""
    try:
        worlds = aws.list_worlds()
    except Exception as e:
        logger.error("Could not read the world list: %s", e)
        raise HTTPException(503, "World list unavailable")
    if world not in worlds:
        raise HTTPException(400, f"Unknown world: {world}")


def check_server_empty(instance) -> None:
    """Refuse a switch unless the server answers and has nobody online.

    Checked before the tag is set: a refused tag would otherwise sit pending
    and switch whenever the server next empties, or at the next boot.
    """
    ip = instance.get("PublicIpAddress")
    try:
        players = server.players_online(ip)
    except Exception as e:
        logger.info("Server list ping to %s failed: %s", ip, e)
        raise HTTPException(
            409, "Server isn't answering yet; try again once the world has loaded"
        )
    if players:
        who = "1 player is" if players == 1 else f"{players} players are"
        raise HTTPException(409, f"{who} online; switch once the server is empty")


@app.post("/stop")
def stop():
    return aws.stop_instance(settings["INSTANCE_ID"])


@app.post("/syncdns")
def syncdns():
    if settings["DNS_NAME"]:
        return aws.update_dns(settings["INSTANCE_ID"], settings["DNS_NAME"])
    return {"message": "DNS not configured"}
