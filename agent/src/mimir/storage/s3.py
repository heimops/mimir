from ..config import S3Config
from .base import StorageBackend


class S3Backend(StorageBackend):
    def __init__(self, config: S3Config) -> None:
        self.config = config

    def rclone_env(self) -> dict[str, str]:
        env: dict[str, str] = {
            "RCLONE_CONFIG_REMOTE_TYPE": "s3",
            "RCLONE_CONFIG_REMOTE_PROVIDER": self.config.provider,
            "RCLONE_CONFIG_REMOTE_ACCESS_KEY_ID": self.config.access_key_id,
            "RCLONE_CONFIG_REMOTE_SECRET_ACCESS_KEY": self.config.secret_access_key,
            "RCLONE_CONFIG_REMOTE_REGION": self.config.region,
            # Skip bucket creation check — the bucket must exist beforehand.
            # Required when the HMAC credentials lack s3:CreateBucket permission.
            "RCLONE_S3_NO_CHECK_BUCKET": "true",
        }
        if self.config.endpoint_url:
            env["RCLONE_CONFIG_REMOTE_ENDPOINT"] = self.config.endpoint_url
        return env

    def remote_path(self, namespace: str, pvc_name: str, timestamp: str) -> str:
        return f"remote:{self.config.bucket}/{namespace}/{pvc_name}/{timestamp}"
