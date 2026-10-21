from flask import Blueprint, jsonify, request
import src.app.interface_adapters.controllers.resource_controller as service

router = Blueprint("resource", __name__)


@router.get("/")
def list_all():
    try:
        return jsonify(service.list_all())
    except NotImplementedError:
        return jsonify({"detail": "Resource: not implemented"}), 501


@router.get("/<item_id>")
def get_by_id(item_id: str):
    try:
        item = service.get(item_id)
        if item is None:
            return jsonify({"detail": f"Resource {item_id} not found"}), 404
        return jsonify(item)
    except NotImplementedError:
        return jsonify({"detail": "Resource: not implemented"}), 501


@router.post("/")
def create():
    payload = request.get_json(silent=True) or {}
    try:
        return jsonify(service.create(payload)), 201
    except NotImplementedError:
        return jsonify({"detail": "Resource: not implemented"}), 501


@router.patch("/<item_id>")
def update(item_id: str):
    payload = request.get_json(silent=True) or {}
    try:
        return jsonify(service.update(item_id, payload))
    except NotImplementedError:
        return jsonify({"detail": "Resource: not implemented"}), 501


@router.delete("/<item_id>")
def delete(item_id: str):
    try:
        service.delete(item_id)
        return "", 204
    except NotImplementedError:
        return jsonify({"detail": "Resource: not implemented"}), 501

