import os
# crudding:db-imports


class Config:
    PORT: int = int(os.getenv("PORT", "8000"))
    ENV: str = os.getenv("APP_ENV", "development")
    # crudding:db-config


config = Config()

# crudding:db-methods

