"""Tests for the details lambda handler."""

import json
import os
import socket
import pytest
from unittest.mock import patch, MagicMock


class TestRespFunction:
    """Tests for the resp helper function."""

    def test_resp_200(self):
        """Test successful response format."""
        from app.main import resp

        result = resp(200, {"players": 5}, "*")

        assert result["statusCode"] == 200
        assert json.loads(result["body"]) == {"players": 5}
        assert result["headers"]["Access-Control-Allow-Origin"] == "*"

    def test_resp_400(self):
        """Test error response format."""
        from app.main import resp

        result = resp(400, {"message": "Bad Request"}, "https://example.com")

        assert result["statusCode"] == 400
        assert json.loads(result["body"]) == {"message": "Bad Request"}
        assert result["headers"]["Access-Control-Allow-Origin"] == "https://example.com"

    def test_resp_headers(self):
        """Test CORS headers are set correctly."""
        from app.main import resp

        result = resp(200, {}, "*")

        assert "Access-Control-Allow-Origin" in result["headers"]
        assert "Access-Control-Allow-Headers" in result["headers"]
        assert "Access-Control-Allow-Methods" in result["headers"]
        assert "content-type" in result["headers"]["Access-Control-Allow-Headers"]
        assert "GET" in result["headers"]["Access-Control-Allow-Methods"]


class TestLambdaHandler:
    """Tests for the lambda_handler function."""

    def test_missing_hostname(self):
        """Test handler returns 400 when hostname is missing."""
        from app.main import lambda_handler

        event = {"queryStringParameters": {}}
        result = lambda_handler(event, None)

        assert result["statusCode"] == 400
        body = json.loads(result["body"])
        assert "Missing" in body["message"]

    def test_missing_query_params(self):
        """Test handler returns 400 when query params are None."""
        from app.main import lambda_handler

        event = {"queryStringParameters": None}
        result = lambda_handler(event, None)

        assert result["statusCode"] == 400

    def test_empty_event(self):
        """Test handler returns 400 with empty event."""
        from app.main import lambda_handler

        event = {}
        result = lambda_handler(event, None)

        assert result["statusCode"] == 400

    @patch("app.main.JavaServer")
    def test_successful_status(self, mock_server_class):
        """Test handler returns server status on success."""
        # Set up mock
        mock_status = MagicMock()
        mock_status.raw = {
            "players": {"online": 5, "max": 20},
            "version": {"name": "1.20.4"},
        }

        mock_server = MagicMock()
        mock_server.status.return_value = mock_status
        mock_server_class.lookup.return_value = mock_server

        from app.main import lambda_handler

        event = {"queryStringParameters": {"hostname": "minecraft.example.com"}}
        result = lambda_handler(event, None)

        assert result["statusCode"] == 200
        body = json.loads(result["body"])
        assert body["players"]["online"] == 5
        mock_server_class.lookup.assert_called_once_with("minecraft.example.com")

    @patch("app.main.JavaServer")
    def test_server_timeout(self, mock_server_class):
        """Test handler returns 503 on timeout."""
        mock_server = MagicMock()
        mock_server.status.side_effect = socket.timeout("Connection timed out")
        mock_server_class.lookup.return_value = mock_server

        from app.main import lambda_handler

        event = {"queryStringParameters": {"hostname": "minecraft.example.com"}}
        result = lambda_handler(event, None)

        assert result["statusCode"] == 503
        body = json.loads(result["body"])
        assert "Timeout" in body["message"]

    @patch("app.main.JavaServer")
    def test_server_error(self, mock_server_class):
        """Test handler returns 500 on other errors."""
        mock_server = MagicMock()
        mock_server.status.side_effect = Exception("Connection refused")
        mock_server_class.lookup.return_value = mock_server

        from app.main import lambda_handler

        event = {"queryStringParameters": {"hostname": "minecraft.example.com"}}
        result = lambda_handler(event, None)

        assert result["statusCode"] == 500
        body = json.loads(result["body"])
        assert "Error" in body["message"]

    @patch("app.main.JavaServer")
    def test_cors_origin_from_env(self, mock_server_class):
        """Test CORS origin is read from environment."""
        os.environ["CORS_ORIGIN"] = "https://my-site.com"

        mock_status = MagicMock()
        mock_status.raw = {}
        mock_server = MagicMock()
        mock_server.status.return_value = mock_status
        mock_server_class.lookup.return_value = mock_server

        from app.main import lambda_handler

        event = {"queryStringParameters": {"hostname": "test.com"}}
        result = lambda_handler(event, None)

        assert result["headers"]["Access-Control-Allow-Origin"] == "https://my-site.com"

        # Cleanup
        del os.environ["CORS_ORIGIN"]

    def test_cors_origin_default(self):
        """Test CORS origin defaults to * when not set."""
        os.environ.pop("CORS_ORIGIN", None)

        from app.main import lambda_handler

        event = {"queryStringParameters": {}}
        result = lambda_handler(event, None)

        assert result["headers"]["Access-Control-Allow-Origin"] == "*"


class TestHostnameParsing:
    """Tests for hostname parameter handling."""

    @patch("app.main.JavaServer")
    def test_hostname_with_port(self, mock_server_class):
        """Test hostname with port is passed correctly."""
        mock_status = MagicMock()
        mock_status.raw = {}
        mock_server = MagicMock()
        mock_server.status.return_value = mock_status
        mock_server_class.lookup.return_value = mock_server

        from app.main import lambda_handler

        event = {"queryStringParameters": {"hostname": "minecraft.example.com:25565"}}
        lambda_handler(event, None)

        mock_server_class.lookup.assert_called_once_with("minecraft.example.com:25565")

    @patch("app.main.JavaServer")
    def test_ip_address_hostname(self, mock_server_class):
        """Test IP address as hostname."""
        mock_status = MagicMock()
        mock_status.raw = {}
        mock_server = MagicMock()
        mock_server.status.return_value = mock_status
        mock_server_class.lookup.return_value = mock_server

        from app.main import lambda_handler

        event = {"queryStringParameters": {"hostname": "192.168.1.100"}}
        lambda_handler(event, None)

        mock_server_class.lookup.assert_called_once_with("192.168.1.100")
