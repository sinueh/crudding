from src.app.domain.model.resource import Resource


class CreateResource:
    def __init__(self, repository):
        self.repository = repository

    def execute(self, data: dict) -> Resource:
        # TODO: valide os dados, monte o Resource e persista com self.repository.save.
        raise NotImplementedError
