"""
Evaluation for A2A Agent Team using Pydantic Evals + Phoenix.

Evaluates agent responses for correctness, completeness, and quality.
Results are uploaded to Phoenix for visualization.

Usage:
    # Ensure agents are running and Phoenix is accessible
    # kubectl port-forward svc/phoenix-svc -n phoenix 6006:6006
    # cd a2a-agent && ./bin/a2a-agent &

    python evaluate_team.py
"""

import json
import os
import time

import requests
from opentelemetry import trace
from opentelemetry.exporter.otlp.proto.http.trace_exporter import OTLPSpanExporter
from opentelemetry.sdk.resources import Resource
from opentelemetry.sdk.trace import TracerProvider
from opentelemetry.sdk.trace.export import BatchSpanProcessor

PHOENIX_URL = os.getenv("PHOENIX_URL", "http://localhost:6006")
PHOENIX_COLLECTOR = os.getenv("PHOENIX_COLLECTOR", f"{PHOENIX_URL}/v1/traces")
HEALTH_AGENT_URL = os.getenv("HEALTH_AGENT_URL", "http://localhost:9090")


def setup_tracing():
    resource = Resource.create({"service.name": "a2a-team-evaluator"})
    provider = TracerProvider(resource=resource)
    exporter = OTLPSpanExporter(endpoint=PHOENIX_COLLECTOR)
    provider.add_span_processor(BatchSpanProcessor(exporter))
    trace.set_tracer_provider(provider)
    return trace.get_tracer("a2a-evaluator")


def send_a2a(url, text):
    """Send A2A message and return response text."""
    payload = {
        "jsonrpc": "2.0",
        "method": "SendMessage",
        "params": {
            "message": {
                "messageId": f"eval-{int(time.time()*1000)}",
                "role": "user",
                "parts": [{"text": text}],
            }
        },
        "id": 1,
    }
    resp = requests.post(url, json=payload, timeout=60)
    result = resp.json()
    r = result.get("result", {})
    task = r.get("task", r)
    for artifact in task.get("artifacts", []):
        for part in artifact.get("parts", []):
            if part.get("text"):
                return part["text"]
    return ""


# Evaluation test cases
TEST_CASES = [
    {
        "name": "node_status",
        "input": "Show cluster node status",
        "expected_keywords": ["Ready", "mcp-gitops"],
        "description": "Should return node names and Ready status",
    },
    {
        "name": "pod_status_kagent",
        "input": "Show pods in kagent namespace",
        "expected_keywords": ["kagent", "Running", "ready"],
        "description": "Should list pods in kagent namespace with Running status",
    },
    {
        "name": "pod_status_default",
        "input": "Show pods in default namespace",
        "expected_keywords": ["No pods found", "default"],
        "description": "Should report no pods in default namespace",
    },
    {
        "name": "events_check",
        "input": "Show warning events",
        "expected_keywords": ["event", "healthy", "Warning"],
        "description": "Should return events or indicate cluster is healthy",
    },
    {
        "name": "deployment_status",
        "input": "Show deployments in kube-system namespace",
        "expected_keywords": ["kube-system", "ready", "coredns"],
        "description": "Should list kube-system deployments including coredns",
    },
    {
        "name": "cluster_summary",
        "input": "Give me a full cluster health summary",
        "expected_keywords": ["Cluster", "Nodes", "Pods"],
        "description": "Should return comprehensive summary with nodes, pods, events",
    },
]


def evaluate_response(test_case, response):
    """Score a response against expected keywords."""
    if not response:
        return {"score": 0.0, "reason": "Empty response"}

    response_lower = response.lower()
    matched = []
    missing = []

    for kw in test_case["expected_keywords"]:
        if kw.lower() in response_lower:
            matched.append(kw)
        else:
            missing.append(kw)

    # Score: percentage of keywords found (at least one match = partial credit)
    score = len(matched) / len(test_case["expected_keywords"]) if test_case["expected_keywords"] else 1.0

    reason = f"Matched {len(matched)}/{len(test_case['expected_keywords'])} keywords"
    if missing:
        reason += f". Missing: {missing}"

    return {"score": round(score, 2), "reason": reason}


def main():
    tracer = setup_tracing()

    print("=== A2A Agent Team Evaluation ===\n")
    print(f"Agent: {HEALTH_AGENT_URL}")
    print(f"Phoenix: {PHOENIX_URL}")
    print(f"Test cases: {len(TEST_CASES)}\n")

    results = []
    total_score = 0

    for tc in TEST_CASES:
        with tracer.start_as_current_span(
            f"eval.{tc['name']}",
            attributes={
                "eval.test_case": tc["name"],
                "eval.input": tc["input"],
                "eval.description": tc["description"],
            },
        ) as span:
            print(f"Test: {tc['name']}")
            print(f"  Input: {tc['input']}")

            start = time.time()
            response = send_a2a(HEALTH_AGENT_URL, tc["input"])
            duration = time.time() - start

            evaluation = evaluate_response(tc, response)
            total_score += evaluation["score"]

            span.set_attribute("eval.score", evaluation["score"])
            span.set_attribute("eval.reason", evaluation["reason"])
            span.set_attribute("eval.duration_ms", int(duration * 1000))
            span.set_attribute("eval.response_length", len(response))
            span.set_attribute("eval.response_preview", response[:200] if response else "")

            status = "PASS" if evaluation["score"] >= 0.5 else "FAIL"
            print(f"  Score: {evaluation['score']} ({status})")
            print(f"  Reason: {evaluation['reason']}")
            print(f"  Duration: {int(duration*1000)}ms")
            print()

            results.append({
                "test_case": tc["name"],
                "score": evaluation["score"],
                "status": status,
                "reason": evaluation["reason"],
                "duration_ms": int(duration * 1000),
            })

    # Summary
    avg_score = total_score / len(TEST_CASES) if TEST_CASES else 0
    passed = sum(1 for r in results if r["status"] == "PASS")
    failed = len(results) - passed

    print("=" * 50)
    print(f"Results: {passed} passed, {failed} failed")
    print(f"Average score: {avg_score:.2f}")
    print(f"\nCheck Phoenix UI for detailed traces: {PHOENIX_URL}")

    # Flush traces
    trace.get_tracer_provider().force_flush()
    print("Traces flushed to Phoenix.")


if __name__ == "__main__":
    main()
