# mimir

[![CI](https://github.com/heimops/mimir/actions/workflows/ci.yml/badge.svg)](https://github.com/heimops/mimir/actions/workflows/ci.yml)
[![Artifact Hub](https://img.shields.io/endpoint?url=https://artifacthub.io/badge/repository/mimir)](https://artifacthub.io/packages/helm/heimops/mimir)
[![License](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](LICENSE)

Kubernetes operator for declarative PVC backup scheduling. Define a `BackupSchedule` resource and mimir creates the CronJobs that copy your persistent volumes to S3, Azure Blob Storage, or GCS.

## Features

- **Declarative API** — `BackupSchedule` CRD manages CronJobs via OwnerReferences (automatic GC on delete)
- **Multi-backend** — AWS S3, IBM Cloud COS (S3-compatible), Azure Blob Storage, Google Cloud Storage
- **Admission validation** — validating webhook rejects invalid cron expressions at apply time
- **Standard status** — `Ready` condition on every resource following the Kubernetes condition standard
- **Supply chain security** — images signed with cosign (keyless) + signed SPDX SBOM attestations on every release

## Architecture

```
kubectl apply BackupSchedule
        │
        ▼
  [Go Operator]  ←── watches BackupSchedule + CronJob
        │
        │  creates/updates
        ▼
  [CronJob]  (in mimir-system)
        │
        │  triggers on schedule
        ▼
  [Python Agent Pod]
        │
        │  lists PVCs → creates one Job per PVC
        ▼
  [rclone Job]  (in target namespace)
        │
        └──► S3 / Azure / GCS
```

## Requirements

| Component    | Version   |
|-------------|-----------|
| Kubernetes  | ≥ 1.26    |
| Helm        | ≥ 3.12    |
| cert-manager| ≥ 1.13    |

## Installation

```bash
# 1. cert-manager (skip if already installed)
kubectl apply -f https://github.com/cert-manager/cert-manager/releases/latest/download/cert-manager.yaml

# 2. Install mimir
helm install mimir oci://ghcr.io/heimops/charts/mimir \
  --namespace mimir-system \
  --create-namespace \
  --version 0.1.0
```

Without cert-manager (webhook disabled):
```bash
helm install mimir oci://ghcr.io/heimops/charts/mimir \
  --namespace mimir-system \
  --create-namespace \
  --set webhook.enabled=false
```

## Quick start

### 1. Create the storage secret

**AWS S3:**
```bash
kubectl create secret generic mimir-s3-creds -n mimir-system \
  --from-literal=MIMIR_S3__BUCKET=my-backup-bucket \
  --from-literal=MIMIR_S3__ACCESS_KEY_ID=AKIAIOSFODNN7EXAMPLE \
  --from-literal=MIMIR_S3__SECRET_ACCESS_KEY=wJalrXUtnFEMI/... \
  --from-literal=MIMIR_S3__REGION=eu-west-1
```

See [config/samples/](config/samples/) for Azure, GCS, and IBM COS examples.

### 2. Create a BackupSchedule

```yaml
apiVersion: mimir.io/v1alpha1
kind: BackupSchedule
metadata:
  name: prod-nightly
  namespace: mimir-system
spec:
  namespace: production       # namespace whose PVCs will be backed up
  schedule: "0 2 * * *"       # every day at 02:00 UTC
  storageBackend: s3
  storageSecret: mimir-s3-creds
  # pvcs: []                  # omit to back up ALL PVCs in the namespace
```

```bash
kubectl apply -f backup-prod.yaml
```

### 3. Monitor

```bash
# Check the operator created a CronJob
kubectl get backupschedule -n mimir-system
kubectl get cronjob -n mimir-system

# Trigger a manual backup run
kubectl create job --from=cronjob/mimir-prod-nightly manual-test -n mimir-system

# Watch rclone jobs in the target namespace
kubectl get jobs -n production -w
```

## BackupSchedule reference

| Field            | Required | Description                                                   |
|-----------------|----------|---------------------------------------------------------------|
| `namespace`      | yes      | Kubernetes namespace whose PVCs will be backed up             |
| `schedule`       | yes      | Standard 5-field cron expression (validated at admission)     |
| `storageBackend` | yes      | `s3`, `azure`, or `gcs`                                       |
| `storageSecret`  | yes      | Name of the Secret in `mimir-system` with credentials         |
| `pvcs`           | no       | List of PVC names to back up; omit to back up all PVCs        |
| `image`          | no       | Override the default agent image                              |

### Secret keys by backend

**S3 / IBM COS:**
`MIMIR_S3__BUCKET`, `MIMIR_S3__ACCESS_KEY_ID`, `MIMIR_S3__SECRET_ACCESS_KEY`, `MIMIR_S3__REGION`
Optional: `MIMIR_S3__ENDPOINT_URL` (for S3-compatible APIs), `MIMIR_S3__PROVIDER` (e.g. `IBMCOS`)

**Azure Blob Storage:**
`MIMIR_AZURE__CONTAINER`, `MIMIR_AZURE__ACCOUNT`, `MIMIR_AZURE__KEY`

**GCS:**
`MIMIR_GCS__BUCKET`
Optional: `MIMIR_GCS__SERVICE_ACCOUNT_JSON` (omit when using Workload Identity on GKE)

## Known limitations

- PVCs with `ReadWriteOnce` access mode that are already mounted on a node may cause backup jobs to stay in `Pending` on a different node. Use `ReadWriteMany` or ensure the backup job lands on the same node.
- Backup jobs run in the same namespace as the target PVCs; the `mimir` ServiceAccount needs read access there (handled by the Helm chart ClusterRole).

## Backup path format

```
{namespace}/{pvc_name}/{YmdHis}
```

Example: `production/postgres-data/20260101020000`

## Supply chain verification

See [VERIFICATION.md](VERIFICATION.md) for commands to verify image signatures and SBOM attestations with cosign.

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md). Please read [SECURITY.md](SECURITY.md) before reporting vulnerabilities.

## License

Apache 2.0 — see [LICENSE](LICENSE).
