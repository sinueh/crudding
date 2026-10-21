from src.app.domain.model.resource import Resource


class GetResource:
    def __init__(self, repository):
        self.repository = repository

    def execute(self, item_id: str) -> Resource | None:
        return self.repository.find_by_id(item_id)
