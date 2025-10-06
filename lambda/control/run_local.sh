#!/usr/bin/env bash
set -euo pipefail

# --- Config ---
APP="lambda.control.app:app"
REGION="us-east-2"
DNS_NAME="minecraft.example.com"
ZONE_NAME="example.com"
AWS_LOCAL=1
PORT=8000

export AWS_ACCESS_KEY_ID=test
export AWS_SECRET_ACCESS_KEY=test
export AWS_DEFAULT_REGION=${REGION:-us-east-2}


# --- Start LocalStack ---
echo "▶ Starting LocalStack..."
docker run -d --rm --name localstack \
  -p 4566:4566 \
  -e SERVICES=ec2,route53,s3 \
  -e DEBUG=1 \
  localstack/localstack:latest >/dev/null

# --- Wait for it to be ready ---
echo "⏳ Waiting for LocalStack to be ready..."
attempts=0
while true; do
  # Check for any kind of valid JSON health response
  if curl -sf http://localhost:4566/_localstack/health | grep -q '"services"'; then
    echo "✅ LocalStack (new API) is ready."
    break
  # elif curl -sf http://localhost:4566 | grep -q 'running'; then
  #   echo "✅ LocalStack (edge router) is responding."
  #   break
  fi

  attempts=$((attempts+1))
  if [ "$attempts" -gt 20 ]; then
    echo "⚠️  LocalStack did not return a health payload after 40s; continuing anyway."
    break
  fi
  sleep 2
done

# --- Create fake resources ---
echo "🌐 Creating fake AWS resources..."

# create a dummy instance and grab its ID
INSTANCE_ID=$(
  aws --endpoint-url=http://localhost:4566 ec2 run-instances \
    --image-id ami-12345678 --count 1 --region "$REGION" \
    --query 'Instances[0].InstanceId' --output text
)
echo "🖥️  Created mock EC2 instance: $INSTANCE_ID"

# create a fake hosted zone if none exists yet
ZONE_ID=$(
  aws --endpoint-url=http://localhost:4566 route53 create-hosted-zone \
    --name "example.com" --caller-reference "test-$(date +%s)" \
    --query 'HostedZone.Id' --output text
)
echo "🌎 Created mock Route53 zone: $ZONE_ID"


# --- Export environment vars for the app ---
export INSTANCE_ID="$INSTANCE_ID"
export REGION="$REGION"
export DNS_NAME="$DNS_NAME"
export CORS_ORIGIN="*"
export AWS_LOCAL="$AWS_LOCAL"

# --- Run FastAPI app ---
echo "🚀 Starting FastAPI app on http://localhost:$PORT ..."

trap 'echo "🧹 Stopping LocalStack..."; docker stop localstack >/dev/null || true' EXIT

# uv run uvicorn "$APP" --reload --port "$PORT"
uv run fastapi dev --port "$PORT"
