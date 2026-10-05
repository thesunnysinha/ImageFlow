import logging
from django.core.management.base import BaseCommand
from django.core.management import call_command
from django.apps import apps

class Command(BaseCommand):
    """Django command to execute custom management commands"""

    def handle(self, *args, **options):
        logger = logging.getLogger(__name__)

        # Delete old migrations
        self.stdout.write('Deleting old migrations and pycache...')
        try:
            call_command('deletemigrationspycache')
        except Exception as e:
            logger.error('Error deleting migrations: %s', e)
            self.stdout.write(self.style.ERROR('Error deleting migrations and pycache!'))
            return

        # Create new migrations
        self.stdout.write('Creating new migrations...')
        try:
            call_command('makemigrations', 'api')
        except Exception as e:
            logger.error('Error creating migrations: %s', e)
            self.stdout.write(self.style.ERROR(f'Error creating migrations! {e}'))
            return

        # Collect Static Data
        self.stdout.write('Collecting static files...')
        try:
            call_command('collectstatic', '--no-input')
            self.stdout.write(self.style.SUCCESS('Collectstatic complete!'))
        except Exception as e:
            logger.error('Error collecting static files: %s', e)
            self.stdout.write(self.style.ERROR('Error collecting static files!'))
            return