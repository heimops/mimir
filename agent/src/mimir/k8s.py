from __future__ import annotations

import hashlib
import logging

from kubernetes import client
from kubernetes import config as k8s_config

logger = logging.getLogger(__name__)


def load_config() -> None:
    try:
        k8s_config.load_incluster_config()
    except k8s_config.ConfigException:
        k8s_config.load_kube_config()


def list_pvcs(namespace: str, pvc_names: list[str] | None) -> list[str]:
    v1 = client.CoreV1Api()
    result = v1.list_namespaced_persistent_volume_claim(namespace)
    names = [pvc.metadata.name for pvc in result.items]
    if pvc_names:
        names = [n for n in names if n in pvc_names]
    return names


def _job_name(namespace: str, pvc_name: str, timestamp: str) -> str:
    base = f"mimir-{namespace}-{pvc_name}-{timestamp}"
    if len(base) <= 63:
        return base.lower()
    # keep timestamp at the end for readability; use a hash prefix to stay unique
    digest = hashlib.sha256(f"{namespace}/{pvc_name}".encode()).hexdigest()[:6]
    return f"mimir-{digest}-{timestamp}".lower()


def create_backup_job(
    *,
    namespace: str,
    pvc_name: str,
    timestamp: str,
    rclone_env: dict[str, str],
    remote_path: str,
    backup_image: str,
    ttl_seconds: int,
    service_account: str,
    image_pull_secret: str | None = None,
) -> str:
    """Create a Kubernetes Job that syncs pvc_name to remote_path via rclone.

    The Job is created in the same namespace as the PVC (required for volume access).
    """
    batch = client.BatchV1Api()
    name = _job_name(namespace, pvc_name, timestamp)

    env = [client.V1EnvVar(name=k, value=v) for k, v in rclone_env.items()]

    container = client.V1Container(
        name="rclone",
        image=backup_image,
        args=["sync", "/data", remote_path, "--progress"],
        env=env,
        volume_mounts=[
            client.V1VolumeMount(name="data", mount_path="/data", read_only=True)
        ],
    )

    pod_spec = client.V1PodSpec(
        containers=[container],
        volumes=[
            client.V1Volume(
                name="data",
                persistent_volume_claim=client.V1PersistentVolumeClaimVolumeSource(
                    claim_name=pvc_name, read_only=True
                ),
            )
        ],
        restart_policy="Never",
    )
    if service_account:
        pod_spec.service_account_name = service_account
    if image_pull_secret:
        pod_spec.image_pull_secrets = [client.V1LocalObjectReference(name=image_pull_secret)]

    job = client.V1Job(
        api_version="batch/v1",
        kind="Job",
        metadata=client.V1ObjectMeta(
            name=name,
            namespace=namespace,
            labels={
                "app.kubernetes.io/name": "mimir",
                "mimir/source-namespace": namespace,
                "mimir/source-pvc": pvc_name,
            },
        ),
        spec=client.V1JobSpec(
            template=client.V1PodTemplateSpec(
                metadata=client.V1ObjectMeta(labels={"app.kubernetes.io/name": "mimir"}),
                spec=pod_spec,
            ),
            backoff_limit=2,
            ttl_seconds_after_finished=ttl_seconds,
        ),
    )

    batch.create_namespaced_job(namespace=namespace, body=job)
    logger.info("Created backup job %s → %s", name, remote_path)
    return name
