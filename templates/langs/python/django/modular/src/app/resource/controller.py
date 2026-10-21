import json
from django.http import HttpResponse, JsonResponse
from django.urls import path
from django.views.decorators.csrf import csrf_exempt
import src.app.resource.service as service


@csrf_exempt
def list_or_create(request):
    try:
        if request.method == "GET":
            return JsonResponse(service.list_all(), safe=False)
        if request.method == "POST":
            payload = json.loads(request.body) if request.body else {}
            return JsonResponse(service.create(payload), status=201)
        return HttpResponse(status=405)
    except NotImplementedError:
        return JsonResponse({"detail": "Resource: not implemented"}, status=501)


@csrf_exempt
def detail(request, item_id: str):
    try:
        if request.method == "GET":
            item = service.get(item_id)
            if item is None:
                return JsonResponse({"detail": f"Resource {item_id} not found"}, status=404)
            return JsonResponse(item)
        if request.method == "PATCH":
            payload = json.loads(request.body) if request.body else {}
            return JsonResponse(service.update(item_id, payload))
        if request.method == "DELETE":
            service.delete(item_id)
            return HttpResponse(status=204)
        return HttpResponse(status=405)
    except NotImplementedError:
        return JsonResponse({"detail": "Resource: not implemented"}, status=501)


urlpatterns = [
    path("", list_or_create),
    path("<str:item_id>/", detail),
]

