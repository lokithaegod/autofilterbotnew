#!/usr/bin/env python3
"""Reconcile UptimeRobot monitoring with AutoFilterMG's India quiet hours.

Render Free instances sleep after inactivity and wake on HTTP traffic.
Do not call the Render suspend/resume API: it returned HTTP 403 and left
monitoring paused on 2026-11-23. This script never changes Render settings.
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

BOT_URL = "https://autofilterbotnew-ioi4.onrender.com/"
UPTIMEROBOT_API = "https://api.uptimerobot.com/v2/"
TZ = ZoneInfo("Asia/Kolkata")
HTTP_TIMEOUT = 15


def desired_online(now: dt.datetime) -> bool:
    """Online from 07:00 IST through 00:59 IST, offline 01:00-06:59."""
    hour = now.astimezone(TZ).hour
    return not 1 <= hour < 7


def request_json(url: str, *, form: dict[str, str]) -> dict:
    body = urllib.parse.urlencode(form).encode()
    headers = {"Accept": "application/json",
               "Content-Type": "application/x-www-form-urlencoded"}
    req = urllib.request.Request(url, data=body, headers=headers, method="POST")
    try:
        with urllib.request.urlopen(req, timeout=HTTP_TIMEOUT) as response:
            payload = response.read(100_000)
            return json.loads(payload) if payload else {}
    except urllib.error.HTTPError as exc:
        raise RuntimeError(f"UptimeRobot API HTTP {exc.code}") from None


def uptime_robot_call(method: str, key: str, **fields: str) -> dict:
    result = request_json(UPTIMEROBOT_API + method,
                          form={"api_key": key, "format": "json", **fields})
    if result.get("stat") != "ok":
        raise RuntimeError(f"UptimeRobot {method} rejected the request")
    return result


def find_our_monitor(key: str) -> dict:
    """Only alter the monitor whose exact URL matches this bot."""
    results = []
    offset = 0
    while True:
        data = uptime_robot_call("getMonitors", key, limit="50", offset=str(offset))
        page = data.get("monitors", [])
        if not isinstance(page, list):
            raise RuntimeError("UptimeRobot returned an invalid monitor list")
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
        print("UptimeRobot monitor already enabled", flush=True)
        return
    if not online and status == 0:
        print("UptimeRobot monitor already paused", flush=True)
        return
    uptime_robot_call("editMonitor", key, id=str(monitor["id"]),
                      status="1" if online else "0")
    print("UptimeRobot monitor " + ("resumed" if online else "paused"), flush=True)


def health_ready() -> bool:
    """HTTP 200 means the bot's health handler can reach Telegram and MongoDB."""
    try:
        req = urllib.request.Request(BOT_URL + "healthz",
                                     headers={"User-Agent": "AutoFilterMG-schedule/2.0"})
        with urllib.request.urlopen(req, timeout=HTTP_TIMEOUT) as response:
            return (response.status == 200
                    and b"Telegram bot process is running" in response.read(1000))
    except (urllib.error.URLError, TimeoutError, OSError):
        return False


def reconcile(key: str, online: bool) -> None:
    monitor = find_our_monitor(key)
    if not online:
        # Stop monitoring; Render decides when an idle Free instance sleeps.
        set_monitor(key, monitor, False)
        print("Night quiet hours active; no Render API changes", flush=True)
        return

    # Critical: do not leave monitoring paused merely because the bot
    # is still cold-starting or health checks are failing.
    set_monitor(key, monitor, True)
    for attempt in range(8):
        if health_ready():
            print("Bot healthy; daytime monitoring enabled", flush=True)
            return
        if attempt < 7:
            print(f"Waiting for bot health ({attempt + 1}/8)", flush=True)
            time.sleep(15)
    raise RuntimeError("Bot is not healthy yet; monitoring is enabled and the next run will retry")


def self_test() -> None:
    # Boundary checks include midnight crossover and the entire quiet period.
    for hour, minute, expected in [
        (0, 59, True), (1, 0, False), (3, 0, False),
        (6, 59, False), (7, 0, True), (23, 59, True)
    ]:
        moment = dt.datetime(2026, 11, 23, hour, minute, tzinfo=TZ)
        assert desired_online(moment) is expected, (moment, expected)
    print("Schedule boundary self-test passed")


def main() -> int:
    if "--self-test" in sys.argv:
        self_test()
        return 0
    key = os.getenv("UPTIMEROBOT_API_KEY", "").strip()
    if not key:
        print("UPTIMEROBOT_API_KEY missing; no changes made", file=sys.stderr)
        return 1
    online = desired_online(dt.datetime.now(dt.timezone.utc))
    print("Desired state (Asia/Kolkata): " + ("ONLINE" if online else "OFFLINE"), flush=True)
    try:
        reconcile(key, online)
    except Exception as exc:
        print(f"Schedule reconciliation failed: {exc}", file=sys.stderr, flush=True)
        return 1
    return 0


if __name__ == "__main__":
    sys.exit(main())
