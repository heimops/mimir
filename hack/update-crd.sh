#!/usr/bin/env bash
# Regenerate the CRD YAML from Go types using controller-gen.
# Run this whenever backupschedule_types.go changes.
#
# Prerequisites:
#   go install sigs.k8s.io/controller-tools/cmd/controller-gen@latest
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

controller-gen \
  crd:crdVersions=v1 \
  paths="${REPO_ROOT}/operator/api/..." \
  output:crd:dir="${REPO_ROOT}/config/crd/bases"

echo "CRD written to config/crd/bases/"
