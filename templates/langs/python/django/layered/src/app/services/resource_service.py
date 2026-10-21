
from dataclasses import asdict
from src.app.models.resource import Resource


def list_all() -> list[dict]:
    # TODO: busque todos os registros e retorne como lista de dicionários
    raise NotImplementedError


def get(item_id: str) -> dict | None:
    # TODO: busque o registro por id
    raise NotImplementedError


def create(payload: dict) -> dict:
    # TODO: crie o registro a partir do payload
    raise NotImplementedError


def update(item_id: str, payload: dict) -> dict:
    # TODO: atualize o registro a partir do payload
    raise NotImplementedError


def delete(item_id: str) -> None:
    # TODO: remova o registro por id
    raise NotImplementedError
