class DeleteResource:
    def __init__(self, repository):
        self.repository = repository

    def execute(self, item_id: str) -> None:
        self.repository.delete(item_id)
