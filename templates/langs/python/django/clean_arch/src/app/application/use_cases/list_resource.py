from src.app.domain.model.resource import Resource


class ListResource:
    def __init__(self, repository):
        self.repository = repository

    def execute(self) -> list[Resource]:
        return self.repository.find_all()
