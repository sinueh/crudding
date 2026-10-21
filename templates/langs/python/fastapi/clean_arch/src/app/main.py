from fastapi import FastAPI
from src.app.frameworks_and_drivers.web.resource_controller import router as resource_router

app = FastAPI(title="python-fastapi-clean-arch")

app.include_router(resource_router, prefix="/resource", tags=["Resource"])

if __name__ == "__main__":
    import uvicorn
    uvicorn.run("src.app.main:app", host="0.0.0.0", port=8000, reload=True)