"""Tests for querying the Minecraft server."""

from unittest.mock import MagicMock, patch

import pytest

from app import server


def test_players_online_pings_the_minecraft_port():
    status = MagicMock()
    status.players.online = 2
    with patch("app.server.JavaServer") as mock_server:
        mock_server.return_value.status.return_value = status
        assert server.players_online("203.0.113.7") == 2
    mock_server.assert_called_once_with("203.0.113.7", 25565, timeout=3)


def test_players_online_raises_when_the_server_does_not_answer():
    with patch("app.server.JavaServer") as mock_server:
        mock_server.return_value.status.side_effect = TimeoutError("timed out")
        with pytest.raises(TimeoutError):
            server.players_online("203.0.113.7")
