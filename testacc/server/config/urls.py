from django.http import JsonResponse
from django.urls import include, path
from drf_spectacular.views import SpectacularAPIView


def health(request):
    return JsonResponse({'status': 'ok'})


urlpatterns = [
    path('health/', health),
    path('api/schema/', SpectacularAPIView.as_view(), name='schema'),
    path('', include('widgets.urls')),
]
