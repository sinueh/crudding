
from dataclasses import asdict

from src.app.domain.model.resource import Resource


def present_resource(item: Resource) -> dict:
    return asdict(item)

