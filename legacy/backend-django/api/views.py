import csv
import io
import logging
from typing import List, Optional

from django.shortcuts import get_object_or_404
from rest_framework import status
from rest_framework.response import Response
from rest_framework.views import APIView

from .models import ProcessingRequest, Product
from .serializers import ProcessingRequestSerializer
from .tasks import process_csv_images

logger = logging.getLogger(__name__)

class CSVUploadView(APIView):
    """
    API endpoint to upload a CSV file with product image URLs.
    
    This endpoint:
      - Validates the CSV format.
      - Stores the file and creates product records.
      - Accepts an optional webhook URL.
      - Enqueues an asynchronous task for image processing.
      
    Returns:
      A unique request ID upon successful submission.
    """
    def post(self, request, format: Optional[str] = None) -> Response:
        file_obj = request.FILES.get('file', None)
        if not file_obj:
            logger.error("No file provided in the request.")
            return Response({"error": "No file provided"}, status=status.HTTP_400_BAD_REQUEST)

        try:
            webhook_url = request.data.get('webhook_url')
            # Create a new processing request record.
            processing_request = ProcessingRequest.objects.create(csv_file=file_obj, webhook_url=webhook_url)
            
            # Parse CSV content and create products.
            rows = self._parse_csv(file_obj)
            for row in rows:
                self._create_product_from_row(row, processing_request)
            
            # Enqueue asynchronous image processing task.
            process_csv_images.delay(str(processing_request.request_id))
            logger.info("CSV uploaded. Processing task enqueued for request_id: %s", processing_request.request_id)
            return Response({"request_id": processing_request.request_id}, status=status.HTTP_202_ACCEPTED)
        except Exception as e:
            logger.exception("Error processing CSV file: %s", str(e))
            return Response({"error": "Error processing CSV file"}, status=status.HTTP_500_INTERNAL_SERVER_ERROR)

    def _parse_csv(self, file_obj) -> List[List[str]]:
        """
        Private helper method to parse the CSV file.

        Args:
            file_obj: Uploaded CSV file object.

        Returns:
            A list of rows where each row is a list of string values.
        """
        decoded_file = file_obj.read().decode('utf-8')
        io_string = io.StringIO(decoded_file)
        reader = csv.reader(io_string, delimiter=',')
        # Optionally skip header row.
        header = next(reader, None)
        rows = [row for row in reader if row]
        return rows

    def _create_product_from_row(self, row: List[str], processing_request: ProcessingRequest) -> None:
        """
        Private helper method to create a Product instance from a CSV row.

        Args:
            row: List of strings representing CSV columns.
            processing_request: The ProcessingRequest instance to link the product to.
        """
        if len(row) < 3:
            logger.warning("Row skipped due to insufficient columns: %s", row)
            return
        try:
            serial_number = int(row[0])
            product_name = row[1]
            input_image_urls = row[2]
            Product.objects.create(
                processing_request=processing_request,
                serial_number=serial_number,
                product_name=product_name,
                input_image_urls=input_image_urls
            )
        except Exception as e:
            logger.exception("Error creating product from row %s: %s", row, str(e))


class StatusView(APIView):
    """
    API endpoint to check the processing status using a unique request ID.
    """
    def get(self, request, request_id: str, format: Optional[str] = None) -> Response:
        processing_request = get_object_or_404(ProcessingRequest, request_id=request_id)
        serializer = ProcessingRequestSerializer(processing_request)
        logger.info("Status queried for request_id: %s, status: %s", request_id, processing_request.status)
        return Response(serializer.data, status=status.HTTP_200_OK)