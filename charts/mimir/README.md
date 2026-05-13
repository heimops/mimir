# mimir

Kubernetes operator for declarative PVC backup scheduling. Define a `BackupSchedule` resource and mimir creates the CronJobs that copy your persistent volumes to S3, Azure Blob Storage, or GCS.

## Prerequisites

| Requirement | Version |
|-------------|---------|
| Kubernetes  | ≥ 1.26  |
| Helm        | ≥ 3.12  |
| cert-manager| ≥ 1.13 (only if `webhook.enabled: true`) |

## Installation

```bash
# Install cert-manager if not already present
kubectl apply -f https://github.com/cert-manager/cert-manager/releases/latest/download/cert-manager.yaml

# Install mimir
helm install mimir oci://ghcr.io/heimops/charts/mimir \
  --namespace mimir-system \
  --create-namespace \
  --version 0.1.1
```

Install without cert-manager (webhook disabled):

```bash
helm install mimir oci://ghcr.io/heimops/charts/mimir \
  --namespace mimir-system \
  --create-namespace \
  --set webhook.enabled=false
```

## Configuration

| Parameter | Default | Description |
|-----------|---------|-------------|
| `operator.image.repository` | `ghcr.io/heimops/mimir-operator` | Operator image |
| `operator.image.tag` | `latest` | Operator image tag |
| `operator.image.pullPolicy` | `IfNotPresent` | Image pull policy |
| `operator.replicas` | `1` | Number of operator replicas |
| `app.image.repository` | `ghcr.io/heimops/mimir` | Agent image used by backup CronJobs |
| `app.image.tag` | `latest` | Agent image tag |
| `serviceAccount.create` | `true` | Create a ServiceAccount for the operator |
| `serviceAccount.name` | `mimir` | ServiceAccount name |
| `rbac.create` | `true` | Create ClusterRole and ClusterRoleBinding |
| `webhook.enabled` | `true` | Enable the validating webhook (requires cert-manager) |

## Quick start

### 1. Create a storage secret

**AWS S3:**
```bash
kubectl create secret generic mimir-s3-creds -n mimir-system \
  --from-literal=MIMIR_S3__BUCKET=my-backup-bucket \
  --from-literal=MIMIR_S3__ACCESS_KEY_ID=AKIAIOSFODNN7EXAMPLE \
  --from-literal=MIMIR_S3__SECRET_ACCESS_KEY=wJalrXUtnFEMI/... \
  --from-literal=MIMIR_S3__REGION=eu-west-1
```

**Azure Blob Storage:**
```bash
kubectl create secret generic mimir-azure-creds -n mimir-system \
  --from-literal=MIMIR_AZURE__CONTAINER=k8s-backups \
  --from-literal=MIMIR_AZURE__ACCOUNT=mystorageaccount \
  --from-literal=MIMIR_AZURE__KEY=<storage-account-key>
```

**GCS (with Workload Identity, no JSON key needed):**
```bash
kubectl create secret generic mimir-gcs-creds -n mimir-system \
  --from-literal=MIMIR_GCS__BUCKET=my-gcs-bucket
```

**IBM Cloud Object Storage (S3-compatible):**
```bash
kubectl create secret generic mimir-cos-creds -n mimir-system \
  --from-literal=MIMIR_S3__BUCKET=my-cos-bucket \
  --from-literal=MIMIR_S3__ACCESS_KEY_ID=<hmac-access-key> \
  --from-literal=MIMIR_S3__SECRET_ACCESS_KEY=<hmac-secret-key> \
  --from-literal=MIMIR_S3__REGION=us-south \
  --from-literal=MIMIR_S3__ENDPOINT_URL=https://s3.us-south.cloud-object-storage.appdomain.cloud \
  --from-literal=MIMIR_S3__PROVIDER=IBMCOS
```

### 2. Create a BackupSchedule

```yaml
apiVersion: mimir.io/v1alpha1
kind: BackupSchedule
metadata:
  name: prod-nightly
  namespace: mimir-system
spec:
  namespace: production       # Kubernetes namespace whose PVCs will be backed up
  schedule: "0 2 * * *"       # Every day at 02:00 UTC (standard 5-field cron)
  storageBackend: s3          # s3 | azure | gcs
  storageSecret: mimir-s3-creds
  # pvcs: []                  # Omit to back up ALL PVCs; or list specific names:
  # pvcs:
  #   - database-data
  #   - uploads
```

```bash
kubectl apply -f backup-prod.yaml

# Check the operator created a CronJob
kubectl get backupschedule -n mimir-system
kubectl get cronjob -n mimir-system

# Trigger a manual backup run immediately
kubectl create job --from=cronjob/mimir-prod-nightly manual-test -n mimir-system
```

## BackupSchedule reference

| Field | Required | Description |
|-------|----------|-------------|
| `spec.namespace` | yes | Kubernetes namespace whose PVCs will be backed up |
| `spec.schedule` | yes | Standard 5-field cron expression, validated at admission time |
| `spec.storageBackend` | yes | `s3`, `azure`, or `gcs` |
| `spec.storageSecret` | yes | Name of the Secret in `mimir-system` containing backend credentials |
| `spec.pvcs` | no | List of PVC names to back up; omit or leave empty to back up all PVCs |
| `spec.image` | no | Override the default agent image for this schedule |

## Backup path format

Each backup is stored at:

```
{namespace}/{pvc_name}/{YmdHis}
```

Example: `production/postgres-data/20260101020000`

## How it works

```
BackupSchedule CR
      │
      ▼
  [Operator]  creates / updates
      │
      ▼
  [CronJob]  runs on schedule
      │
      ▼
  [Agent Pod]  lists PVCs → creates one rclone Job per PVC
      │
      ▼
  [rclone Job]  mounts PVC read-only → syncs to S3 / Azure / GCS
```

## Known limitations

- **ReadWriteOnce PVCs**: A PVC with `ReadWriteOnce` access mode can only be mounted on one node at a time. If the workload pod already holds the PVC on a different node, the backup Job will remain `Pending`. Use `ReadWriteMany` storage classes (NFS, CephFS) to avoid this.
- Backup Jobs are created in the same namespace as the target PVCs; the `mimir` ServiceAccount needs read access there (covered by the Helm chart ClusterRole).

## Source code

[https://github.com/heimops/mimir](https://github.com/heimops/mimir)
