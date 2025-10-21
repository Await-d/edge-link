# Changelog

All notable changes to the EdgeLink project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Initial project setup
- Backend microservices architecture (API Gateway, Device Service, Topology Service, NAT Coordinator, Alert Service, Background Worker)
- Frontend React 19 + Ant Design 5 application
- Desktop clients for Linux/Windows/macOS
- CI/CD pipelines with GitHub Actions
- Docker multi-stage builds with pinned base image digests
- SBOM generation for all Docker images
- Comprehensive deployment documentation
- Disaster recovery and rollback procedures

### Changed
- Frontend build system migrated from npm to pnpm for better dependency management
- Docker base images now pinned to specific SHA256 digests for reproducible builds

### Security
- Added multi-dimensional rate limiting (global/IP/org/user)
- Implemented CORS middleware with environment-specific configurations
- Added input validation and SQL injection protection
- Configured TLS certificate management with cert-manager
- Integrated Trivy and Gosec security scanners in CI/CD

### Fixed
- Database connection pool tuning for improved performance
- Redis caching strategy with multi-level TTL

## How to Use This Changelog

### For Developers
- Use [Conventional Commits](https://www.conventionalcommits.org/) format for all commit messages
- Changelog is automatically generated from commit messages on release

### Commit Message Format
```
<type>(<scope>): <subject>

<body>

<footer>
```

**Types**:
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation changes
- `style`: Code style changes (formatting, etc.)
- `refactor`: Code refactoring
- `perf`: Performance improvements
- `test`: Adding or updating tests
- `chore`: Maintenance tasks
- `ci`: CI/CD changes
- `build`: Build system changes

**Examples**:
```
feat(api-gateway): add rate limiting middleware

Implements multi-dimensional rate limiting with configurable thresholds.
Supports global, per-IP, per-organization, and per-user limits.

Closes #123
```

```
fix(database): resolve connection pool exhaustion

- Set MaxIdleConns to 50% of MaxOpenConns
- Added connection lifetime limits
- Implemented pool monitoring

BREAKING CHANGE: Database configuration format changed, requires migration
```

### Semantic Versioning

- **MAJOR** version (X.0.0): Incompatible API changes or breaking changes
- **MINOR** version (0.X.0): New features (backward compatible)
- **PATCH** version (0.0.X): Bug fixes (backward compatible)

### Creating a Release

1. Ensure all commits follow Conventional Commits format
2. Create and push a version tag:
   ```bash
   git tag v0.1.0
   git push origin v0.1.0
   ```
3. GitHub Actions will automatically:
   - Generate changelog from commits
   - Create GitHub Release
   - Update CHANGELOG.md
   - Build and push Docker images with version tags

---

**Note**: This changelog is automatically maintained. Manual edits may be overwritten.
For historical releases before automation, see [LEGACY_CHANGELOG.md](./LEGACY_CHANGELOG.md).
