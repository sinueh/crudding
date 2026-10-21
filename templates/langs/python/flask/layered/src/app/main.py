from flask import Flask
from src.app.controllers.resource_controller import router as resource_router

app = Flask(__name__)

app.register_blueprint(resource_router, url_prefix="/resource")

if __name__ == "__main__":
    app.run(host="0.0.0.0", port=5000, debug=True)