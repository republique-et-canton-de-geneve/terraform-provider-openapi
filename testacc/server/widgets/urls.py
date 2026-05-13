from django.urls import path

from . import views

widget_list = views.WidgetViewSet.as_view({'post': 'create'})
widget_detail = views.WidgetViewSet.as_view({
    'delete': 'destroy',
    'get': 'retrieve',
    'patch': 'partial_update',
})

urlpatterns = [
    path('api/v1/widgets/', widget_list),
    path('api/v1/widgets/<int:pk>/', widget_detail),
]
