# Architecture

## Components

```
┌─────────────────────────────────────────────────────────────────┐
│  User                                                           │
│  kubectl apply BackupSchedule CR                                │
└────────────────────────────┬────────────────────────────────────┘
                             │ API Server
                             ▼
┌─────────────────────────────────────────────────────────────────┐
│  mimir-operator  (Go, controller-runtime)                       │
│                                                                 │
│  Watches: BackupSchedule + CronJob (Owns)                       │
│  Creates: CronJob in mimir-system namespace                     │
│  Sets:    OwnerReference → CronJob GC'd on CR delete            │
│  Updates: status.conditions[Ready]                              │
└────────────────────────────┬────────────────────────────────────┘
                             │ schedule triggers
                             ▼
┌─────────────────────────────────────────────────────────────────┐
│  CronJob  (in mimir-system)                                     │
│  Runs the Python agent pod on schedule                          │
└────────────────────────────┬────────────────────────────────────┘
                             │ spawns
                             ▼
┌─────────────────────────────────────────────────────────────────┐
│  Python agent pod  (agent/)                                     │
│                                                                 │
│  1. lists PVCs in target namespace                              │
│  2. for each PVC → creates a Kubernetes Job (rclone)            │
└────────────────────────────┬────────────────────────────────────┘
                             │ one Job per PVC
                             ▼
┌─────────────────────────────────────────────────────────────────┐
│  rclone Job  (in target namespace, mounts PVC read-only)        │
│                                                                 │
│  Destination: {namespace}/{pvc_name}/{YmdHis}                   │
│  Backends:    S3 │ IBM COS │ Azure Blob │ GCS                   │
└─────────────────────────────────────────────────────────────────┘
```

## Repository layout

```
mimir/
├── agent/          Python backup orchestrator
│   ├── src/mimir/
│   │   ├── backup.py       orchestration entry point
│   │   ├── k8s.py          PVC list + rclone Job creation
│   │   ├── scheduler.py    APScheduler blocking loop
│   │   ├── config.py       pydantic-settings (MIMIR_* env vars)
│   │   └── storage/        one file per backend (s3, azure, gcs)
│   └── Dockerfile
├── operator/       Go Kubernetes operator
│   ├── api/v1alpha1/
│   │   ├── backupschedule_types.go    CRD types + condition constants
│   │   ├── backupschedule_webhook.go  validating webhook (cron validation)
│   │   └── groupversion_info.go
│   ├── internal/controller/
│   │   └── backupschedule_controller.go
│   └── main.go
├── charts/mimir/   Helm chart
│   └── templates/  CRD, RBAC, Deployment, webhook (cert-manager)
├── config/
│   ├── crd/bases/  raw CRD YAML (used by envtest + Helm)
│   └── samples/    example BackupSchedule CRs (S3, Azure, GCS, IBM COS)
├── docs/           architecture documentation (this file)
└── hack/           helper scripts for code generation
```

## Key design decisions

### OwnerReference garbage collection
The operator uses `controllerutil.SetControllerReference` to set the CronJob's owner to the BackupSchedule CR. When the CR is deleted, Kubernetes automatically garbage-collects the CronJob — no explicit delete handler is needed.

### Webhook validation
The validating webhook calls `cron.ParseStandard` (robfig/cron) at admission time, rejecting any `BackupSchedule` with an invalid 5-field cron expression before it reaches the reconciler. Disable with `--enable-webhook=false` for local development.

### Storage backend abstraction
Each backend in `agent/src/mimir/storage/` implements two methods:
- `rclone_env() → dict` — rclone remote environment variables
- `remote_path() → str` — destination path prefix

rclone translates these to native SDK calls for each provider, so the agent does not link against any cloud SDK.

### RWO PVC limitation
`ReadWriteOnce` PVCs can only be mounted on one node at a time. If the workload pod holding the PVC is on a different node than the rclone Job, the Job stays in `Pending`. This is a Kubernetes storage constraint. Use `ReadWriteMany` storage classes (e.g. NFS, CephFS) to avoid this.

### Job naming
Job names are `mimir-{namespace}-{pvc}-{timestamp}`. If this exceeds the 63-character Kubernetes limit, a SHA-256 prefix is used instead (see `k8s.py:_job_name`).
