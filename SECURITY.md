# Security Policy

## Supported versions

| Version | Supported |
|---------|-----------|
| latest (`main`) | yes |

Once the project reaches a stable release, only the two most recent minor versions will receive security fixes.

## Reporting a vulnerability

**Do not open a public GitHub issue for security vulnerabilities.**

Please report them by emailing **security@mimir.io** (replace with your actual contact before publishing).  
Include:

- A description of the vulnerability and its potential impact.
- Steps to reproduce or a proof-of-concept.
- The version or commit hash where you found it.

You will receive an acknowledgement within **72 hours** and a resolution timeline within **7 days**.

## Disclosure policy

We follow [coordinated disclosure](https://en.wikipedia.org/wiki/Coordinated_vulnerability_disclosure). Once a fix is ready we will:

1. Release a patched version.
2. Publish a GitHub Security Advisory (CVE if applicable).
3. Credit the reporter unless they prefer to remain anonymous.

## Scope

| In scope | Out of scope |
|----------|--------------|
| Operator container image vulnerabilities | Vulnerabilities in rclone (report upstream) |
| Privilege escalation via CRD/RBAC | Misconfigured user clusters |
| Storage credential leakage in logs or Job specs | CVEs in base OS packages already tracked by the image scanner |

## Security practices

- Container images are built from `gcr.io/distroless/static:nonroot` (operator) and `python:3.12-slim` (app).
- Images are signed with [cosign](https://docs.sigstore.dev/cosign/overview/) keyless signing on every release; signatures are recorded in the public [Sigstore Rekor](https://rekor.sigstore.dev) transparency log.
- Each image ships a signed [SPDX SBOM attestation](https://spdx.dev/) verifiable with `cosign verify-attestation --type spdx`.
- See [VERIFICATION.md](VERIFICATION.md) for step-by-step instructions on verifying signatures and attestations.
- Dependency updates are automated via Dependabot.
