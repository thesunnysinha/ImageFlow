import os
import shutil
from django.core.management.base import BaseCommand
from django.conf import settings


class Command(BaseCommand):
    """Django command to delete migration and __pycache__ directories"""

    def handle(self, *args, **options):
        project_directory = settings.BASE_DIR
        self.ignore_migrations = set(options.get('ignore_migrations', []))
        self.ignore_migrations.add("0001_vector_extension.py")
        self.ignore_folders = set(options.get('ignore_folders', []))
        self.ignore_folders.add(".venv")

        app_directories = [
            app for app in os.listdir(project_directory)
            if os.path.isdir(os.path.join(project_directory, app))
        ]

        for app in app_directories:
            app_directory = os.path.join(project_directory, app)
            self.delete_nested_directories(app_directory)

    def delete_nested_directories(self, directory):
        if os.path.basename(directory) in self.ignore_folders:
            self.stdout.write(self.style.WARNING(f"Ignored folder: {directory}"))
            return

        migrations_directory = os.path.join(directory, "migrations")
        pycache_directory = os.path.join(directory, "__pycache__")

        self.delete_directory(migrations_directory)
        self.delete_directory(pycache_directory)

        # Traverse subdirectories
        for root, dirs, files in os.walk(directory):
            for dir_name in dirs:
                subdirectory = os.path.join(root, dir_name)
                if dir_name in self.ignore_folders:
                    self.stdout.write(self.style.WARNING(f"Ignored folder: {subdirectory}"))
                    continue

                migrations_directory = os.path.join(subdirectory, "migrations")
                pycache_directory = os.path.join(subdirectory, "__pycache__")

                self.delete_directory(migrations_directory)
                self.delete_directory(pycache_directory)

    def delete_directory(self, directory):
        if os.path.basename(directory) == "migrations":
            for root, _, files in os.walk(directory):
                for file in files:
                    if file in self.ignore_migrations:
                        self.stdout.write(self.style.WARNING(f"Ignored migration file: {os.path.join(root, file)}"))
                        return

        if os.path.exists(directory):
            shutil.rmtree(directory)
            self.stdout.write(self.style.SUCCESS(f"Deleted: {directory}"))
        else:
            self.stdout.write(self.style.ERROR(f"Directory not found: {directory}"))