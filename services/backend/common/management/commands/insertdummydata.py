from django.core.management.base import BaseCommand, CommandError
from django.db import IntegrityError
from django.contrib.auth.models import User


class Command(BaseCommand):
    help = 'Create a superuser'

    def handle(self, *args, **options):
        self.execute_all()

    def execute_all(self):
        self.create_super_user()

    def create_super_user(self):
        self.stdout.write('Creating superuser')

        username = "admin"
        email = "admin@example.com"
        password = "admin@123"
        first_name= "admin"
        last_name= "admin"
        
        try:
            if User.objects.filter(email=email).exists():
                self.stdout.write(self.style.WARNING(f'Superuser already exists: {email}'))
                return

            user = User.objects.create_superuser(
                username=username,
                email=email,
                password=password,
                first_name=first_name,
                last_name=last_name
            )

            self.stdout.write(self.style.SUCCESS(f'Superuser created successfully: {user}'))

        except IntegrityError as e:
            self.stdout.write(self.style.ERROR(f'Error creating superuser: {e}'))
            raise CommandError('Error creating superuser.')