#!/usr/bin/env python3
"""Reconcile AutoFilterMG's off-hours across Render and UptimeRobot.

The service remains untouched unless BOTH GitHub Actions secrets are set.
No credentials or response bodies containing tokens are printed.
"""
from __future__ import annotations

import datetime as dt
import json
import os
import sys
import time
import urllib.error
import urllib.parse
import urllib.request
from zoneinfo import ZoneInfo

RENDER_SERVICE_ID = "srv-dc0r1767bikc73bktnf0"
RENDER_ENDPOINT = f"https://api.render.com/v1/services/{RENDER_SERVICE_ID}"
BOT_URL = "https://autofilterbotnew-ioi4.onrender.com/"
UPTIMEROBOT_API = "https://api.uptimerobot.com/v2/"
TZ = ZoneInfo("Asia/Kolkata")
HTTP_TIMEOUT = 15


def desired_online(now: dt.datetime) -> bool:
    """07:00 <= local time < 01:00 following day (1 AM-7 AM closed)."""
    hour = now.astimezone(TZ).hour
    return not 1 <= hour < 7


def request_json(url: str, method: str = "GET", *, token: str = "",
                 form: dict[str, str] | None = None) -> dict:
    body = urllib.parse.urlencode(form).encode() if form is not None else None
    headers = {"Accept": "application/json"}
    if token:
        headers["Authorization"] = f"Bearer {token}"
    if body is not None:
        headers["Content-Type"] = "application/x-www-form-urlencoded"
    req = urllib.request.Request(url, data=body, headers=headers, method=method)
    try:
        with urllib.request.urlopen(req, timeout=HTTP_TIMEOUT) as response:
            payload = response.read(100_000)
            return json.loads(payload) if payload else {}
    except urllib.error.HTTPError as exc:
        # Do not dump response body (might include information from the API).
        raise RuntimeError(f"{method} request failed with HTTP {exc.code}") from None


def uptime_robot_call(method: str, key: str, **fields: str) -> dict:
    result = request_json(UPTIMEROBOT_API + method, "POST",
                          form={"api_key": key, "format": "json", **fields})
    if result.get("stat") != "ok":
        raise RuntimeError(f"UptimeRobot {method} rejected the request")
    return result


def find_our_monitor(key: str) -> dict:
    """Require exactly one match; never modify another UptimeRobot monitor."""
    results = []
    offset = 0
    while True:
        data = uptime_robot_call("getMonitors", key, limit="50", offset=str(offset))
        page = data.get("monitors", [])
        if not isinstance(page, list):
            raise RuntimeError("Invalid monitor list")
        results.extend(page)
        if len(page) < 50:
            break
        offset += 50
        if offset > 2000:
            raise RuntimeError("Monitor list exceeds safe traversal limit")
    matching = [m for m in results
                if str(m.get("url", "")).rstrip("/") == BOT_URL.rstrip("/")]
    if len(matching) != 1:
        raise RuntimeError(f"Expected exactly one AutoFilterMG monitor; found {len(matching)}")
    return matching[0]


def set_monitor(key: str, monitor: dict, online: bool) -> None:
    status = int(monitor.get("status", -1))
    if online and status != 0:
        print("UptimeRobot monitoring is already enabled")
        return
    if not online and status == 0:
        print("UptimeRobot monitoring is already paused")
        return
    uptime_robot_call("editMonitor", key, id=str(monitor["id"]),
                      status="1" if online else "0")
    print("UptimeRobot monitoring " + ("resumed" if online else "paused"))


def render_state(key: str) -> bool:
    data = request_json(RENDER_ENDPOINT, token=key)
    state = data.get("suspended")
    if state not in {"not_suspended", "suspended"}:
        raise RuntimeError("Unknown Render service suspension state")
    if data.get("id") != RENDER_SERVICE_ID:
        raise RuntimeError("Render returned an unexpected service")
    return state == "not_suspended"


def set_render(key: str, online: bool) -> None:
    if render_state(key) == online:
        print("Render already " + ("resumed" if online else "suspended"))
        return
    suffix = "resume" if online else "suspend"
    request_json(RENDER_ENDPOINT + "/" + suffix, "POST", token=key)
    print("Render " + suffix + " request accepted")


def health_ready() -> bool:
    try:
        req = urllib.request.Request(BOT_URL, headers={"User-Agent": "AutoFilterMG-schedule/1.0"})
        with urllib.request.urlopen(req, timeout=15) as response:
            return response.status == 200 and b"Telegram bot process is running" in response.read(1000)
    except (urllib.error.URLError, TimeoutError, OSError):
        return False


def reconcile(render_key: str, uptime_key: str, online: bool) -> None:
    # Resolve monitor before changing anything. Missing or duplicate monitors
    # cause a safe failure instead of touching the wrong UptimeRobot monitor.
    monitor = find_our_monitor(uptime_key)
    if online:
        # Resume the bot first. Keep UptimeRobot paused while it warms up so
        # planned startup doesn't create false downtime alerts.
        set_render(render_key, True)
        for attempt in range(6):
            if health_ready():
                set_monitor(uptime_key, monitor, True)
                print("Bot is ready and scheduled monitoring is enabled")
                return
            if attempt < 5:
                time.sleep(15)
        raise RuntimeError("Bot has not become healthy yet; next run will retry")
    else:
        # Pause ALL monitoring before suspending so HTTP checks cannot wake
        # the bot or generate expected downtime alerts.
        set_monitor(uptime_key, monitor, False)
        set_render(render_key, False)
        print("Night quiet hours are active")


def main() -> int:
    render_key = os.getenv("RENDER_API_KEY", "").strip()
    uptime_key = os.getenv("UPTIMEROBOT_API_KEY", "").strip()
    if not render_key or not uptime_key:
        print("Not activated: add both RENDER_API_KEY and UPTIMEROBOT_API_KEY "
              "as repository Actions secrets. No changes made.")
        return 0
    online = desired_online(dt.datetime.now(dt.timezone.utc))
    print("Desired state (Asia/Kolkata): " + ("ONLINE" if online else "OFFLINE"))
    try:
        reconcile(render_key, uptime_key, online)
    except Exception as exc:
        print(f"Schedule reconciliation failed: {exc}", file=sys.stderr)
        return 1
    return 0


if __name__ == "__main__":
    sys.exit(main())
