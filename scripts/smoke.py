"""Create, enable, observe a successful callback, query and disable a timer."""
import argparse
import json
import threading
import time
import urllib.request
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--base-url", default="http://127.0.0.1:8082")
    parser.add_argument("--callback-url", help="Existing callback endpoint; otherwise start a local receiver")
    parser.add_argument("--timeout", type=float, default=150)
    args = parser.parse_args()
    receiver = None
    if not args.callback_url:
        class Handler(BaseHTTPRequestHandler):
            def do_POST(self):
                self.rfile.read(int(self.headers.get("Content-Length", "0")))
                self.send_response(200)
                self.end_headers()
                self.wfile.write(b'{"received":true}')

            def log_message(self, *_):
                pass

        receiver = ThreadingHTTPServer(("127.0.0.1", 0), Handler)
        threading.Thread(target=receiver.serve_forever, daemon=True).start()
        args.callback_url = f"http://127.0.0.1:{receiver.server_port}/callback"

    def api(path, payload=None):
        data = None if payload is None else json.dumps(payload).encode()
        request = urllib.request.Request(args.base_url.rstrip("/") + path, data=data,
                                         headers={"Content-Type": "application/json"})
        with urllib.request.urlopen(request, timeout=10) as response:
            result = json.load(response)
        if "code" in result and result["code"] != 0:
            raise RuntimeError(result)
        return result

    timer_id = None
    try:
        api("/readyz")
        created = api("/xtimer/createTimer", {
            "app": "smoke", "name": f"smoke-{time.time_ns()}", "cron": "@every 2s",
            "notify_http_param": {"method": "POST", "url": args.callback_url,
                                  "header": {"Content-Type": "application/json"}, "body": '{"hello":"scheduler"}'},
        })
        timer_id = created["data"]["timer_id"]
        if timer_id <= 0:
            raise RuntimeError("create returned no timer ID")
        api("/xtimer/enableTimer", {"app": "smoke", "timer_id": timer_id, "status": 1})
        if api(f"/xtimer/getTimer?timer_id={timer_id}")["data"]["status"] != 1:
            raise RuntimeError("timer was not enabled")
        deadline = time.monotonic() + args.timeout
        tasks = []
        while time.monotonic() < deadline:
            tasks = api(f"/xtimer/getTasks?timer_id={timer_id}&limit=20")["data"]
            if any(task["status"] == 1 and '"received":true' in task["output"] for task in tasks):
                print(f"PASS: timer {timer_id} created, enabled, callback succeeded and execution history persisted")
                break
            time.sleep(1)
        else:
            raise RuntimeError(f"no successful callback within {args.timeout}s; recent tasks: {tasks}")
    finally:
        if timer_id is not None:
            api("/xtimer/enableTimer", {"app": "smoke", "timer_id": timer_id, "status": 0})
            if api(f"/xtimer/getTimer?timer_id={timer_id}")["data"]["status"] != 0:
                raise RuntimeError("timer was not disabled")
            print(f"PASS: timer {timer_id} disabled")
        if receiver is not None:
            receiver.shutdown()
            receiver.server_close()


if __name__ == "__main__":
    main()
