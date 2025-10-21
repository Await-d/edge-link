#!/bin/bash
# API Documentation Generation Script
# Generates API documentation from OpenAPI specifications

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
CONTRACTS_DIR="$PROJECT_ROOT/specs/001-edge-link-core/contracts"
DOCS_OUTPUT_DIR="$PROJECT_ROOT/docs/api"
OPENAPI_SPEC="$CONTRACTS_DIR/control-plane-api-v1.yaml"

# Tool versions
REDOCLY_VERSION="latest"
SWAGGER_CODEGEN_VERSION="3.0.42"

echo -e "${GREEN}EdgeLink API Documentation Generator${NC}"
echo "=========================================="
echo ""

# Check if OpenAPI spec exists
if [ ! -f "$OPENAPI_SPEC" ]; then
    echo -e "${RED}Error: OpenAPI specification not found at $OPENAPI_SPEC${NC}"
    exit 1
fi

echo -e "${GREEN}✓${NC} Found OpenAPI specification"

# Create output directory
mkdir -p "$DOCS_OUTPUT_DIR"

# Function to check if command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Function to generate docs with Redocly
generate_redocly_docs() {
    echo ""
    echo "Generating HTML documentation with Redocly..."

    if command_exists npx; then
        npx @redocly/cli build-docs "$OPENAPI_SPEC" \
            --output "$DOCS_OUTPUT_DIR/index.html" \
            --title "EdgeLink Control Plane API" \
            --theme.colors.primary.main="#1890ff"

        echo -e "${GREEN}✓${NC} Generated HTML documentation: $DOCS_OUTPUT_DIR/index.html"
    else
        echo -e "${YELLOW}⚠${NC} npx not found, skipping Redocly documentation"
        return 1
    fi
}

# Function to generate docs with Swagger UI
generate_swagger_ui() {
    echo ""
    echo "Generating Swagger UI documentation..."

    SWAGGER_UI_DIR="$DOCS_OUTPUT_DIR/swagger-ui"
    mkdir -p "$SWAGGER_UI_DIR"

    # Download Swagger UI dist
    if command_exists curl; then
        SWAGGER_UI_VERSION="5.10.3"
        SWAGGER_UI_URL="https://github.com/swagger-api/swagger-ui/archive/refs/tags/v${SWAGGER_UI_VERSION}.tar.gz"

        echo "Downloading Swagger UI v${SWAGGER_UI_VERSION}..."
        curl -L "$SWAGGER_UI_URL" -o /tmp/swagger-ui.tar.gz
        tar -xzf /tmp/swagger-ui.tar.gz -C /tmp

        # Copy dist files
        cp -r /tmp/swagger-ui-${SWAGGER_UI_VERSION}/dist/* "$SWAGGER_UI_DIR/"

        # Update swagger-initializer.js to point to our spec
        cat > "$SWAGGER_UI_DIR/swagger-initializer.js" <<EOF
window.onload = function() {
  window.ui = SwaggerUIBundle({
    url: "../control-plane-api-v1.yaml",
    dom_id: '#swagger-ui',
    deepLinking: true,
    presets: [
      SwaggerUIBundle.presets.apis,
      SwaggerUIStandalonePreset
    ],
    plugins: [
      SwaggerUIBundle.plugins.DownloadUrl
    ],
    layout: "StandaloneLayout"
  });
};
EOF

        # Copy OpenAPI spec to docs directory
        cp "$OPENAPI_SPEC" "$DOCS_OUTPUT_DIR/"

        echo -e "${GREEN}✓${NC} Generated Swagger UI: $SWAGGER_UI_DIR/index.html"

        # Cleanup
        rm -f /tmp/swagger-ui.tar.gz
        rm -rf /tmp/swagger-ui-${SWAGGER_UI_VERSION}
    else
        echo -e "${YELLOW}⚠${NC} curl not found, skipping Swagger UI generation"
        return 1
    fi
}

# Function to generate markdown documentation
generate_markdown_docs() {
    echo ""
    echo "Generating Markdown documentation..."

    if command_exists npx; then
        npx widdershins "$OPENAPI_SPEC" \
            --code \
            --language_tabs 'shell:cURL' 'go:Go' 'python:Python' 'javascript:JavaScript' \
            --summary \
            -o "$DOCS_OUTPUT_DIR/API.md"

        echo -e "${GREEN}✓${NC} Generated Markdown documentation: $DOCS_OUTPUT_DIR/API.md"
    else
        echo -e "${YELLOW}⚠${NC} npx not found, skipping Markdown documentation"
        return 1
    fi
}

# Function to validate OpenAPI spec
validate_spec() {
    echo ""
    echo "Validating OpenAPI specification..."

    if command_exists npx; then
        npx @redocly/cli lint "$OPENAPI_SPEC"
        echo -e "${GREEN}✓${NC} OpenAPI specification is valid"
    else
        echo -e "${YELLOW}⚠${NC} npx not found, skipping validation"
        return 1
    fi
}

# Function to generate client SDKs
generate_client_sdks() {
    echo ""
    echo "Generating client SDKs..."

    SDK_DIR="$PROJECT_ROOT/sdk"
    mkdir -p "$SDK_DIR"

    if command_exists docker; then
        # Generate Go SDK
        echo "Generating Go SDK..."
        docker run --rm -v "$PROJECT_ROOT:/local" \
            openapitools/openapi-generator-cli:v${SWAGGER_CODEGEN_VERSION} generate \
            -i /local/specs/001-edge-link-core/contracts/control-plane-api-v1.yaml \
            -g go \
            -o /local/sdk/go \
            --additional-properties=packageName=edgelinkclient,isGoSubmodule=true

        echo -e "${GREEN}✓${NC} Generated Go SDK: $SDK_DIR/go"

        # Generate Python SDK
        echo "Generating Python SDK..."
        docker run --rm -v "$PROJECT_ROOT:/local" \
            openapitools/openapi-generator-cli:v${SWAGGER_CODEGEN_VERSION} generate \
            -i /local/specs/001-edge-link-core/contracts/control-plane-api-v1.yaml \
            -g python \
            -o /local/sdk/python \
            --additional-properties=packageName=edgelink_client,projectName=edgelink-client

        echo -e "${GREEN}✓${NC} Generated Python SDK: $SDK_DIR/python"

        # Generate TypeScript SDK
        echo "Generating TypeScript SDK..."
        docker run --rm -v "$PROJECT_ROOT:/local" \
            openapitools/openapi-generator-cli:v${SWAGGER_CODEGEN_VERSION} generate \
            -i /local/specs/001-edge-link-core/contracts/control-plane-api-v1.yaml \
            -g typescript-axios \
            -o /local/sdk/typescript \
            --additional-properties=npmName=@edgelink/client,supportsES6=true

        echo -e "${GREEN}✓${NC} Generated TypeScript SDK: $SDK_DIR/typescript"
    else
        echo -e "${YELLOW}⚠${NC} Docker not found, skipping SDK generation"
        return 1
    fi
}

# Main execution
echo "Step 1: Validating OpenAPI specification..."
validate_spec || true

echo ""
echo "Step 2: Generating documentation formats..."
generate_redocly_docs || true
generate_swagger_ui || true
generate_markdown_docs || true

echo ""
echo "Step 3: Generating client SDKs..."
read -p "Generate client SDKs? This requires Docker and may take several minutes. (y/N): " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    generate_client_sdks || true
fi

# Generate index page
echo ""
echo "Generating documentation index..."
cat > "$DOCS_OUTPUT_DIR/README.md" <<EOF
# EdgeLink API Documentation

This directory contains auto-generated API documentation for the EdgeLink Control Plane API.

## Available Documentation Formats

- **[Interactive HTML Documentation](index.html)** - Redocly-powered interactive docs
- **[Swagger UI](swagger-ui/index.html)** - Try the API directly in your browser
- **[Markdown Documentation](API.md)** - Readable API reference in Markdown format
- **[OpenAPI Specification](control-plane-api-v1.yaml)** - Machine-readable API definition

## Client SDKs

Client SDKs can be generated using the \`generate-api-docs.sh\` script with the SDK generation option.

Supported languages:
- Go
- Python
- TypeScript/JavaScript
- Java
- Ruby
- C#
- PHP

## Usage

### Viewing Documentation Locally

1. **Redocly HTML**:
   \`\`\`bash
   # Using Python
   cd docs/api
   python3 -m http.server 8000
   # Visit http://localhost:8000/index.html
   \`\`\`

2. **Swagger UI**:
   \`\`\`bash
   cd docs/api/swagger-ui
   python3 -m http.server 8001
   # Visit http://localhost:8001
   \`\`\`

### Regenerating Documentation

\`\`\`bash
./scripts/generate-api-docs.sh
\`\`\`

## API Endpoints

The Control Plane API provides the following endpoint categories:

- **Device Management** (\`/api/v1/device/*\`) - Device registration, configuration, metrics
- **Network Management** (\`/api/v1/virtual-networks/*\`) - Virtual network CRUD operations
- **Admin Operations** (\`/api/v1/admin/*\`) - Device lifecycle, alert management
- **Monitoring** (\`/api/v1/metrics/*\`) - System metrics and health checks
- **WebSocket** (\`/ws\`) - Real-time event streaming

## Authentication

All API endpoints require authentication using one of:
- **Pre-shared Key (PSK)** - For device registration
- **Device Signature** - Ed25519 signature for registered devices
- **Bearer Token** - JWT token for admin users

See the full API documentation for authentication details.

## Support

For issues or questions:
- GitHub Issues: https://github.com/yourusername/edge-link/issues
- Documentation: https://docs.edgelink.io
- Email: support@edgelink.com

---

Generated: $(date)
EOF

echo -e "${GREEN}✓${NC} Generated documentation index: $DOCS_OUTPUT_DIR/README.md"

echo ""
echo "=========================================="
echo -e "${GREEN}Documentation generation complete!${NC}"
echo ""
echo "Documentation location: $DOCS_OUTPUT_DIR"
echo ""
echo "Next steps:"
echo "  1. View HTML docs: open $DOCS_OUTPUT_DIR/index.html"
echo "  2. View Swagger UI: open $DOCS_OUTPUT_DIR/swagger-ui/index.html"
echo "  3. Read Markdown: cat $DOCS_OUTPUT_DIR/API.md"
echo ""
