from rest_framework import mixins, viewsets

from . import models, serializers


class WidgetViewSet(
    mixins.CreateModelMixin,
    mixins.RetrieveModelMixin,
    mixins.UpdateModelMixin,
    mixins.DestroyModelMixin,
    viewsets.GenericViewSet,
):
    queryset = models.Widget.objects.all()
    serializer_class = serializers.WidgetSerializer
    http_method_names = ['delete', 'get', 'head', 'options', 'patch', 'post']
