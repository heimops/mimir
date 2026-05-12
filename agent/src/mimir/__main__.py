import logging

import click

from .backup import run_backup
from .config import Config
from .scheduler import run_scheduler

logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s %(levelname)s %(name)s %(message)s",
)


@click.group()
def cli() -> None:
    """Mimir — Kubernetes persistent volume backup tool."""


@cli.command()
def backup() -> None:
    """Run a one-shot backup immediately and exit."""
    run_backup(Config())


@cli.command()
def schedule() -> None:
    """Start the scheduler (blocking). Requires MIMIR_SCHEDULE to be set."""
    run_scheduler(Config())


if __name__ == "__main__":
    cli()
