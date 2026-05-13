# Changelog

All notable changes to this project will be documented in this file.
The format follows [Keep a Changelog](https://keepachangelog.com/en/1.0.0/) and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.1.1] - 2026-05-13

### Fixed
- `ValidatingWebhookConfiguration` template had a spurious `spec:` wrapper not valid in the Kubernetes API, causing `helm install` to fail with "field not declared in schema"
- Added explicit `privateKey.rotationPolicy: Always` to the cert-manager Certificate to silence the deprecation warning in cert-manager ≥ v1.18.0

## [0.1.0] - 2026-05-12

### Added
- `BackupSchedule` CRD (`mimir.io/v1alpha1`) for declarative backup scheduling
- Go operator (controller-runtime v0.16) that reconciles `BackupSchedule` CRs into Kubernetes CronJobs
- Validating webhook that rejects invalid cron expressions at admission time
- Python backup agent that creates one rclone `Job` per PVC
- Storage backends: AWS S3, IBM Cloud COS, Azure Blob Storage, GCS
- Backup path format: `{namespace}/{pvc_name}/{YmdHis}`
- Helm chart with cert-manager webhook integration, RBAC, and ServiceAccount
- `Ready` status condition on `BackupSchedule` following the Kubernetes condition standard
- GitHub Actions CI (test, lint, docker build, helm lint) and release pipeline
- Cosign keyless image signing and SPDX SBOM attestations on every release
- Artifact Hub metadata (`artifacthub-repo.yml`, `Chart.yaml` annotations)
