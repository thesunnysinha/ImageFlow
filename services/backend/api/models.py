import uuid
from django.db import models

class ProcessingRequest(models.Model):
    """
    Model representing a CSV processing request.
    
    Attributes:
        request_id (UUIDField): Unique identifier for the request.
        csv_file (FileField): Uploaded CSV file containing product data.
        webhook_url (URLField): Optional webhook URL for post-processing notification.
        status (CharField): Current processing status (pending, processing, completed, failed).
        created_at (DateTimeField): Timestamp when the request was created.
        updated_at (DateTimeField): Timestamp when the request was last updated.
    """
    request_id = models.UUIDField(default=uuid.uuid4, editable=False, unique=True)
    csv_file = models.FileField(upload_to='uploads/')
    webhook_url = models.URLField(null=True, blank=True)
    status = models.CharField(max_length=20, default='pending')
    created_at = models.DateTimeField(auto_now_add=True)
    updated_at = models.DateTimeField(auto_now=True)

class Product(models.Model):
    """
    Model representing a product and its associated image URLs.
    
    Attributes:
        processing_request (ForeignKey): Link to the processing request.
        serial_number (IntegerField): Serial number from the CSV.
        product_name (CharField): Name of the product.
        input_image_urls (TextField): Comma-separated input image URLs.
        output_image_urls (TextField): Comma-separated output image URLs after processing.
    """
    processing_request = models.ForeignKey(ProcessingRequest, related_name='products', on_delete=models.CASCADE)
    serial_number = models.IntegerField()
    product_name = models.CharField(max_length=255)
    input_image_urls = models.TextField()
    output_image_urls = models.TextField(null=True, blank=True)
