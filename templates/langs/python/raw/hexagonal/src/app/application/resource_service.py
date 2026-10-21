
from dataclasses import asdict
from src.app.domain.resource import Resource
from src.app.ports.resource_repository import ResourceRepository
from src.app.infrastructure.resource_repository import ResourceRepositoryAdapter


def _to_dict(item: Resource) -> dict:
    return asdict(item)


class ResourceService:
    def __init__(self, repository: ResourceRepository):
        self.repository = repository

    def list_all(self) -> list[dict]:
        return [_to_dict(item) for item in self.repository.find_all()]

    def get(self, item_id: str) -> dict | None:
        item = self.repository.find_by_id(item_id)
        return _to_dict(item) if item is not None else None

    def create(self, payload: dict) -> dict:
        raise NotImplementedError

    def update(self, item_id: str, payload: dict) -> dict:
        raise NotImplementedError

    def delete(self, item_id: str) -> None:
        self.repository.delete(item_id)


_service = ResourceService(ResourceRepositoryAdapter())

list_all = _service.list_all
get = _service.get
create = _service.create
update = _service.update
delete = _service.delete

