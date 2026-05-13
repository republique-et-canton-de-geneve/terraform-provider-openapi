from django.db import models


class Widget(models.Model):
    name = models.CharField(max_length=255)
    size = models.PositiveIntegerField(null=True, blank=True)
    created_at = models.DateTimeField(auto_now_add=True)

    class Meta:
        ordering = ['id']
