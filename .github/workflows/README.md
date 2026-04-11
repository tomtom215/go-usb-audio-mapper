<!-- SPDX-License-Identifier: MIT -->

# CI/CD Workflows

| Workflow | Trigger | Purpose |
|----------|---------|---------|
| [`ci.yml`](ci.yml) | Push/PR to main | Lint (gofmt, go vet, golangci-lint), test (Go 1.22-1.24 matrix), multi-arch build |
| [`coverage.yml`](coverage.yml) | Push/PR to main | Code coverage with Codecov upload |
| [`security.yml`](security.yml) | Push/PR to main + weekly | `govulncheck` vulnerability scanning, license audit |

## CI Matrix

- **Go versions:** 1.23, 1.24, 1.25
- **Build targets:** linux/amd64, linux/arm64, linux/arm
- **Linters:** gofmt, go vet, golangci-lint (staticcheck, gosec, gocritic, etc.)

## Release Automation

Tag pushes matching `v*` trigger GoReleaser to build multi-arch binaries and create a GitHub Release with checksums.
