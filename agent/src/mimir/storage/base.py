from abc import ABC, abstractmethod


class StorageBackend(ABC):
    @abstractmethod
    def rclone_env(self) -> dict[str, str]:
        """Environment variables that configure the rclone remote named 'remote'."""

    @abstractmethod
    def remote_path(self, namespace: str, pvc_name: str, timestamp: str) -> str:
        """rclone destination path for a single PVC backup."""
