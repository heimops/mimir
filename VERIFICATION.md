# Verification

All release images are signed with [cosign](https://docs.sigstore.dev/cosign/overview/) using
**keyless signing** via GitHub Actions OIDC. No long-lived private key is stored anywhere.
Signatures and SBOM attestations are recorded in the public [Sigstore Rekor](https://rekor.sigstore.dev) transparency log.

## Prerequisites

```bash
# Install cosign (https://docs.sigstore.dev/cosign/installation/)
brew install cosign          # macOS
# or
go install github.com/sigstore/cosign/v2/cmd/cosign@latest
```

## Verify an image signature

Replace `<version>` with the tag you want to verify (e.g. `0.1.0`).

```bash
# Operator
cosign verify \
  --certificate-identity-regexp \
    "https://github.com/heimops/mimir/.github/workflows/release.yml@refs/tags/.*" \
  --certificate-oidc-issuer "https://token.actions.githubusercontent.com" \
  ghcr.io/heimops/mimir-operator:<version>

# App
cosign verify \
  --certificate-identity-regexp \
    "https://github.com/heimops/mimir/.github/workflows/release.yml@refs/tags/.*" \
  --certificate-oidc-issuer "https://token.actions.githubusercontent.com" \
  ghcr.io/heimops/mimir:<version>
```

A successful verification prints the certificate details and exits 0.

## Verify the SBOM attestation

Each image has a signed [SPDX](https://spdx.dev/) SBOM attestation stored in the OCI registry.

```bash
# Verify and extract the SBOM for the operator image
cosign verify-attestation \
  --type spdx \
  --certificate-identity-regexp \
    "https://github.com/heimops/mimir/.github/workflows/release.yml@refs/tags/.*" \
  --certificate-oidc-issuer "https://token.actions.githubusercontent.com" \
  ghcr.io/heimops/mimir-operator:<version> \
  | jq -r '.payload' | base64 -d | jq .
```

The decoded payload contains the full SPDX document listing every package and dependency in the image.

## Inspect the signing certificate

To see exactly which GitHub Actions run produced the signature:

```bash
cosign verify \
  --certificate-identity-regexp \
    "https://github.com/heimops/mimir/.github/workflows/release.yml@refs/tags/.*" \
  --certificate-oidc-issuer "https://token.actions.githubusercontent.com" \
  ghcr.io/heimops/mimir-operator:<version> \
  | jq -r '.[0] | .optional'
```

The output includes `Issuer`, `Subject`, `githubWorkflowRepository`, `githubWorkflowRef`, and `githubWorkflowSha` — pinning the signature to a specific commit and workflow run.

## Download the SBOM from a GitHub Release

The SPDX SBOM files are also attached as assets to every [GitHub Release](https://github.com/heimops/mimir/releases):

- `sbom-operator.spdx.json` — operator image
- `sbom-app.spdx.json` — Python backup app image

## What is signed

| Artifact | Signed | SBOM attestation |
|----------|--------|-----------------|
| `ghcr.io/heimops/mimir-operator` | yes | yes |
| `ghcr.io/heimops/mimir` | yes | yes |
| Helm chart (OCI) | no | no |
