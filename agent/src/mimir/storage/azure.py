from ..config import AzureConfig
from .base import StorageBackend


class AzureBackend(StorageBackend):
    def __init__(self, config: AzureConfig) -> None:
        self.config = config

    def rclone_env(self) -> dict[str, str]:
        return {
            "RCLONE_CONFIG_REMOTE_TYPE": "azureblob",
            "RCLONE_CONFIG_REMOTE_ACCOUNT": self.config.account,
            "RCLONE_CONFIG_REMOTE_KEY": self.config.key,
        }

    def remote_path(self, namespace: str, pvc_name: str, timestamp: str) -> str:
        return f"remote:{self.config.container}/{namespace}/{pvc_name}/{timestamp}"
