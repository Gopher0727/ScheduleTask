"""Development callback receiver. Run: python3 examples/callback.py."""
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer


class Handler(BaseHTTPRequestHandler):
    def do_POST(self):
        body = self.rfile.read(int(self.headers.get("Content-Length", "0")))
        print(f"callback {self.path}: {body.decode(errors='replace')}", flush=True)
        response = b'{"received":true}'
        self.send_response(200)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(response)))
        self.end_headers()
        self.wfile.write(response)


if __name__ == "__main__":
    ThreadingHTTPServer(("0.0.0.0", 9090), Handler).serve_forever()
