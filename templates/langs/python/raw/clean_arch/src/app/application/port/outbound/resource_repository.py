from abc import ABC, abstractmethod

from src.app.domain.model.resource import Resource


class ResourceRepository(ABC):
    @abstractmethod
    def save(self, item: Resource) -> Resource:
        raise NotImplementedError

    @abstractmethod
    def find_by_id(self, item_id: str) -> Resource | None:
        raise NotImplementedError

    @abstractmethod
    def find_all(self) -> list[Resource]:
        raise NotImplementedError

    @abstractmethod
    def delete(self, item_id: str) -> None:
        raise NotImplementedError
