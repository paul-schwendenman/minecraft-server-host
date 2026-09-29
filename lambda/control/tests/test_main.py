"""Tests for the control lambda FastAPI endpoints."""

import os
import pytest
from unittest.mock import patch, MagicMock


class TestConfigModule:
    """Tests for config.py"""

    def test_get_settings_returns_dict(self):
        """Test that get_settings returns expected keys."""
        os.environ["INSTANCE_ID"] = "i-test123"
        os.environ["REGION"] = "us-east-2"

        # Clear the cache before testing
        from app.config import get_settings
        get_settings.cache_clear()

        settings = get_settings()

        assert "INSTANCE_ID" in settings
        assert "REGION" in settings
        assert "DNS_NAME" in settings
        assert "ZONE_ID" in settings
        assert "CORS_ORIGIN" in settings
        assert "AWS_LOCAL" in settings

    def test_get_settings_defaults(self):
        """Test default values."""
        os.environ["INSTANCE_ID"] = "i-test123"
        os.environ.pop("REGION", None)
        os.environ.pop("DNS_NAME", None)
        os.environ.pop("CORS_ORIGIN", None)

        from app.config import get_settings
        get_settings.cache_clear()

        settings = get_settings()

        assert settings["REGION"] == "us-east-2"
        assert settings["DNS_NAME"] == ""
        assert settings["CORS_ORIGIN"] == "*"

    def test_get_settings_aws_local(self):
        """Test AWS_LOCAL flag parsing."""
        os.environ["INSTANCE_ID"] = "i-test123"

        from app.config import get_settings

        os.environ["AWS_LOCAL"] = "1"
        get_settings.cache_clear()
        assert get_settings()["AWS_LOCAL"] is True

        os.environ["AWS_LOCAL"] = "0"
        get_settings.cache_clear()
        assert get_settings()["AWS_LOCAL"] is False

        os.environ["AWS_LOCAL"] = ""
        get_settings.cache_clear()
        assert get_settings()["AWS_LOCAL"] is False


class TestFastAPIEndpoints:
    """Tests for FastAPI endpoints using TestClient."""

    @pytest.fixture(autouse=True)
    def setup_env(self):
        """Set up environment before each test."""
        os.environ["INSTANCE_ID"] = "i-test123"
        os.environ["REGION"] = "us-east-2"
        os.environ["DNS_NAME"] = "minecraft.example.com"
        os.environ["ZONE_ID"] = "Z123456"
        os.environ["AWS_LOCAL"] = ""

        # Clear config cache
        from app.config import get_settings
        get_settings.cache_clear()

    def test_root_endpoint(self):
        """Test the root endpoint returns expected message."""
        from fastapi.testclient import TestClient
        from app.main import app

        client = TestClient(app)
        response = client.get("/")

        assert response.status_code == 200
        assert response.json() == {"message": "Minecraft Server API"}

    @patch("app.main.aws.describe_state")
    def test_status_endpoint(self, mock_describe):
        """Test the status endpoint calls describe_state."""
        mock_describe.return_value = {
            "instance": {"state": "running", "ip_address": "1.2.3.4"},
            "dns_record": {"name": "minecraft.example.com", "value": "1.2.3.4", "type": "A"},
        }

        from fastapi.testclient import TestClient
        from app.main import app

        client = TestClient(app)
        response = client.get("/status")

        assert response.status_code == 200
        data = response.json()
        assert data["instance"]["state"] == "running"
        assert data["instance"]["ip_address"] == "1.2.3.4"
        mock_describe.assert_called_once()

    @patch("app.main.aws.start_instance")
    def test_start_endpoint(self, mock_start):
        """Test the start endpoint calls start_instance."""
        mock_start.return_value = {"message": "Success"}

        from fastapi.testclient import TestClient
        from app.main import app

        client = TestClient(app)
        response = client.post("/start")

        assert response.status_code == 200
        assert response.json() == {"message": "Success"}
        mock_start.assert_called_once_with("i-test123")

    def _instance(self, state, world=None):
        tags = [{"Key": "ActiveWorld", "Value": world}] if world else []
        return {"State": {"Name": state}, "Tags": tags, "PublicIpAddress": "203.0.113.7"}

    def _start_world(self, world, state="stopped", active=None, worlds=None, players=0):
        """POST /start?world= with the AWS helpers mocked; returns (response, mocks).

        players is what the server list ping reports, or an exception it raises.
        """
        from fastapi.testclient import TestClient
        from app.main import app

        ping = {"side_effect": players} if isinstance(players, Exception) else {"return_value": players}
        with patch("app.main.aws.list_worlds", return_value=worlds or ["default", "old"]), \
             patch("app.main.aws.get_instance", return_value=self._instance(state, active)), \
             patch("app.main.aws.set_active_world") as mock_set, \
             patch("app.main.aws.start_instance") as mock_start, \
             patch("app.main.server.players_online", **ping) as self.mock_ping:
            response = TestClient(app).post("/start", params={"world": world})
        return response, mock_set, mock_start

    def test_start_world_when_stopped_sets_tag_then_starts(self):
        response, mock_set, mock_start = self._start_world("old")

        assert response.status_code == 200
        assert response.json() == {"message": "Success", "world": "old"}
        mock_set.assert_called_once_with("i-test123", "old")
        mock_start.assert_called_once_with("i-test123")

    def test_start_unknown_world_is_rejected(self):
        response, mock_set, mock_start = self._start_world("nope")

        assert response.status_code == 400
        mock_set.assert_not_called()
        mock_start.assert_not_called()

    def test_switch_while_running_and_empty_sets_tag_only(self):
        response, mock_set, mock_start = self._start_world(
            "old", state="running", active="default"
        )

        assert response.status_code == 202
        assert response.json() == {"message": "Switching", "world": "old"}
        self.mock_ping.assert_called_once_with("203.0.113.7")
        mock_set.assert_called_once_with("i-test123", "old")
        mock_start.assert_not_called()

    def test_switch_refused_with_players_online(self):
        response, mock_set, mock_start = self._start_world(
            "old", state="running", active="default", players=2
        )

        assert response.status_code == 409
        assert "2 players are online" in response.json()["detail"]
        mock_set.assert_not_called()
        mock_start.assert_not_called()

    def test_switch_refused_with_one_player_online(self):
        response, mock_set, _ = self._start_world(
            "old", state="running", active="default", players=1
        )

        assert response.status_code == 409
        assert "1 player is online" in response.json()["detail"]
        mock_set.assert_not_called()

    def test_switch_refused_when_server_not_answering(self):
        response, mock_set, _ = self._start_world(
            "old", state="running", active="default", players=TimeoutError("timed out")
        )

        assert response.status_code == 409
        assert "isn't answering" in response.json()["detail"]
        mock_set.assert_not_called()

    def test_switch_while_pending_conflicts(self):
        response, mock_set, mock_start = self._start_world(
            "old", state="pending", active="default"
        )

        assert response.status_code == 409
        self.mock_ping.assert_not_called()
        mock_set.assert_not_called()
        mock_start.assert_not_called()

    def test_start_untagged_running_instance_counts_as_default(self):
        response, mock_set, _ = self._start_world("default", state="running")

        assert response.status_code == 200
        mock_set.assert_not_called()

    def test_start_same_world_while_running_is_a_no_op(self):
        response, mock_set, mock_start = self._start_world(
            "old", state="running", active="old"
        )

        assert response.status_code == 200
        self.mock_ping.assert_not_called()
        mock_set.assert_not_called()
        mock_start.assert_not_called()

    def test_start_world_while_stopping_conflicts(self):
        response, mock_set, mock_start = self._start_world("old", state="stopping")

        assert response.status_code == 409
        mock_set.assert_not_called()
        mock_start.assert_not_called()

    def test_start_world_when_world_list_unavailable(self):
        from fastapi.testclient import TestClient
        from app.main import app

        with patch("app.main.aws.list_worlds", side_effect=Exception("boom")), \
             patch("app.main.aws.start_instance") as mock_start:
            response = TestClient(app).post("/start", params={"world": "old"})

        assert response.status_code == 503
        mock_start.assert_not_called()

    @patch("app.main.aws.stop_instance")
    def test_stop_endpoint(self, mock_stop):
        """Test the stop endpoint calls stop_instance."""
        mock_stop.return_value = {"message": "Success"}

        from fastapi.testclient import TestClient
        from app.main import app

        client = TestClient(app)
        response = client.post("/stop")

        assert response.status_code == 200
        assert response.json() == {"message": "Success"}
        mock_stop.assert_called_once_with("i-test123")

    @patch("app.main.aws.update_dns")
    def test_syncdns_endpoint(self, mock_update):
        """Test the syncdns endpoint calls update_dns."""
        mock_update.return_value = {"message": "Success"}

        from fastapi.testclient import TestClient
        from app.main import app

        client = TestClient(app)
        response = client.post("/syncdns")

        assert response.status_code == 200
        assert response.json() == {"message": "Success"}
        mock_update.assert_called_once()

    @patch("app.main.settings", {"INSTANCE_ID": "i-test", "DNS_NAME": ""})
    def test_syncdns_no_dns_configured(self):
        """Test syncdns returns message when DNS not configured."""
        from fastapi.testclient import TestClient
        from app.main import app

        client = TestClient(app)
        response = client.post("/syncdns")

        assert response.status_code == 200
        assert response.json() == {"message": "DNS not configured"}


class TestMangumHandler:
    """Tests for the Mangum Lambda handler."""

    def test_handler_exists(self):
        """Test that the handler is exported."""
        from app.main import handler
        assert handler is not None

    def test_app_title(self):
        """Test FastAPI app configuration."""
        from app.main import app
        assert app.title == "Minecraft Server API"
        assert app.version == "1.0"
