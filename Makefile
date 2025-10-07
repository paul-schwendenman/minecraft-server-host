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
LAMBDAS          := control details

# UI config
UI_DIR           := ui
UI_DIST          := $(UI_DIR)/dist

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
# Maintenance
# ============================================================

clean:
	rm -rf $(PKG_DIR) $(DIST_DIR) $(UI_DIST)
	@echo "🧹 Cleaned all build artifacts"

.PHONY: $(LAMBDAS) clean
