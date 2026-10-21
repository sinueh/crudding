from src.app.domain.resource import Resource
from src.app.ports.resource_repository import ResourceRepository


class ResourceRepositoryAdapter(ResourceRepository):
    """Adapter de persistência de Resource."""
    def save(self, item: Resource) -> Resource:
        raise NotImplementedError

    def find_by_id(self, item_id: str) -> Resource | None:
        raise NotImplementedError

    def find_all(self) -> list[Resource]:
        raise NotImplementedError

    def delete(self, item_id: str) -> None:
        raise NotImplementedError
