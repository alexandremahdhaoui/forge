# testenv-helm-install Testing Guide

This document describes how to set up and run tests for testenv-helm-install, including all required external dependencies and test infrastructure.

## Overview

testenv-helm-install supports multiple chart source types (Git, OCI, HTTP, S3) and requires various external services for comprehensive testing. This guide covers:

- Setting up test clusters with testenv-kind
- Configuring local Git servers (optional, for offline testing)
- Setting up local OCI registries (testenv-lcr or Docker registry:2)
- Configuring MinIO for S3 testing
- Running tests with appropriate environment variables

## Prerequisites

### Required Tools

```bash
# Core tools
- Go 1.21+
- Docker or Podman
- kind (Kubernetes in Docker)
- kubectl
- Helm 3.8+ (for OCI support)

# Optional for specific test scenarios
- Git (for Git source type testing)
- MinIO client (mc) for S3 verification
```

### Install Prerequisites

```bash
# Install kind
curl -Lo ./kind https://kind.sigs.k8s.io/dl/v0.20.0/kind-linux-amd64
chmod +x ./kind
sudo mv ./kind /usr/local/bin/kind

# Install kubectl
curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl"
chmod +x kubectl
sudo mv kubectl /usr/local/bin/kubectl

# Install Helm 3.8+
curl https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 | bash
```

## Test Infrastructure Setup

### 1. Test Cluster Setup (testenv-kind)

testenv-helm-install requires a Kubernetes cluster for testing. Use testenv-kind to create isolated test clusters:

#### Build testenv-kind

```bash
cd /path/to/forge
go build -o ./build/bin/testenv-kind ./cmd/testenv-kind
```

#### Create Test Cluster via forge

```bash
# Using forge test orchestration (recommended)
./build/bin/forge test integration create

# This creates:
# - Kind cluster with unique name: test-integration-{testID}
# - Kubeconfig in tmpDir: /tmp/forge-test-integration-{testID}/kubeconfig
# - Metadata stored in artifact store
```

#### Create Test Cluster Directly (for debugging)

```bash
# Direct testenv-kind invocation
./build/bin/testenv-kind --mcp

# Via MCP call:
# {"method": "tools/call", "params": {"name": "create", "arguments": {"testID": "debug-001", "stage": "integration", "tmpDir": "/tmp/test-debug-001"}}}
```

#### Verify Cluster

```bash
export KUBECONFIG=/tmp/forge-test-integration-{testID}/kubeconfig
kubectl cluster-info
kubectl get nodes
```

#### Cleanup Test Cluster

```bash
# Via forge
./build/bin/forge test integration delete {testID}

# Or directly
kind delete cluster --name test-integration-{testID}
```

### 2. Local OCI Registry Setup

testenv-helm-install supports OCI chart sources (`oci://registry/path`). You can use testenv-lcr or a standalone Docker registry.

#### Option A: Using testenv-lcr (Recommended)

testenv-lcr provides a fully integrated, TLS-enabled registry inside the Kind cluster:

```bash
# Build testenv-lcr
go build -o ./build/bin/testenv-lcr ./cmd/testenv-lcr

# testenv-lcr is automatically invoked when using forge test orchestration
# if configured in forge.yaml:
#
# test:
#   - name: integration
#     testenv: go://testenv
#     subengines:
#       - go://testenv-kind
#       - go://testenv-lcr  # Adds OCI registry support
```

testenv-lcr provides:
- TLS-enabled registry with self-signed certificates
- htpasswd authentication
- Automatic /etc/hosts configuration
- Registry FQDN: `testenv-lcr.testenv-lcr.svc.cluster.local:5000`
- CA certificate exported to tmpDir
- Credentials exported to tmpDir

See [testenv-architecture.md](./architecture/testenv-architecture.md) for more details.

#### Option B: Standalone Docker Registry

For external OCI registry testing:

```bash
# Run Docker registry:2
docker run -d \
  --name test-oci-registry \
  -p 5000:5000 \
  --restart=always \
  registry:2

# Push a test chart
helm package /path/to/chart
helm push my-chart-1.0.0.tgz oci://localhost:5000/charts

# Verify
helm pull oci://localhost:5000/charts/my-chart --version 1.0.0

# Cleanup
docker stop test-oci-registry
docker rm test-oci-registry
```

**Note**: For production-like testing with TLS and authentication, prefer testenv-lcr.

### 3. Local Git Server Setup (Optional)

For testing Git chart sources without external dependencies:

#### Option A: Git Daemon (Simple, Unauthenticated)

```bash
# Create test repository
mkdir -p /tmp/git-repos/helm-charts
cd /tmp/git-repos/helm-charts
git init --bare

# Add sample chart
cd /tmp
helm create test-chart
cd test-chart
git init
git add .
git commit -m "Initial commit"
git remote add origin file:///tmp/git-repos/helm-charts
git push origin main

# Start git daemon
git daemon --reuseaddr --base-path=/tmp/git-repos --export-all --verbose --enable=receive-pack &

# Test access
git clone git://localhost/helm-charts /tmp/test-clone

# Cleanup
pkill git-daemon
```

#### Option B: Use GitHub/GitLab Directly

For integration tests, you can use public Git repositories:

```yaml
# Example forge.yaml test configuration
test:
  - name: integration
    runner: go://go-test
    testenv: go://testenv
    env:
      GIT_CHART_REPO: "https://github.com/example/charts.git"
      GIT_CHART_PATH: "charts/my-chart"
```

**Recommendation**: Use public repositories for CI/CD. Use local git daemon only for offline development/testing.

### 4. MinIO Setup for S3 Testing

testenv-helm-install supports S3 chart sources. Use MinIO for local S3-compatible testing:

#### Start MinIO Server

```bash
# Run MinIO in Docker
docker run -d \
  --name minio-test \
  -p 9000:9000 \
  -p 9001:9001 \
  -e "MINIO_ROOT_USER=minioadmin" \
  -e "MINIO_ROOT_PASSWORD=minioadmin" \
  quay.io/minio/minio server /data --console-address ":9001"

# Verify MinIO is running
curl http://localhost:9000/minio/health/live
```

#### Configure MinIO Client

```bash
# Install mc (MinIO client)
curl https://dl.min.io/client/mc/release/linux-amd64/mc -o mc
chmod +x mc
sudo mv mc /usr/local/bin/

# Configure alias
mc alias set local http://localhost:9000 minioadmin minioadmin

# Create test bucket
mc mb local/helm-charts

# Upload test chart
helm package /path/to/chart
mc cp my-chart-1.0.0.tgz local/helm-charts/

# Verify
mc ls local/helm-charts/
```

#### S3 Environment Variables for Tests

```bash
export S3_ENDPOINT="http://localhost:9000"
export S3_ACCESS_KEY="minioadmin"
export S3_SECRET_KEY="minioadmin"
export S3_BUCKET="helm-charts"
export S3_REGION="us-east-1"  # MinIO default
```

#### Cleanup MinIO

```bash
docker stop minio-test
docker rm minio-test
```

### 5. HTTP Server Setup (Optional)

For testing HTTP chart sources:

```bash
# Simple Python HTTP server
mkdir -p /tmp/http-charts
cd /tmp/http-charts

# Package chart
helm package /path/to/chart
helm repo index .

# Start HTTP server
python3 -m http.server 8080 &

# Test access
curl http://localhost:8080/index.yaml
curl http://localhost:8080/my-chart-1.0.0.tgz

# Cleanup
pkill -f "http.server 8080"
```

## Running Tests

### Environment Variables

#### Skip Test Categories

Use these environment variables to skip test categories during development:

```bash
# Skip integration tests (tests requiring external services)
export SKIP_INTEGRATION_TESTS=true

# Skip end-to-end tests (tests requiring full test environment)
export SKIP_E2E_TESTS=true

# Run only unit tests
export SKIP_INTEGRATION_TESTS=true SKIP_E2E_TESTS=true
go test ./cmd/testenv-helm-install/...
```

#### Test Configuration

```bash
# OCI registry configuration
export OCI_REGISTRY="localhost:5000"
export OCI_REGISTRY_USERNAME="admin"
export OCI_REGISTRY_PASSWORD="password"

# Git configuration
export GIT_CHART_REPO="git://localhost/helm-charts"
export GIT_CHART_BRANCH="main"
export GIT_CHART_PATH="charts/my-chart"

# S3 configuration
export S3_ENDPOINT="http://localhost:9000"
export S3_ACCESS_KEY="minioadmin"
export S3_SECRET_KEY="minioadmin"
export S3_BUCKET="helm-charts"
export S3_REGION="us-east-1"

# HTTP configuration
export HTTP_CHART_URL="http://localhost:8080"
```

### Test Execution

#### Unit Tests Only

```bash
# Run unit tests (no external dependencies)
export SKIP_INTEGRATION_TESTS=true SKIP_E2E_TESTS=true
./build/bin/forge test unit run

# Or directly with go test
go test -tags=unit ./cmd/testenv-helm-install/...
```

#### Integration Tests

```bash
# Start required services first
docker run -d --name minio-test -p 9000:9000 -p 9001:9001 \
  -e "MINIO_ROOT_USER=minioadmin" -e "MINIO_ROOT_PASSWORD=minioadmin" \
  quay.io/minio/minio server /data --console-address ":9001"

docker run -d --name test-oci-registry -p 5000:5000 registry:2

# Set environment variables
export S3_ENDPOINT="http://localhost:9000"
export S3_ACCESS_KEY="minioadmin"
export S3_SECRET_KEY="minioadmin"
export OCI_REGISTRY="localhost:5000"

# Run integration tests
./build/bin/forge test integration run

# Or with go test
go test -tags=integration ./cmd/testenv-helm-install/...

# Cleanup
docker stop minio-test test-oci-registry
docker rm minio-test test-oci-registry
```

#### End-to-End Tests

```bash
# E2E tests require full test environment (Kind cluster + all services)
./build/bin/forge test e2e run

# This will:
# 1. Create Kind cluster via testenv-kind
# 2. Setup OCI registry via testenv-lcr (if configured)
# 3. Run all E2E tests
# 4. Cleanup test environment
```

#### Run All Tests

```bash
# Build all + run all test stages (fail-fast, auto-cleanup)
./build/bin/forge test-all

# This runs:
# 1. forge build (build all artifacts)
# 2. forge test verify-tags run (verify build tags)
# 3. forge test unit run (unit tests)
# 4. forge test lint run (linters)
# 5. forge test integration run (integration tests)
# 6. forge test e2e run (end-to-end tests)
```

### Test Workflow Best Practices

#### Development Workflow

```bash
# 1. Start with unit tests (fast feedback)
export SKIP_INTEGRATION_TESTS=true SKIP_E2E_TESTS=true
go test -v ./cmd/testenv-helm-install/...

# 2. Run integration tests when changing external integrations
unset SKIP_INTEGRATION_TESTS
# Start required services (MinIO, OCI registry, etc.)
go test -tags=integration -v ./cmd/testenv-helm-install/...

# 3. Run E2E tests before committing
./build/bin/forge test e2e run

# 4. Final verification
./build/bin/forge test-all
```

#### CI/CD Workflow

```yaml
# Example GitHub Actions workflow
- name: Run unit tests
  run: |
    export SKIP_INTEGRATION_TESTS=true SKIP_E2E_TESTS=true
    ./build/bin/forge test unit run

- name: Run integration tests
  run: |
    # Start MinIO and OCI registry
    docker run -d --name minio -p 9000:9000 -p 9001:9001 \
      -e MINIO_ROOT_USER=minioadmin -e MINIO_ROOT_PASSWORD=minioadmin \
      quay.io/minio/minio server /data --console-address ":9001"
    docker run -d --name registry -p 5000:5000 registry:2

    # Run tests
    export S3_ENDPOINT=http://localhost:9000
    export S3_ACCESS_KEY=minioadmin
    export S3_SECRET_KEY=minioadmin
    export OCI_REGISTRY=localhost:5000
    ./build/bin/forge test integration run

- name: Run E2E tests
  run: ./build/bin/forge test e2e run

- name: Cleanup
  if: always()
  run: |
    docker stop minio registry || true
    docker rm minio registry || true
```

## Troubleshooting

### Common Issues

#### Kind Cluster Creation Fails

```bash
# Check Docker is running
docker ps

# Check kind version
kind version

# View kind logs
kind get clusters
docker logs test-integration-{testID}-control-plane
```

#### OCI Registry Push Fails

```bash
# Check registry is accessible
curl -v http://localhost:5000/v2/

# For testenv-lcr, check cert-manager
export KUBECONFIG=/tmp/forge-test-integration-{testID}/kubeconfig
kubectl get pods -n cert-manager
kubectl get certificate -n testenv-lcr

# Check registry logs
kubectl logs -n testenv-lcr deployment/testenv-lcr
```

#### MinIO Connection Fails

```bash
# Check MinIO is running
docker ps | grep minio

# Check MinIO health
curl http://localhost:9000/minio/health/live

# View MinIO logs
docker logs minio-test

# Test with mc client
mc alias set local http://localhost:9000 minioadmin minioadmin
mc ls local/
```

#### Helm Install Fails

```bash
# Check Helm version (must be 3.8+ for OCI)
helm version

# Check kubeconfig
export KUBECONFIG=/tmp/forge-test-integration-{testID}/kubeconfig
kubectl config current-context

# Dry-run the install
helm install --dry-run --debug my-release chart/

# Check Helm logs
helm install my-release chart/ --debug --wait
```

### Debug Mode

Enable verbose logging for troubleshooting:

```bash
# Enable debug logging
export LOG_LEVEL=debug

# Run tests with verbose output
go test -v -tags=integration ./cmd/testenv-helm-install/...

# Enable Helm debug output
export HELM_DEBUG=1
```

## Quick Reference

### Complete Setup Script

```bash
#!/bin/bash
# setup-testenv-helm-install.sh

# Start MinIO
docker run -d --name minio-test -p 9000:9000 -p 9001:9001 \
  -e "MINIO_ROOT_USER=minioadmin" -e "MINIO_ROOT_PASSWORD=minioadmin" \
  quay.io/minio/minio server /data --console-address ":9001"

# Start OCI registry
docker run -d --name test-oci-registry -p 5000:5000 registry:2

# Configure MinIO
mc alias set local http://localhost:9000 minioadmin minioadmin
mc mb local/helm-charts

# Set environment variables
export S3_ENDPOINT="http://localhost:9000"
export S3_ACCESS_KEY="minioadmin"
export S3_SECRET_KEY="minioadmin"
export S3_BUCKET="helm-charts"
export S3_REGION="us-east-1"
export OCI_REGISTRY="localhost:5000"

echo "Test environment ready!"
echo "Run tests with: ./build/bin/forge test-all"
```

### Complete Cleanup Script

```bash
#!/bin/bash
# cleanup-testenv-helm-install.sh

# Stop and remove containers
docker stop minio-test test-oci-registry 2>/dev/null
docker rm minio-test test-oci-registry 2>/dev/null

# Delete kind clusters
kind get clusters | grep "^test-integration-" | xargs -r kind delete cluster --name

# Clean temporary directories
rm -rf /tmp/forge-test-integration-*
rm -rf /tmp/git-repos
rm -rf /tmp/http-charts

echo "Cleanup complete!"
```

## See Also

- [testenv-architecture.md](./architecture/testenv-architecture.md) - Test environment architecture
- [testing.md](./user/testing.md) - Forge test command reference
- [MCP.md](../cmd/testenv-helm-install/MCP.md) - testenv-helm-install MCP documentation
