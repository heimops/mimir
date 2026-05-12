#!/usr/bin/env bash
# Regenerate zz_generated_deepcopy.go from Go types using controller-gen.
# Run this whenever struct fields in api/v1alpha1/ change.
#
# Prerequisites:
#   go install sigs.k8s.io/controller-tools/cmd/controller-gen@latest
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

controller-gen \
  object:headerFile="${REPO_ROOT}/hack/boilerplate.go.txt" \
  paths="${REPO_ROOT}/operator/api/..."

echo "DeepCopy written to operator/api/v1alpha1/zz_generated_deepcopy.go"
