from __future__ import absolute_import, unicode_literals
from celery import Celery
from django.conf import settings
import os
import django

os.environ.setdefault('DJANGO_SETTINGS_MODULE', 'config.settings')
django.setup()

app = Celery('config')

app.conf.update(
    broker_url=settings.CELERY_BROKER_URL,
    result_backend='django-db',
    task_serializer='json',
    result_serializer='json',
    accept_content=['json'],
    timezone='Asia/Kolkata',
    task_default_queue = 'default',
    enable_utc=False,
    worker_concurrency=6,
    task_soft_time_limit=900,
    task_time_limit=1200,
    acks_late=True, 
    prefetch_multiplier=1,
    broker_connection_retry_on_startup=True, 
    broker_transport_options={"visibility_timeout": 1100},
    task_track_started=True,
    cache_backend='default',
    database_engine_options={'echo': True},
    result_expires=3600,
    worker_send_task_events=True,
    task_send_sent_event=True,
    worker_pool_restarts=True,
)

# Use Django Celery Beat for scheduling tasks
app.conf.beat_scheduler = 'django_celery_beat.schedulers:DatabaseScheduler'

# Load task modules from all registered Django app configs
app.autodiscover_tasks()


@app.task(bind=True)
def debug_task(self) -> None:
    """A simple debug task to display request details."""
    print(f'Request: {self.request!r}')