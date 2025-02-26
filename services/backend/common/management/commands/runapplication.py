import subprocess
import logging
from django.core.management.base import BaseCommand
from django.core.management import call_command
import structlog

logger = structlog.get_logger(__name__)

class Command(BaseCommand):
    help = 'Run all necessary services for Django application'

    def add_arguments(self, parser):
        parser.add_argument('--celery', action='store_true', help='Run Celery worker')
        parser.add_argument('--celery-beat', action='store_true', help='Run Celery beat')

    def handle(self, *args, **options):
        logger = logging.getLogger(__name__)

        # Setup the application
        if not options['celery'] and not options['celery_beat']:
            self.stdout.write('Setting up the application...')
            try:
                call_command('setupapplication')
                self.stdout.write(self.style.SUCCESS('Application setup complete!'))
            except Exception as e:
                logger.error('Error setting up the application: %s', e)
                self.stdout.write(self.style.ERROR('Error setting up the application!'))
                return

        # Wait for the database to be ready
        if not options['celery'] and not options['celery_beat']:
            self.stdout.write('Waiting for the database to be ready...')
            try:
                subprocess.run(['/app/wait-for-it.sh', 'db:5432', '--timeout=0'], check=True)
                self.stdout.write(self.style.SUCCESS('Database is ready!'))
            except subprocess.CalledProcessError as e:
                logger.error('Database is not ready: %s', e)
                self.stdout.write(self.style.ERROR('Database is not ready!'))
                return

        # Apply Migrations
        self.stdout.write('Applying migrations...')
        try:
            call_command('migrate', '--no-input')
            self.stdout.write(self.style.SUCCESS('Migrations complete!'))
        except Exception as e:
            logger.error('Error applying migrations: %s', e)
            self.stdout.write(self.style.ERROR('Error applying migrations!'))
            return

        # Insert Dummy Data
        if not options['celery'] and not options['celery_beat']:
            self.stdout.write('Inserting dummy data...')
            try:
                call_command('insertdummydata')
            except Exception as e:
                logger.error('Error inserting dummy data: %s', e)
                self.stdout.write(self.style.ERROR('Error inserting dummy data!'))
                return

        # Wait for the backend to be ready
        # Start Celery worker or Celery beat if --celery or --celery-beat option is provided
        if options['celery'] or options['celery_beat']:
            self.stdout.write('Waiting for the backend to be ready...')
            try:
                subprocess.run(['/app/wait-for-it.sh', 'backend:8000', '--timeout=0'], check=True)
                self.stdout.write(self.style.SUCCESS('Backend is ready!'))
            except subprocess.CalledProcessError as e:
                logger.error('Backend is not ready: %s', e)
                self.stdout.write(self.style.ERROR('Backend is not ready!'))
                return

        # Start Celery worker if --celery option is provided
        if options['celery']:
            self.stdout.write(self.style.SUCCESS(f"Starting Celery worker"))
            celery_worker_command = ['celery', '-A', 'config', 'worker', '--loglevel=info']
            try:
                subprocess.run(celery_worker_command, check=True)
            except subprocess.CalledProcessError as e:
                self.stderr.write(self.style.ERROR(f"Error starting Celery worker: {e}"))
            self.stdout.write(self.style.SUCCESS(f"Started Celery worker"))

        # Start Celery beat if --celery-beat option is provided
        if options['celery_beat']:
            self.stdout.write(self.style.SUCCESS(f"Starting Celery beat"))
            celery_beat_command = ['celery', '-A', 'config', 'beat', '--loglevel=info']
            try:
                subprocess.run(celery_beat_command, check=True)
            except subprocess.CalledProcessError as e:
                self.stderr.write(self.style.ERROR(f"Error starting Celery beat: {e}"))
            self.stdout.write(self.style.SUCCESS(f"Started Celery beat"))

        # Start Gunicorn server
        self.stdout.write('Starting Gunicorn server...')
        gunicorn_command = ['gunicorn', 'config.wsgi:application', '--config', '/app/gunicorn.prod.conf.py', '--reload']
        try:
            subprocess.run(gunicorn_command, check=True)
            self.stdout.write(self.style.SUCCESS('Gunicorn server started!'))
        except subprocess.CalledProcessError as e:
            logger.error('Error starting Gunicorn server: %s', e)
            self.stdout.write(self.style.ERROR('Error starting Gunicorn server!'))
            return