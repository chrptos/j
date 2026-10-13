#!/usr/bin/env python3
"""Summarize k6 JSON points; keep ramp and response-drain separate from measurement."""
import json
import math
import sys
from collections import Counter, defaultdict
from pathlib import Path


def percentile(values, p):
    if not values:
        return None
    ordered = sorted(values)
    index = (len(ordered) - 1) * p
    low = math.floor(index)
    return ordered[low] + (ordered[math.ceil(index)] - ordered[low]) * (index - low)


def summarize(prefix):
    config = json.loads(Path(f"{prefix}.config.json").read_text())
    counts = Counter()
    durations = defaultdict(list)
    phase_counts = Counter()
    dropped = 0
    successful_decrements = 0
    http_requests = 0
    invalid_checks = 0
    with Path(f"{prefix}.json").open() as source:
        for line in source:
            point = json.loads(line)
            if point.get("type") != "Point":
                continue
            data = point["data"]
            tags = data.get("tags", {})
            metric = point["metric"]
            value = data["value"]
            if metric == "dropped_iterations":
                dropped += int(value)
            elif metric == "http_reqs":
                http_requests += int(value)
            elif metric == "checks" and value == 0:
                invalid_checks += 1
            elif metric == "load_requests":
                phase_counts[tags["phase"]] += int(value)
                if tags["api"] == "decrement" and tags["status"] == "200":
                    successful_decrements += int(value)
                if tags["phase"] == "measure":
                    counts[(tags["api"], tags["outcome"], tags["status"], tags["code"])] += int(value)
            elif metric == "load_duration_ms" and tags["phase"] == "measure":
                durations[(tags["api"], tags["outcome"])].append(value)
    measured = sum(counts.values())
    successes = sum(n for (_, outcome, _, _), n in counts.items() if outcome == "success")
    failures = sum(n for (_, outcome, _, _), n in counts.items() if outcome == "failure")
    business = sum(n for (_, outcome, _, _), n in counts.items() if outcome == "business")
    timeouts = sum(n for (_, _, _, code), n in counts.items() if code == "TIMEOUT")
    all_durations = [v for values in durations.values() for v in values]
    latency = [dict(api=api, outcome=outcome, count=len(values), p95=percentile(values, .95), p99=percentile(values, .99)) for (api, outcome), values in sorted(durations.items())]
    for api in sorted({key[0] for key in durations}):
        values = [v for (a, _), group in durations.items() if a == api for v in group]
        latency.append(dict(api=api, outcome="all", count=len(values), p95=percentile(values, .95), p99=percentile(values, .99)))
    for outcome in sorted({key[1] for key in durations}):
        values = [v for (_, o), group in durations.items() if o == outcome for v in group]
        latency.append(dict(api="all", outcome=outcome, count=len(values), p95=percentile(values, .95), p99=percentile(values, .99)))
    latency.append(dict(api="all", outcome="all", count=len(all_durations), p95=percentile(all_durations, .95), p99=percentile(all_durations, .99)))
    return {
        "config": config,
        "http_requests_all_phases": http_requests,
        "responses_by_phase": dict(phase_counts),
        "measurement_responses": measured,
        "success_rps": successes / config["hold"],
        "failure_rate": failures / measured if measured else None,
        "business_rejection_rate": business / measured if measured else None,
        "timeout_rate": timeouts / measured if measured else None,
        "dropped_iterations_all_phases": dropped,
        "invalid_checks": invalid_checks,
        "completed_requests_match": http_requests == sum(phase_counts.values()),
        "successful_decrements_all_phases": successful_decrements,
        "http_and_code_counts": [dict(api=api, outcome=outcome, status=status, code=code, count=n) for (api, outcome, status, code), n in sorted(counts.items())],
        "latency_ms": latency,
    }


if __name__ == "__main__":
    prefix = sys.argv[1]
    result = summarize(prefix)
    Path(f"{prefix}.report.json").write_text(json.dumps(result, ensure_ascii=False, indent=2) + "\n")
    print(json.dumps(result, ensure_ascii=False, indent=2))
    sys.exit(0 if result["completed_requests_match"] and result["invalid_checks"] == 0 and result["dropped_iterations_all_phases"] == 0 and result["http_requests_all_phases"] > 0 and (result["config"]["phase"] == "warmup" or result["measurement_responses"] > 0) else 1)
