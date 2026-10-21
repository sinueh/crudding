import json
from http.server import BaseHTTPRequestHandler, HTTPServer
from urllib.parse import urlsplit

from src.app.resource.controller import handle as handle_resource

ROUTES = {
    "resource": handle_resource,
}


class RouterHandler(BaseHTTPRequestHandler):
    def read_json(self):
        length = int(self.headers.get("Content-Length", 0))
        if not length:
            return {}
        try:
            return json.loads(self.rfile.read(length))
        except json.JSONDecodeError:
            return {}

    def send_json(self, status: int, data: any):
        payload = json.dumps(data).encode("utf-8")
        self.send_response(status)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(payload)))
        self.end_headers()
        self.wfile.write(payload)

    def _dispatch(self):
        path = urlsplit(self.path).path.strip("/")
        parts = path.split("/") if path else []
        if not parts or parts[0] not in ROUTES:
            self.send_json(404, {"detail": "not found"})
            return
        prefix = parts[0]
        sub_parts = parts[1:]
        ROUTES[prefix](self, self.command, sub_parts)

    do_GET = _dispatch
    do_POST = _dispatch
    do_PATCH = _dispatch
    do_DELETE = _dispatch


if __name__ == "__main__":
    print("Serving on http://127.0.0.1:8000")
    HTTPServer(("127.0.0.1", 8000), RouterHandler).serve_forever()