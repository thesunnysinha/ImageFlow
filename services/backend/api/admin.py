from django.contrib import admin
from .models import ProcessingRequest, Product

@admin.register(ProcessingRequest)
class ProcessingRequestAdmin(admin.ModelAdmin):
    list_display = ('request_id', 'status', 'created_at', 'updated_at')
    search_fields = ('request_id', 'status')
    list_filter = ('status', 'created_at')
    readonly_fields = ('request_id', 'created_at', 'updated_at')

@admin.register(Product)
class ProductAdmin(admin.ModelAdmin):
    list_display = ('processing_request', 'serial_number', 'product_name')
    search_fields = ('product_name',)
    list_filter = ('processing_request',)

