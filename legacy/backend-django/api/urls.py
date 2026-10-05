from django.urls import path
from .views import CSVUploadView, StatusView

urlpatterns = [
    path('upload/', CSVUploadView.as_view(), name='csv-upload'),
    path('status/<str:request_id>/', StatusView.as_view(), name='status'),
]
