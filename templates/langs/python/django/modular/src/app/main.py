import sys
from django.conf import settings

settings.configure(
    DEBUG=True,
    SECRET_KEY="development-only",
    ROOT_URLCONF=__name__,
    ALLOWED_HOSTS=["*"],
    MIDDLEWARE=[],
    INSTALLED_APPS=[],
)

import django

django.setup()

from django.urls import include, path
from src.app.resource.controller import urlpatterns as resource_urls

urlpatterns = [
    path("resource/", include(resource_urls)),
]

if __name__ == "__main__":
    from django.core.management import execute_from_command_line

    execute_from_command_line(sys.argv)