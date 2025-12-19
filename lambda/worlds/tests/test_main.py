"""Tests for the worlds lambda handler."""

import json
import os
import pytest
import boto3
from moto import mock_aws


@pytest.fixture(autouse=True)
def setup_env():
    """Set up environment variables for each test."""
    os.environ["AWS_REGION"] = "us-east-2"
    os.environ["AWS_ACCESS_KEY_ID"] = "testing"
    os.environ["AWS_SECRET_ACCESS_KEY"] = "testing"
    os.environ["AWS_DEFAULT_REGION"] = "us-east-2"
    os.environ["MAPS_BUCKET"] = "test-maps-bucket"
    os.environ["BASE_URL"] = "https://maps.example.com"
    os.environ["MAP_PREFIX"] = "maps/"
    os.environ["CORS_ORIGIN"] = "*"


class TestMakeResponse:
    """Tests for the make_response helper function."""

    @mock_aws
    def test_make_response_200(self):
        """Test successful response format."""
        # Need to set up S3 before importing the module
        s3 = boto3.client("s3", region_name="us-east-2")
        s3.create_bucket(
            Bucket="test-maps-bucket",
            CreateBucketConfiguration={"LocationConstraint": "us-east-2"},
        )

        import importlib
        import app.main as main_module

        importlib.reload(main_module)

        result = main_module.make_response(200, {"data": "test"})

        assert result["statusCode"] == 200
        assert json.loads(result["body"]) == {"data": "test"}
        assert result["headers"]["Content-Type"] == "application/json"
        assert result["headers"]["Access-Control-Allow-Origin"] == "*"

    @mock_aws
    def test_make_response_cors_headers(self):
        """Test CORS headers are set correctly."""
        s3 = boto3.client("s3", region_name="us-east-2")
        s3.create_bucket(
            Bucket="test-maps-bucket",
            CreateBucketConfiguration={"LocationConstraint": "us-east-2"},
        )

        import importlib
        import app.main as main_module

        importlib.reload(main_module)

        result = main_module.make_response(200, {})

        assert "Access-Control-Allow-Origin" in result["headers"]
        assert "Access-Control-Allow-Methods" in result["headers"]
        assert "GET" in result["headers"]["Access-Control-Allow-Methods"]


class TestReadJsonFromS3:
    """Tests for the read_json_from_s3 helper function."""

    @mock_aws
    def test_read_existing_json(self):
        """Test reading an existing JSON file from S3."""
        s3 = boto3.client("s3", region_name="us-east-2")
        s3.create_bucket(
            Bucket="test-maps-bucket",
            CreateBucketConfiguration={"LocationConstraint": "us-east-2"},
        )
        s3.put_object(
            Bucket="test-maps-bucket",
            Key="maps/test.json",
            Body=json.dumps({"key": "value"}),
        )

        import importlib
        import app.main as main_module

        importlib.reload(main_module)

        result = main_module.read_json_from_s3("maps/test.json")

        assert result == {"key": "value"}

    @mock_aws
    def test_read_missing_key(self):
        """Test reading a non-existent key returns None."""
        s3 = boto3.client("s3", region_name="us-east-2")
        s3.create_bucket(
            Bucket="test-maps-bucket",
            CreateBucketConfiguration={"LocationConstraint": "us-east-2"},
        )

        import importlib
        import app.main as main_module

        importlib.reload(main_module)

        result = main_module.read_json_from_s3("maps/nonexistent.json")

        assert result is None


class TestLambdaHandler:
    """Tests for the lambda_handler function."""

    @mock_aws
    def test_options_request(self):
        """Test OPTIONS request returns 200."""
        s3 = boto3.client("s3", region_name="us-east-2")
        s3.create_bucket(
            Bucket="test-maps-bucket",
            CreateBucketConfiguration={"LocationConstraint": "us-east-2"},
        )

        import importlib
        import app.main as main_module

        importlib.reload(main_module)

        event = {
            "rawPath": "/api/worlds",
            "requestContext": {"http": {"method": "OPTIONS"}},
        }
        result = main_module.lambda_handler(event, None)

        assert result["statusCode"] == 200

    @mock_aws
    def test_list_worlds(self):
        """Test GET /api/worlds returns world list with URLs."""
        s3 = boto3.client("s3", region_name="us-east-2")
        s3.create_bucket(
            Bucket="test-maps-bucket",
            CreateBucketConfiguration={"LocationConstraint": "us-east-2"},
        )
        s3.put_object(
            Bucket="test-maps-bucket",
            Key="maps/world_manifest.json",
            Body=json.dumps([
                {"world": "survival", "name": "Survival World"},
                {"world": "creative", "name": "Creative World", "preview": "creative/custom.png"},
            ]),
        )

        import importlib
        import app.main as main_module

        importlib.reload(main_module)

        event = {
            "rawPath": "/api/worlds",
            "requestContext": {"http": {"method": "GET"}},
        }
        result = main_module.lambda_handler(event, None)

        assert result["statusCode"] == 200
        body = json.loads(result["body"])
        assert len(body) == 2
        assert body[0]["world"] == "survival"
        assert body[0]["previewUrl"] == "https://maps.example.com/maps/survival/preview.png"
        assert body[0]["mapUrl"] == "https://maps.example.com/maps/survival/"
        # Custom preview path
        assert body[1]["previewUrl"] == "https://maps.example.com/maps/creative/custom.png"

    @mock_aws
    def test_list_worlds_not_found(self):
        """Test GET /api/worlds returns 404 when manifest missing."""
        s3 = boto3.client("s3", region_name="us-east-2")
        s3.create_bucket(
            Bucket="test-maps-bucket",
            CreateBucketConfiguration={"LocationConstraint": "us-east-2"},
        )

        import importlib
        import app.main as main_module

        importlib.reload(main_module)

        event = {
            "rawPath": "/api/worlds",
            "requestContext": {"http": {"method": "GET"}},
        }
        result = main_module.lambda_handler(event, None)

        assert result["statusCode"] == 404
        body = json.loads(result["body"])
        assert "not found" in body["error"]

    @mock_aws
    def test_get_world(self):
        """Test GET /api/worlds/{name} returns world details."""
        s3 = boto3.client("s3", region_name="us-east-2")
        s3.create_bucket(
            Bucket="test-maps-bucket",
            CreateBucketConfiguration={"LocationConstraint": "us-east-2"},
        )
        s3.put_object(
            Bucket="test-maps-bucket",
            Key="maps/survival/manifest.json",
            Body=json.dumps({
                "name": "Survival World",
                "maps": [
                    {"name": "overworld", "dimension": "minecraft:overworld"},
                    {"name": "nether", "dimension": "minecraft:the_nether"},
                ],
            }),
        )

        import importlib
        import app.main as main_module

        importlib.reload(main_module)

        event = {
            "rawPath": "/api/worlds/survival",
            "requestContext": {"http": {"method": "GET"}},
        }
        result = main_module.lambda_handler(event, None)

        assert result["statusCode"] == 200
        body = json.loads(result["body"])
        assert body["name"] == "Survival World"
        assert body["previewUrl"] == "https://maps.example.com/maps/survival/preview.png"
        assert len(body["maps"]) == 2
        assert body["maps"][0]["previewUrl"] == "https://maps.example.com/maps/survival/overworld/preview.png"
        assert body["maps"][0]["mapUrl"] == "https://maps.example.com/maps/survival/overworld/"

    @mock_aws
    def test_get_world_not_found(self):
        """Test GET /api/worlds/{name} returns 404 for missing world."""
        s3 = boto3.client("s3", region_name="us-east-2")
        s3.create_bucket(
            Bucket="test-maps-bucket",
            CreateBucketConfiguration={"LocationConstraint": "us-east-2"},
        )

        import importlib
        import app.main as main_module

        importlib.reload(main_module)

        event = {
            "rawPath": "/api/worlds/nonexistent",
            "requestContext": {"http": {"method": "GET"}},
        }
        result = main_module.lambda_handler(event, None)

        assert result["statusCode"] == 404
        body = json.loads(result["body"])
        assert "nonexistent" in body["error"]

    @mock_aws
    def test_get_map(self):
        """Test GET /api/worlds/{name}/{map} returns map details."""
        s3 = boto3.client("s3", region_name="us-east-2")
        s3.create_bucket(
            Bucket="test-maps-bucket",
            CreateBucketConfiguration={"LocationConstraint": "us-east-2"},
        )
        s3.put_object(
            Bucket="test-maps-bucket",
            Key="maps/survival/overworld/manifest.json",
            Body=json.dumps({
                "name": "overworld",
                "dimension": "minecraft:overworld",
                "center": [0, 0],
            }),
        )

        import importlib
        import app.main as main_module

        importlib.reload(main_module)

        event = {
            "rawPath": "/api/worlds/survival/overworld",
            "requestContext": {"http": {"method": "GET"}},
        }
        result = main_module.lambda_handler(event, None)

        assert result["statusCode"] == 200
        body = json.loads(result["body"])
        assert body["name"] == "overworld"
        assert body["dimension"] == "minecraft:overworld"
        assert body["previewUrl"] == "https://maps.example.com/maps/survival/overworld/preview.png"
        assert body["mapUrl"] == "https://maps.example.com/maps/survival/overworld/"

    @mock_aws
    def test_get_map_not_found(self):
        """Test GET /api/worlds/{name}/{map} returns 404 for missing map."""
        s3 = boto3.client("s3", region_name="us-east-2")
        s3.create_bucket(
            Bucket="test-maps-bucket",
            CreateBucketConfiguration={"LocationConstraint": "us-east-2"},
        )

        import importlib
        import app.main as main_module

        importlib.reload(main_module)

        event = {
            "rawPath": "/api/worlds/survival/nonexistent",
            "requestContext": {"http": {"method": "GET"}},
        }
        result = main_module.lambda_handler(event, None)

        assert result["statusCode"] == 404
        body = json.loads(result["body"])
        assert "nonexistent" in body["error"]

    @mock_aws
    def test_unknown_path(self):
        """Test unknown path returns 404."""
        s3 = boto3.client("s3", region_name="us-east-2")
        s3.create_bucket(
            Bucket="test-maps-bucket",
            CreateBucketConfiguration={"LocationConstraint": "us-east-2"},
        )

        import importlib
        import app.main as main_module

        importlib.reload(main_module)

        event = {
            "rawPath": "/api/unknown",
            "requestContext": {"http": {"method": "GET"}},
        }
        result = main_module.lambda_handler(event, None)

        assert result["statusCode"] == 404

    @mock_aws
    def test_legacy_event_format(self):
        """Test handler works with legacy API Gateway event format."""
        s3 = boto3.client("s3", region_name="us-east-2")
        s3.create_bucket(
            Bucket="test-maps-bucket",
            CreateBucketConfiguration={"LocationConstraint": "us-east-2"},
        )
        s3.put_object(
            Bucket="test-maps-bucket",
            Key="maps/world_manifest.json",
            Body=json.dumps([{"world": "test"}]),
        )

        import importlib
        import app.main as main_module

        importlib.reload(main_module)

        # Legacy format uses 'path' and 'httpMethod'
        event = {
            "path": "/api/worlds",
            "httpMethod": "GET",
        }
        result = main_module.lambda_handler(event, None)

        assert result["statusCode"] == 200


class TestCorsOrigin:
    """Tests for CORS origin configuration."""

    @mock_aws
    def test_custom_cors_origin(self):
        """Test custom CORS origin is used."""
        os.environ["CORS_ORIGIN"] = "https://minecraft.example.com"

        s3 = boto3.client("s3", region_name="us-east-2")
        s3.create_bucket(
            Bucket="test-maps-bucket",
            CreateBucketConfiguration={"LocationConstraint": "us-east-2"},
        )

        import importlib
        import app.main as main_module

        importlib.reload(main_module)

        result = main_module.make_response(200, {})

        assert result["headers"]["Access-Control-Allow-Origin"] == "https://minecraft.example.com"

        # Reset for other tests
        os.environ["CORS_ORIGIN"] = "*"


class TestBaseUrl:
    """Tests for BASE_URL configuration."""

    @mock_aws
    def test_default_base_url(self):
        """Test default BASE_URL uses S3 bucket URL."""
        os.environ.pop("BASE_URL", None)
        os.environ["MAPS_BUCKET"] = "my-bucket"
        os.environ["AWS_REGION"] = "us-west-2"

        s3 = boto3.client("s3", region_name="us-west-2")
        s3.create_bucket(
            Bucket="my-bucket",
            CreateBucketConfiguration={"LocationConstraint": "us-west-2"},
        )
        s3.put_object(
            Bucket="my-bucket",
            Key="maps/world_manifest.json",
            Body=json.dumps([{"world": "test"}]),
        )

        import importlib
        import app.main as main_module

        importlib.reload(main_module)

        event = {
            "rawPath": "/api/worlds",
            "requestContext": {"http": {"method": "GET"}},
        }
        result = main_module.lambda_handler(event, None)

        assert result["statusCode"] == 200
        body = json.loads(result["body"])
        assert "s3.us-west-2.amazonaws.com" in body[0]["previewUrl"]

        # Reset
        os.environ["BASE_URL"] = "https://maps.example.com"
        os.environ["MAPS_BUCKET"] = "test-maps-bucket"
        os.environ["AWS_REGION"] = "us-east-2"
