# Contributing to Mimir

## Prerequisites

| Tool | Version | Purpose |
|------|---------|---------|
| Go | ≥ 1.22 | Operator |
| Python | ≥ 3.11 | Backup app |
| Docker | any | Build images |
| kubectl | any | Manual testing |
| helm | ≥ 3.12 | Chart testing |
| golangci-lint | ≥ 1.57 | Go linting |

## Repository layout

```
agent/      Python backup orchestrator (hatch/pyproject.toml)
operator/   Go controller-runtime operator
charts/     Helm chart deploying both components
config/     CRD manifests and sample BackupSchedule resources
docs/       Architecture documentation
hack/       Helper scripts (CRD and deepcopy generation)
```

## Development workflow

### Operator (Go)

```bash
cd operator
go mod tidy
go build ./...
go test ./...

# Run against your current kubeconfig (no image build needed)
go run . --leader-elect=false
```

### Backup agent (Python)

```bash
cd agent
pip install -e ".[dev]"
ruff check .
pytest
```

### Running the full stack locally

A local [kind](https://kind.sigs.k8s.io/) cluster is enough:

```bash
kind create cluster

# Install CRD only (skip operator Deployment for local dev)
kubectl apply -f charts/mimir/templates/crd.yaml

# Run the operator out-of-cluster
cd operator && go run . --leader-elect=false
```

## Pull requests

- One logical change per PR.
- Add or update tests for any changed behaviour.
- Run `go vet ./...` and `golangci-lint run` before opening a PR; CI will block on lint failures.
- Update `CHANGELOG.md` under `[Unreleased]` with a short description of the change.

## Commits

Follow [Conventional Commits](https://www.conventionalcommits.org/):

```
feat(operator): add validation webhook for schedule field
fix(agent): handle RWO PVC mount failure gracefully
docs: update deployment steps in CLAUDE.md
```

## Cutting a release

Releases are driven by git tags. Pushing a `vX.Y.Z` tag triggers the release GitHub Action which:

1. Builds and pushes images to `ghcr.io`.
2. Signs images with cosign.
3. Packages and publishes the Helm chart.
4. Creates a GitHub Release with the changelog section.

Only maintainers can push release tags.

## Code of Conduct

This project follows the [CNCF Code of Conduct](https://github.com/cncf/foundation/blob/main/code-of-conduct.md).
