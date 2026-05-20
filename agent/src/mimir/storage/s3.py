from ..config import S3Config
from .base import StorageBackend

_AWS_CLI_IMAGE = "amazon/aws-cli:latest"


class S3Backend(StorageBackend):
    def __init__(self, config: S3Config) -> None:
        self.config = config

    def env_vars(self) -> dict[str, str]:
        env: dict[str, str] = {
            "AWS_ACCESS_KEY_ID": self.config.access_key_id,
            "AWS_SECRET_ACCESS_KEY": self.config.secret_access_key,
            "AWS_DEFAULT_REGION": self.config.region,
        }
        if self.config.endpoint_url:
            env["AWS_ENDPOINT_URL"] = self.config.endpoint_url
        return env

    def job_image(self) -> str:
        return _AWS_CLI_IMAGE

    def job_args(self, remote_path: str) -> list[str]:
        return ["s3", "sync", "/data", remote_path, "--no-progress"]

    def remote_path(self, namespace: str, pvc_name: str, timestamp: str) -> str:
        return f"s3://{self.config.bucket}/{namespace}/{pvc_name}/{timestamp}/"
