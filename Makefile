# ============================================================
# Project Build Makefile
# Builds:
#   - Lambda zips (control + details)
#   - Web UI (pnpm / vite / svelte)
# Usage:
#   make control        # builds dist/control.zip
#   make details        # builds dist/details.zip
#   make ui             # builds ui/dist/
#   make clean          # removes build artifacts
# ============================================================

# Lambda config
PYTHON_VERSION   := 3.13
PYTHON_PLATFORM  := x86_64-manylinux2014
DIST_DIR         := dist
PKG_DIR          := packages
UV_FLAGS         := --frozen --no-dev --no-editable
LAMBDAS          := control details worlds

# UI config
UI_DIR           := minecraft-ui/apps/manager
UI_DIST          := $(UI_DIR)/dist
WORLDS_DIR       := minecraft-ui/apps/worlds
WORLDS_DIST      := $(WORLDS_DIR)/build

# --- AWS config (override with `make VAR=value`) ---
AWS_REGION       ?= us-east-2
CONTROL_FUNC     ?= minecraft-test-control
DETAILS_FUNC     ?= minecraft-test-details
WORLD_FUNC       ?= minecraft-test-worlds
S3_BUCKET        ?= minecraft-test-webapp
CLOUDFRONT_DIST  ?= E35JG9QWEEVI98

# ============================================================
# Top-level targets
# ============================================================

all: lambdas ui
.PHONY: all

lambdas: $(LAMBDAS)
.PHONY: lambdas

ui:
	@echo "🖥️  Building web UI..."
	cd $(UI_DIR) && pnpm install --frozen-lockfile && pnpm run build
	@echo "✅ UI built at $(UI_DIST)"
.PHONY: ui

# ============================================================
# Generic pattern rule (builds dist/<lambda>.zip)
# Each Lambda's code lives in lambda/<lambda>/app/
# ============================================================

$(LAMBDAS): %:
	@echo "🐍 Building Lambda package: $@"
	rm -rf $(PKG_DIR)
	mkdir -p $(DIST_DIR)
	rm -f $(DIST_DIR)/$@.zip
	cd lambda/$@ && \
		uv export $(UV_FLAGS) -o requirements.txt && \
		uv pip install \
			--no-installer-metadata \
			--no-compile-bytecode \
			--python-platform $(PYTHON_PLATFORM) \
			--python $(PYTHON_VERSION) \
			--target ../../$(PKG_DIR) \
			-r requirements.txt

	cd $(PKG_DIR) && zip -qr ../$(DIST_DIR)/$@.zip .
	cd lambda/$@ && zip -qr ../../$(DIST_DIR)/$@.zip app

	@echo "✅ Built $(DIST_DIR)/$@.zip"
	@du -h $(DIST_DIR)/$@.zip | awk '{print "📦  Size:", $$1}'

# ============================================================
# Deploy targets
# ============================================================

deploy: deploy-lambdas deploy-ui
.PHONY: deploy

deploy-worlds: worlds
	aws lambda update-function-code \
		--function-name $(WORLD_FUNC) \
		--zip-file fileb://$(DIST_DIR)/worlds.zip \
		--region $(AWS_REGION)


deploy-lambdas:
	@echo "🚀 Deploying Lambda functions..."
	aws lambda update-function-code \
		--function-name $(CONTROL_FUNC) \
		--zip-file fileb://$(DIST_DIR)/control.zip \
		--region $(AWS_REGION)
	aws lambda update-function-code \
		--function-name $(DETAILS_FUNC) \
		--zip-file fileb://$(DIST_DIR)/details.zip \
		--region $(AWS_REGION)
	aws lambda update-function-code \
		--function-name $(WORLD_FUNC) \
		--zip-file fileb://$(DIST_DIR)/worlds.zip \
		--region $(AWS_REGION)
	@echo "✅ Lambda functions updated"

deploy-ui:
	@echo "🌐 Deploying web UI to S3..."
	aws s3 sync $(UI_DIST)/ s3://$(S3_BUCKET)/ --delete --region $(AWS_REGION)
ifdef CLOUDFRONT_DIST
	@echo "💨 Invalidating CloudFront cache..."
	aws cloudfront create-invalidation \
		--distribution-id $(CLOUDFRONT_DIST) \
		--paths '/*'
endif
	@echo "✅ UI deployed to https://$(S3_BUCKET).s3.$(AWS_REGION).amazonaws.com/"

deploy-ui-worlds:
	@echo "🌐 Deploying web UI to S3..."
	aws s3 sync $(WORLDS_DIST)/ s3://$(S3_BUCKET)/ --delete --region $(AWS_REGION)
ifdef CLOUDFRONT_DIST
	@echo "💨 Invalidating CloudFront cache..."
	aws cloudfront create-invalidation \
		--distribution-id $(CLOUDFRONT_DIST) \
		--paths '/*'
endif
	@echo "✅ UI deployed to https://$(S3_BUCKET).s3.$(AWS_REGION).amazonaws.com/"

.PHONY: deploy-lambdas deploy-ui deploy-ui-worlds

# ============================================================
# Maintenance
# ============================================================

clean:
	rm -rf $(PKG_DIR) $(DIST_DIR) $(UI_DIST)
	@echo "🧹 Cleaned all build artifacts"

.PHONY: $(LAMBDAS) clean
