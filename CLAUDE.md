# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Mimir backs up Kubernetes persistent volumes (PVCs) to cloud storage. It consists of:

| Component | Path | Purpose |
|-----------|------|---------|
| Python agent | `agent/` | Orchestrates rclone backup Jobs per PVC |
| Kubernetes operator | `operator/` | Watches BackupSchedule CRDs, manages CronJobs |
| Helm chart | `charts/mimir/` | Deploys operator + RBAC + CRD |

## How it works

1. User installs the chart → deploys the operator and CRD.
2. User creates a `BackupSchedule` CR → operator creates a CronJob that runs the mimir app.
3. The mimir app lists PVCs in the target namespace and creates one rclone Kubernetes **Job** per PVC.
4. Each rclone Job mounts the PVC (read-only) and syncs its contents to the destination path: `{namespace}/{pvc_name}/{YmdHis}`.

> **RWO PVCs**: a ReadWriteOnce volume can only be mounted by one node at a time. If an application pod already holds the volume on a different node, the backup Job will stay Pending. This is a Kubernetes storage constraint, not a mimir bug.

## Commands

```bash
# ── Python agent ─────────────────────────────────────────────────────────────
cd agent
pip install -e .            # or: pip install hatchling && hatch build
mimir backup                # one-shot backup (reads config from env vars)
mimir schedule              # blocking scheduler (requires MIMIR_SCHEDULE)

# ── Operator (Go) ───────────────────────────────────────────────────────────
cd operator
go mod tidy                 # fetch dependencies (generates go.sum)
go build ./...              # compile
go run . --leader-elect=false   # run locally against current kubeconfig

# ── Helm ────────────────────────────────────────────────────────────────────
helm install mimir oci://ghcr.io/heimops/charts/mimir --version 0.1.0 \
  -n mimir-system --create-namespace
# or from local source:
helm install mimir charts/mimir -n mimir-system --create-namespace
helm upgrade  mimir charts/mimir -n mimir-system

# ── Images (published on release) ───────────────────────────────────────────
# ghcr.io/heimops/mimir-operator:<version>
# ghcr.io/heimops/mimir:<version>
```

## Configuration (env vars for the app)

| Variable | Example | Description |
|----------|---------|-------------|
| `MIMIR_NAMESPACES` | `prod,staging` | Comma-separated target namespaces |
| `MIMIR_PVCS` | `db-data,uploads` | Specific PVCs; omit for all |
| `MIMIR_STORAGE_BACKEND` | `s3` | `s3` / `azure` / `gcs` |
| `MIMIR_SCHEDULE` | `0 2 * * *` | Cron expression (schedule mode only) |
| `MIMIR_BACKUP_IMAGE` | `rclone/rclone:latest` | Image used by backup Jobs |

**S3 / IBM COS**
```
MIMIR_S3__BUCKET, MIMIR_S3__ACCESS_KEY_ID, MIMIR_S3__SECRET_ACCESS_KEY
MIMIR_S3__REGION          (default: us-east-1)
MIMIR_S3__ENDPOINT_URL    (IBM COS or custom S3-compatible endpoint)
MIMIR_S3__PROVIDER        (default: AWS — use IBMCOS for IBM)
```

**Azure Blob**
```
MIMIR_AZURE__CONTAINER, MIMIR_AZURE__ACCOUNT, MIMIR_AZURE__KEY
```

**GCS**
```
MIMIR_GCS__BUCKET
MIMIR_GCS__SERVICE_ACCOUNT_JSON   (JSON string; omit to use Workload Identity / ADC)
```

## BackupSchedule CRD example

```yaml
apiVersion: mimir.io/v1alpha1
kind: BackupSchedule
metadata:
  name: prod-nightly
  namespace: mimir-system
spec:
  namespace: production          # namespace whose PVCs to back up
  schedule: "0 2 * * *"         # 02:00 UTC daily
  storageBackend: s3
  storageSecret: mimir-s3-creds  # Secret with MIMIR_S3__* keys in this namespace
  pvcs: []                       # empty = all PVCs; or list specific names
```

The Secret referenced by `storageSecret` must contain the `MIMIR_*` env vars for the chosen backend (e.g. `MIMIR_S3__BUCKET`, `MIMIR_S3__ACCESS_KEY_ID`, etc.).

## Architecture notes

- **Storage abstraction** (`agent/src/mimir/storage/`): each backend implements `rclone_env() → dict` (rclone remote env vars) and `remote_path() → str`. Adding a new backend = one new file + one entry in `storage/__init__.py`.
- **Job naming** (`k8s.py:_job_name`): if `mimir-{namespace}-{pvc}-{ts}` exceeds 63 chars, a SHA-256 prefix is used instead. Jobs live in the same namespace as their PVC.
- **Operator** (`operator/`): written in Go with `controller-runtime` v0.16. The reconciler uses `controllerutil.SetControllerReference` so Kubernetes garbage-collects the CronJob automatically when the BackupSchedule CR is deleted — no manual delete handler needed. `Owns(&batchv1.CronJob{})` in `SetupWithManager` ensures the reconciler re-runs if the CronJob is manually modified.
- **DeepCopy** (`api/v1alpha1/zz_generated_deepcopy.go`): written by hand to avoid a `controller-gen` dependency. If fields change, update both the types and the deepcopy file.
- **RBAC**: a single ClusterRole covers both operator (CRD watch + CronJob management) and app (PVC list + Job create) permissions, bound to the `mimir` ServiceAccount.
