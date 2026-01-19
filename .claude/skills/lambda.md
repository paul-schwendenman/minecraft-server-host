# Lambda Expert

Specialized guidance for working in `lambda/` - Python Lambda functions.

## Project Structure

```
lambda/
├── control/          # FastAPI + Mangum (EC2 start/stop/status, Route53)
├── details/          # Raw handler (Minecraft server ping via mcstatus)
└── worlds/           # Raw handler (Map manifest API, S3 integration)
```

Each lambda:
```
lambda/{name}/
├── app/
│   ├── __init__.py
│   ├── main.py       # Handler or FastAPI app
│   └── config.py     # Optional settings module
├── tests/
│   ├── conftest.py   # Pytest fixtures (moto mocks)
│   └── test_*.py
├── pyproject.toml    # uv-managed deps
├── uv.lock           # Frozen lock
└── requirements.txt  # Exported for Lambda packaging
```

## Conventions

### FastAPI Pattern (control)
```python
from fastapi import FastAPI
from mangum import Mangum

app = FastAPI()

@app.get("/status")
async def get_status():
    return {"status": "running"}

handler = Mangum(app, api_gateway_base_path="/api")
```

### Raw Handler Pattern (details, worlds)
```python
def lambda_handler(event, context):
    try:
        result = do_work()
        return {
            "statusCode": 200,
            "headers": {
                "Content-Type": "application/json",
                "Access-Control-Allow-Origin": "*"
            },
            "body": json.dumps(result)
        }
    except Exception as e:
        return {"statusCode": 500, "body": str(e)}
```

### Config Pattern
```python
from functools import lru_cache
from pydantic_settings import BaseSettings

class Settings(BaseSettings):
    instance_id: str
    cors_origin: str = "*"

@lru_cache
def get_settings():
    return Settings()
```

### Testing with Moto
```python
import pytest
from moto import mock_aws
import boto3

@pytest.fixture(autouse=True)
def aws_credentials(monkeypatch):
    monkeypatch.setenv("AWS_DEFAULT_REGION", "us-east-2")

@pytest.fixture
def ec2_client():
    with mock_aws():
        client = boto3.client("ec2", region_name="us-east-2")
        # Setup mock resources
        yield client

def test_start_server(ec2_client):
    # Import AFTER mock setup
    import importlib
    import app.main
    importlib.reload(app.main)
    # Test...
```

## Guidelines

1. **Reload after mock** - `importlib.reload(app.main)` after moto setup
2. **Clear LRU cache** - `get_settings.cache_clear()` in test teardown
3. **CORS headers** - Always include, use `CORS_ORIGIN` env var
4. **Module-level clients** - Initialize boto3 at module level for connection reuse
5. **Minimal deps** - Keep requirements small for cold start performance

## Python Versions

| Lambda | Python |
|--------|--------|
| control | 3.13 |
| details | 3.12 |
| worlds | 3.13 |

## Commands

```bash
cd lambda/{name}
uv sync --group dev          # Install with dev deps
uv run pytest tests/ -v      # Run tests

# Build (from repo root)
make control                 # -> dist/control.zip
make details                 # -> dist/details.zip
make worlds                  # -> dist/worlds.zip
```
