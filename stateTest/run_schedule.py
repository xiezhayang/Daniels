#!/usr/bin/env python3
import argparse
import csv
import json
import time
from datetime import datetime, timezone
from pathlib import Path

import requests


def now_iso():
    return datetime.now(timezone.utc).isoformat()


def req_retry(method, url, timeout_sec, max_retries, backoff_sec, **kwargs):
    last_err = None
    for i in range(max_retries + 1):
        try:
            r = requests.request(method, url, timeout=timeout_sec, **kwargs)
            return r, None
        except Exception as e:
            last_err = str(e)
            if i < max_retries:
                time.sleep(backoff_sec * (2 ** i))
    return None, last_err


def write_json(p, obj):
    with open(p, "w", encoding="utf-8") as f:
        json.dump(obj, f, ensure_ascii=False, indent=2)


def write_csv(p, rows):
    if not rows:
        return
    fields = list(rows[0].keys())
    with open(p, "w", newline="", encoding="utf-8") as f:
        w = csv.DictWriter(f, fieldnames=fields)
        w.writeheader()
        w.writerows(rows)


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--schedule", required=True)
    ap.add_argument("--out-dir", default="runs")
    ap.add_argument("--stop-scheduler", action="store_true")
    args = ap.parse_args()

    with open(args.schedule, "r", encoding="utf-8") as f:
        sch = json.load(f)

    run_id = datetime.now().strftime("%Y%m%d_%H%M%S")
    run_dir = Path(args.out_dir) / f"{sch.get('run_name','exp')}_{run_id}"
    run_dir.mkdir(parents=True, exist_ok=True)

    base = sch["base_url"].rstrip("/")
    inject_url = base + sch["inject_endpoint"]
    stop_url = base + sch.get("stop_scheduler_endpoint", "")

    timeout_sec = int(sch.get("request_timeout_sec", 15))
    max_retries = int(sch.get("max_retries", 2))
    backoff_sec = float(sch.get("retry_backoff_sec", 1.5))
    events = sorted(sch["events"], key=lambda x: int(x["at_sec"]))

    meta = {
        "run_id": run_id,
        "start_time_utc": now_iso(),
        "inject_url": inject_url,
        "event_count": len(events),
        "stop_scheduler_called": False,
    }

    if args.stop_scheduler and sch.get("stop_scheduler_endpoint"):
        r, err = req_retry("POST", stop_url, timeout_sec, max_retries, backoff_sec)
        meta["stop_scheduler_called"] = True
        meta["stop_scheduler_result"] = {
            "ok": err is None and r is not None,
            "status_code": (r.status_code if r else None),
            "error": err,
            "resp": (r.text[:500] if r else "")
        }

    rows = []
    t0_wall = time.time()
    t0_mono = time.monotonic()

    for ev in events:
        at_sec = int(ev["at_sec"])
        event_id = ev["event_id"]
        body = ev["body"]

        target_mono = t0_mono + at_sec
        sleep_s = target_mono - time.monotonic()
        if sleep_s > 0:
            time.sleep(sleep_s)

        planned_ts = datetime.fromtimestamp(t0_wall + at_sec, tz=timezone.utc).isoformat()
        send_wall = time.time()
        send_ts = datetime.fromtimestamp(send_wall, tz=timezone.utc).isoformat()
        lag_ms = int((send_wall - (t0_wall + at_sec)) * 1000)

        r, err = req_retry("POST", inject_url, timeout_sec, max_retries, backoff_sec, json=body)

        item = {
            "event_id": event_id,
            "at_sec": at_sec,
            "planned_ts_utc": planned_ts,
            "sent_ts_utc": send_ts,
            "lag_ms": lag_ms,
            "ok": False,
            "status_code": "",
            "body": json.dumps(body, ensure_ascii=False),
            "resp_text": "",
            "error": "",
        }

        if err:
            item["error"] = err
        else:
            item["status_code"] = r.status_code
            item["resp_text"] = r.text[:1000]
            item["ok"] = (200 <= r.status_code < 300)

        rows.append(item)

    meta["end_time_utc"] = now_iso()

    write_json(run_dir / "run_meta.json", meta)
    write_json(run_dir / "schedule_used.json", sch)
    write_json(run_dir / "dispatch_log.json", rows)
    write_csv(run_dir / "dispatch_log.csv", rows)

    ok_cnt = sum(1 for x in rows if x["ok"])
    print(f"[OK] run_dir={run_dir}")
    print(f"[STAT] total={len(rows)} success={ok_cnt} fail={len(rows)-ok_cnt}")


if __name__ == "__main__":
    main()
