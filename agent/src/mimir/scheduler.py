from __future__ import annotations

import logging

from apscheduler.schedulers.blocking import BlockingScheduler
from apscheduler.triggers.cron import CronTrigger

from .backup import run_backup
from .config import Config

logger = logging.getLogger(__name__)


def run_scheduler(config: Config) -> None:
    if not config.schedule:
        raise ValueError("MIMIR_SCHEDULE must be set to a cron expression for scheduled mode")

    scheduler = BlockingScheduler()
    scheduler.add_job(
        run_backup,
        CronTrigger.from_crontab(config.schedule),
        args=[config],
        id="backup",
        name="mimir backup",
    )
    logger.info("Scheduler started with cron: %s", config.schedule)
    try:
        scheduler.start()
    except (KeyboardInterrupt, SystemExit):
        logger.info("Scheduler stopped")
