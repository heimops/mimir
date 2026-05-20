from __future__ import annotations

import logging
from datetime import datetime, timezone

from . import k8s
from .config import Config
from .storage import get_backend

logger = logging.getLogger(__name__)


def _timestamp() -> str:
    return datetime.now(timezone.utc).strftime("%Y%m%d%H%M%S")


def run_backup(config: Config) -> None:
    k8s.load_config()
    backend = get_backend(config)
    env = backend.env_vars()
    image = backend.job_image() or config.backup_image
    ts = _timestamp()

    for namespace in config.namespaces:
        try:
            pvcs = k8s.list_pvcs(namespace, config.pvcs)
        except Exception:
            logger.exception("Failed to list PVCs in namespace %s", namespace)
            continue

        logger.info("Namespace %s: found %d PVC(s) to back up", namespace, len(pvcs))

        for pvc_name in pvcs:
            remote_path = backend.remote_path(namespace, pvc_name, ts)
            try:
                k8s.create_backup_job(
                    namespace=namespace,
                    pvc_name=pvc_name,
                    timestamp=ts,
                    env=env,
                    job_args=backend.job_args(remote_path),
                    backup_image=image,
                    ttl_seconds=config.backup_job_ttl,
                    service_account=config.job_service_account,
                    image_pull_secret=config.backup_image_pull_secret,
                )
            except Exception:
                logger.exception("Failed to create backup job for %s/%s", namespace, pvc_name)
