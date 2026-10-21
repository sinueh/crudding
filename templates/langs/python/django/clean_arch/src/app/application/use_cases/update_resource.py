from src.app.domain.model.resource import Resource


class UpdateResource:
    def __init__(self, repository):
        self.repository = repository

    def execute(self, item_id: str, data: dict) -> Resource:
        # TODO: carregue o Resource, aplique as alterações e persista com self.repository.save.
        raise NotImplementedError
