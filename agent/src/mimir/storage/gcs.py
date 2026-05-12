from ..config import GCSConfig
from .base import StorageBackend


class GCSBackend(StorageBackend):
    def __init__(self, config: GCSConfig) -> None:
        self.config = config

    def rclone_env(self) -> dict[str, str]:
        env: dict[str, str] = {
            "RCLONE_CONFIG_REMOTE_TYPE": "google cloud storage",
            "RCLONE_CONFIG_REMOTE_BUCKET_POLICY_ONLY": "true",
        }
        if self.config.service_account_json:
            env["RCLONE_CONFIG_REMOTE_SERVICE_ACCOUNT_CREDENTIALS"] = (
                self.config.service_account_json
            )
        return env

    def remote_path(self, namespace: str, pvc_name: str, timestamp: str) -> str:
        return f"remote:{self.config.bucket}/{namespace}/{pvc_name}/{timestamp}"
