# ============================================================
# Lambda packaging Makefile
# Usage:
#   make control        # builds dist/control.zip
#   make details        # builds dist/details.zip
#   make clean          # removes build artifacts
# ============================================================

# Python & platform targets
PYTHON_VERSION   := 3.13
PYTHON_PLATFORM  := x86_64-manylinux2014     # change to aarch64-manylinux2014 for ARM Lambdas

# Directories
DIST_DIR         := dist
PKG_DIR          := packages

# Common build flags for uv
UV_FLAGS         := --frozen --no-dev --no-editable

# Lambda package names (subdirectories under lambda/)
LAMBDAS          := control details

# ============================================================
# Generic pattern rule (builds each lambda/<name>/package.zip)
# ============================================================

all: $(LAMBDAS)

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
	rm -rf $(PKG_DIR) $(DIST_DIR)
	@echo "🧹 Cleaned build artifacts"

.PHONY: $(LAMBDAS) clean all
