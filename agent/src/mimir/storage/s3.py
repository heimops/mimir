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
            # IBM COS (and most non-AWS S3-compatible endpoints) require path-style
            # requests (endpoint/bucket). Without this, rclone uses virtual-hosted
            # style (bucket.endpoint) which IBM COS rejects with 403.
            "RCLONE_CONFIG_REMOTE_FORCE_PATH_STYLE": "true",
            # Assume bucket exists — HMAC credentials often lack s3:CreateBucket.
            # Must be set at the named-remote level, not as a global RCLONE_S3_* var.
            "RCLONE_CONFIG_REMOTE_NO_CHECK_BUCKET": "true",
            # Skip post-upload HeadObject verification — credentials may lack s3:GetObject.
            "RCLONE_CONFIG_REMOTE_NO_HEAD": "true",
            # IBM COS (and some S3-compatible endpoints) stall rclone PUT requests over
            # HTTP/2 multiplexed connections — transfers start but stay at 0 B/s forever.
            # Forcing HTTP/1.1 unblocks uploads.
            "RCLONE_CONFIG_REMOTE_DISABLE_HTTP2": "true",
        }
        if self.config.endpoint_url:
            env["RCLONE_CONFIG_REMOTE_ENDPOINT"] = self.config.endpoint_url
        return env

    def remote_path(self, namespace: str, pvc_name: str, timestamp: str) -> str:
        # Trailing slash tells rclone this is always a directory, skipping the
        # HeadObject call rclone uses to distinguish files from directory prefixes.
        return f"remote:{self.config.bucket}/{namespace}/{pvc_name}/{timestamp}/"
