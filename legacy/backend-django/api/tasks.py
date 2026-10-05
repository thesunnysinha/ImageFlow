import time
import logging
import requests
from celery import shared_task
from .models import ProcessingRequest, Product

logger = logging.getLogger(__name__)

@shared_task
def process_csv_images(request_id: str) -> None:
    """
    Asynchronously processes images for a CSV processing request.
    
    For each product in the request:
      - Simulates image compression by modifying the image URL.
      - Updates the output_image_urls field.
      - On completion, updates the request status and triggers a webhook callback if provided.
    """
    try:
        processing_request = ProcessingRequest.objects.get(request_id=request_id)
        processing_request.status = 'processing'
        processing_request.save()

        products = processing_request.products.all()
        for product in products:
            _process_product_images(product)
        
        processing_request.status = 'completed'
        processing_request.save()
        logger.info("Completed processing for request_id: %s", request_id)
        
        # Trigger webhook callback if provided.
        if processing_request.webhook_url:
            _trigger_webhook_callback(processing_request)
    except Exception as e:
        logger.exception("Error processing images for request_id %s: %s", request_id, str(e))
        processing_request.status = 'failed'
        processing_request.save()

def _process_product_images(product: Product) -> None:
    """
    Private helper method to simulate image processing for a product.
    
    Args:
        product: The Product instance to process.
    """
    input_urls = product.input_image_urls.split(',')
    output_urls = []
    for url in input_urls:
        url = url.strip()
        # Simulate image compression by appending a suffix.
        processed_url = url.replace('.jpg', '-compressed.jpg')
        output_urls.append(processed_url)
        logger.debug("Processed image %s to %s", url, processed_url)
        time.sleep(1)  # Simulate processing delay.
    product.output_image_urls = ','.join(output_urls)
    product.save()

def _trigger_webhook_callback(processing_request: ProcessingRequest) -> None:
    """
    Private helper method to trigger a webhook callback after processing is complete.
    
    Args:
        processing_request: The ProcessingRequest instance that has completed processing.
    """
    payload = {
        "request_id": str(processing_request.request_id),
        "status": processing_request.status
    }
    try:
        response = requests.post(processing_request.webhook_url, json=payload, timeout=5)
        response.raise_for_status()
        logger.info("Webhook callback succeeded for request_id: %s", processing_request.request_id)
    except Exception as webhook_error:
        logger.error("Webhook callback failed for request_id: %s, error: %s", processing_request.request_id, str(webhook_error))
