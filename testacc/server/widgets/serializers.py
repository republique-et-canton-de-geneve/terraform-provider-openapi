from rest_framework import serializers

from . import models


class WidgetSerializer(serializers.ModelSerializer):
    class Meta:
        model = models.Widget
        fields = ('id', 'name', 'size', 'created_at')
        read_only_fields = ('id', 'created_at')
