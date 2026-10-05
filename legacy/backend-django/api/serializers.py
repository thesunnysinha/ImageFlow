from rest_framework import serializers
from .models import ProcessingRequest, Product

class ProductSerializer(serializers.ModelSerializer):
    """
    Serializer for the Product model.
    """
    class Meta:
        model = Product
        fields = '__all__'

class ProcessingRequestSerializer(serializers.ModelSerializer):
    """
    Serializer for the ProcessingRequest model, including associated products.
    """
    products = ProductSerializer(many=True, read_only=True)
    
    class Meta:
        model = ProcessingRequest
        fields = '__all__'
