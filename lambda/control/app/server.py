"""Queries the Minecraft server itself, rather than AWS."""

from mcstatus import JavaServer

MINECRAFT_PORT = 25565


def players_online(host: str, timeout: float = 3) -> int:
    """Return how many players are online, from the server list ping.

    Raises if the server doesn't answer, e.g. while a world is still loading.
    """
    return JavaServer(host, MINECRAFT_PORT, timeout=timeout).status().players.online
