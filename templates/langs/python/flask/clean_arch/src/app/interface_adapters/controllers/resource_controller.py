from src.app.application.use_cases.create_resource import CreateResource
from src.app.application.use_cases.delete_resource import DeleteResource
from src.app.application.use_cases.get_resource import GetResource
from src.app.application.use_cases.list_resource import ListResource
from src.app.application.use_cases.update_resource import UpdateResource
from src.app.interface_adapters.persistence.resource_repository import ResourceRepositoryAdapter
from src.app.interface_adapters.presenters.resource_presenter import present_resource


class ResourceController:
    def __init__(self):
        repository = ResourceRepositoryAdapter()
        self.create_uc = CreateResource(repository)
        self.get_uc = GetResource(repository)
        self.list_uc = ListResource(repository)
        self.update_uc = UpdateResource(repository)
        self.delete_uc = DeleteResource(repository)

    def list_all(self) -> list[dict]:
        return [present_resource(item) for item in self.list_uc.execute()]

    def get(self, item_id: str) -> dict | None:
        item = self.get_uc.execute(item_id)
        return present_resource(item) if item is not None else None

    def create(self, payload: dict) -> dict:
        return present_resource(self.create_uc.execute(payload))

    def update(self, item_id: str, payload: dict) -> dict:
        return present_resource(self.update_uc.execute(item_id, payload))

    def delete(self, item_id: str) -> None:
        self.delete_uc.execute(item_id)


_controller = ResourceController()

list_all = _controller.list_all
get = _controller.get
create = _controller.create
update = _controller.update
delete = _controller.delete
