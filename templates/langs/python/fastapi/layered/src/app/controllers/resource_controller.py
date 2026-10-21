from fastapi import APIRouter, HTTPException, status
import src.app.services.resource_service as service

router = APIRouter()


@router.get("/")
def list_all():
    try:
        return service.list_all()
    except NotImplementedError:
        raise HTTPException(
            status_code=status.HTTP_501_NOT_IMPLEMENTED,
            detail="Resource: not implemented",
        )


@router.get("/{item_id}")
def get_by_id(item_id: str):
    try:
        item = service.get(item_id)
        if item is None:
            raise HTTPException(
                status_code=status.HTTP_404_NOT_FOUND,
                detail=f"Resource {item_id} not found",
            )
        return item
    except NotImplementedError:
        raise HTTPException(
            status_code=status.HTTP_501_NOT_IMPLEMENTED,
            detail="Resource: not implemented",
        )


@router.post("/", status_code=status.HTTP_201_CREATED)
def create(payload: dict):
    try:
        return service.create(payload)
    except NotImplementedError:
        raise HTTPException(
            status_code=status.HTTP_501_NOT_IMPLEMENTED,
            detail="Resource: not implemented",
        )


@router.patch("/{item_id}")
def update(item_id: str, payload: dict):
    try:
        return service.update(item_id, payload)
    except NotImplementedError:
        raise HTTPException(
            status_code=status.HTTP_501_NOT_IMPLEMENTED,
            detail="Resource: not implemented",
        )


@router.delete("/{item_id}", status_code=status.HTTP_204_NO_CONTENT)
def delete(item_id: str):
    try:
        service.delete(item_id)
    except NotImplementedError:
        raise HTTPException(
            status_code=status.HTTP_501_NOT_IMPLEMENTED,
            detail="Resource: not implemented",
        )

