# ============================================
# Stage 1: Build Stage
# Purpose: Build React frontend application
# ============================================
ARG NODE_VERSION=20
ARG NGINX_VERSION=1.25
ARG NODE_ALPINE_DIGEST=sha256:1ab6fc5a31d515dc7b6b25f6acfda2001821f2c2400252b6cb61044bd9f9ad48
ARG NGINX_ALPINE_DIGEST=sha256:516475cc129da42866742567714ddc681e5eed7b9ee0b9e9c015e464b4221a00

FROM node:${NODE_VERSION}-alpine@${NODE_ALPINE_DIGEST} AS builder

# Declare build-time metadata
ARG BUILD_DATE
ARG VERSION
ARG COMMIT_SHA

WORKDIR /build

# Install pnpm package manager
# - corepack: Node.js package manager manager
# - pnpm: Fast, disk space efficient package manager
RUN corepack enable && corepack prepare pnpm@9 --activate

# Layer 1: Install dependencies (cached unless package files change)
COPY package.json pnpm-lock.yaml ./
RUN pnpm install --frozen-lockfile --production=false

# Layer 2: Copy source code (invalidates cache on code changes)
COPY . .

# Layer 3: Build production bundle
# Optimizations: minification, tree-shaking, code splitting
RUN pnpm run build && \
    # Clean up build artifacts to reduce layer size
    rm -rf node_modules .cache

# ============================================
# Stage 2: Final Runtime Stage
# Purpose: Serve static files with nginx
# ============================================
FROM nginx:${NGINX_VERSION}-alpine@${NGINX_ALPINE_DIGEST}

# Declare build-time metadata (must re-declare in multi-stage build)
ARG BUILD_DATE
ARG VERSION
ARG COMMIT_SHA

# Copy custom nginx configuration
COPY nginx.conf /etc/nginx/conf.d/default.conf

# Copy built assets from builder stage
COPY --from=builder /build/dist /usr/share/nginx/html

# Add OCI metadata labels
LABEL org.opencontainers.image.created="${BUILD_DATE}" \
      org.opencontainers.image.version="${VERSION}" \
      org.opencontainers.image.revision="${COMMIT_SHA}" \
      org.opencontainers.image.title="EdgeLink Frontend" \
      org.opencontainers.image.description="EdgeLink Management UI - React SPA" \
      org.opencontainers.image.vendor="EdgeLink" \
      org.opencontainers.image.authors="EdgeLink Development Team"

# Expose HTTP port
EXPOSE 80

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost/ || exit 1

# Start nginx (runs as non-root by default in nginx:alpine)
CMD ["nginx", "-g", "daemon off;"]
