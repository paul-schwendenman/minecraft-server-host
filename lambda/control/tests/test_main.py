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
