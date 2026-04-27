#!/usr/bin/env python3
import argparse
import json
import random
from pathlib import Path

import yaml


def load_yaml(path: str):
    with open(path, "r", encoding="utf-8") as f:
        return yaml.safe_load(f)


def weighted_choice(rng, d):
    keys = list(d.keys())
    ws = [float(d[k]) for k in keys]
    s = sum(ws)
    if s <= 0:
        raise ValueError("权重总和必须 > 0")
    ws = [x / s for x in ws]
    return rng.choices(keys, weights=ws, k=1)[0]


def rand_in_range(rng, arr2):
    lo, hi = int(arr2[0]), int(arr2[1])
    if lo > hi:
        lo, hi = hi, lo
    return rng.randint(lo, hi)


def normalize_target(t):
    return {
        "target_type": t.get("target_type", "").strip(),
        "target_name": t.get("target_name", "").strip(),
        "target_namespace": t.get("target_namespace", "").strip(),
    }


def pick_target(rng, cfg):
    strategy = cfg.get("target_strategy", "fixed_pod")

    if strategy == "fixed_pod":
        t = normalize_target(cfg["fixed_pod"])
        if t["t != "pod" or not t["target_name"] or not t["target_namespace"]:
            raise ValueError("fixed_pod 必须提供 target_type=pod + target_name + target_namespace")
        return t

    if strategy == "random_pod_pool":
        pool = [normalize_target(x) for x in cfg.get("pod_pool", [])]
        if not pool:
            raise ValueError("pod_pool 为空")
        t = rng.choice(pool)
        if t["target_type"] != "pod" or not t["target_name"] or not t["target_namespace"]:
            raise ValueError("pod_pool 项必须是 pod 且包含 name/namespace")
        return t

    if strategy == "random_node":
        pool = [normalize_target(x) for x in cfg.get("node_pool", [])]
        if pool:
            t = rng.choice(pool)
        else:
            t = {"target_type": "node", "target_name": "", "target_namespace": ""}
        if t["target_type"] != "node":
            raise ValueError("node_pool 项必须是 node")
        return t

    if strategy == "mixed":
        mode = weighted_choice(rng, cfg.g 0.5, "node": 0.5}))
        if mode == "pod":
            pool = [normalize_target(x) for x in cfg.get("pod_pool", [])]
            if not pool:
                raise ValueError("mixed 模式下 pod_pool 为空")
            t = rng.choice(pool)
            if t["target_type"] != "pod" or not t["target_name"] or not t["target_namespace"]:
                raise ValueError("pod_pool 项必须是 pod 且包含 name/namespace")
            return t
        else:
            pool = [normalize_target(x) for x in cfg.get("node_pool", [])]
            if pool:
                t = rng.choice(pool)
            else:
                t = {"target_type": "node", "target_name": "", "target_namespace": ""}
            if t["target_type"] != "node":
                raise ValueError("node_pool 项必须是 node")
            return t

    raise ValueError(f"未知 target_strategy: {strategy}")


def build_body(rng, cfg):
    t = pick_target(rng, cfg)
    exp = weighted_choice(rng, cfg["experiment_weights"])
    duration = rand_in_range(rng, cfg["duration_range_sec"])

    body = {
        "target_type": t["target_type"],
        "experiment_type": exp,
        "duration": duration,
    }

    # pod 必须带 name/namespace
    if t["target_type"] == "pod":
        body["target_name"] = t["target_name"]
        body["target_namespace"] = t["target_namespace"]
    else:
        # node 可选带 name
        if t.get("target_name"):
            body["target_name"] = t["target_name"]

    if exp == "cpu-load":
        body["cpu_load_percent"] = rand_in_range(rng, cfg["cpu_load_percent_range"])
    elif exp == "mem-load":
        body["memory_load_percent"] = rand_in_range(rng, cfg["memory_load_percent_range"])
    else:
        raise ValueError(f"仅支持 cpu-load/mem-load，收到: {exp}")

    return body


def generate_schedule(cfg):
    rng = random.Random(int(cfg["seed"]))

    run_seconds = int(cfg["run_seconds"])
    first_after = int(cfg["first_inject_after_sec"])
    min_gap = int(cfg["min_gap_sec"])
    max_gap = int(p:
        raise ValueError("min_gap_sec/max_gap_sec 非法")

    events = []
    t = first_after
    idx = 0
    while t < run_seconds:
        events.append({
            "event_id": f"ev_{idx:03d}",
            "at_sec": t,
            "body": build_body(rng, cfg),
        })
        idx += 1
        t += rng.randint(min_gap, max_gap)

    return events


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--config", required=True)
    ap.add_argument("--out", required=True)
    args = ap.parse_args()

    cfg = load_yaml(args.config)
    events = generate_schedule(cfg)

    out = {
        "run_name": cfg.get("run_name", "exp"),
        "seed": cfg["seed"],
        "base_url": cfg["base_url"].rstrip("/"),
        "inject_endpoint": cfg["inject_endpoint"],
        "stop_scheduler_endpoint": cfg.get("stop_scheduler_endpoint", ""),
        "request_timeout_sec": int(cfg.get("request_timeout_sec", 15)),
        "max_retries": int(cfg.get("max_retries", 2)),
        "retry_backoff_sec": floag.get("retry_backoff_sec", 1.5)),
        "run_seconds": int(cfg["run_seconds"]),
        "target_strategy": cfg.get("target_strategy", ""),
        "events": events,
    }

    Path(args.out).parent.mkdir(parents=True, exist_ok=True)
    with open(args.out, "w", encoding="utf-8") as f:
        json.dump(out, f, ensure_ascii=False, indent=2)

    print(f"[OK] schedule -> {args.out}, events={len(events)}")


if __name__ == "__main__":
    main()
