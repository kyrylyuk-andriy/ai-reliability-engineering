"""
Tracing for A2A Agent Team with Phoenix.

Instruments A2A communication between agents and sends traces to Phoenix
for observability and evaluation.

Usage:
    # Start Phoenix (if not running in K8s):
    # phoenix serve

    # Or use K8s Phoenix:
    # kubectl port-forward svc/phoenix-svc -n phoenix 6006:6006

    # Start the A2A agents:
    # cd a2a-agent && ./bin/a2a-agent &
    # kubectl port-forward svc/kagent-controller -n kagent 8083:8083 &
    # cd a2a-team && ./bin/a2a-team &

    # Run tracing:
    python trace_team.py
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

# Phoenix endpoint
PHOENIX_URL = os.getenv("PHOENIX_URL", "http://localhost:6006")
PHOENIX_COLLECTOR = os.getenv("PHOENIX_COLLECTOR", f"{PHOENIX_URL}/v1/traces")

# A2A endpoints
TEAM_URL = os.getenv("TEAM_URL", "http://localhost:9092")
HEALTH_AGENT_URL = os.getenv("HEALTH_AGENT_URL", "http://localhost:9090")
SRE_COORDINATOR_URL = os.getenv("SRE_COORDINATOR_URL", "http://localhost:9091")


def setup_tracing():
    """Configure OpenTelemetry to send traces to Phoenix."""
    resource = Resource.create({"service.name": "a2a-team-tracer"})
    provider = TracerProvider(resource=resource)
    exporter = OTLPSpanExporter(endpoint=PHOENIX_COLLECTOR)
    provider.add_span_processor(BatchSpanProcessor(exporter))
    trace.set_tracer_provider(provider)
    return trace.get_tracer("a2a-team")


def send_a2a_message(tracer, agent_name, agent_url, message_text):
    """Send an A2A message and trace it."""
    with tracer.start_as_current_span(
        f"a2a.send_message.{agent_name}",
        attributes={
            "a2a.agent.name": agent_name,
            "a2a.agent.url": agent_url,
            "a2a.message.text": message_text,
            "a2a.protocol": "jsonrpc",
        },
    ) as span:
        # Discover agent card
        with tracer.start_as_current_span(f"a2a.discover.{agent_name}") as card_span:
            try:
                card_url = agent_url.rstrip("/") + "/.well-known/agent-card.json"
                card_resp = requests.get(card_url, timeout=5)
                if card_resp.status_code == 200:
                    card = card_resp.json()
                    card_span.set_attribute("a2a.agent.version", card.get("version", ""))
                    card_span.set_attribute(
                        "a2a.agent.skills_count",
                        len(card.get("skills", [])),
                    )
                else:
                    card_span.set_attribute("a2a.discovery.error", f"HTTP {card_resp.status_code}")
            except Exception as e:
                card_span.set_attribute("a2a.discovery.error", str(e))

        # Send message
        start = time.time()
        try:
            # Use SendMessage for official SDK agents, message/send for kagent
            method = "SendMessage"
            msg_payload = {
                "messageId": f"trace-{int(time.time()*1000)}",
                "role": "user",
                "parts": [{"text": message_text}],
            }

            payload = {
                "jsonrpc": "2.0",
                "method": method,
                "params": {"message": msg_payload},
                "id": 1,
            }

            resp = requests.post(
                agent_url,
                json=payload,
                headers={"Content-Type": "application/json"},
                timeout=60,
            )
            duration = time.time() - start

            span.set_attribute("a2a.response.status_code", resp.status_code)
            span.set_attribute("a2a.response.duration_ms", int(duration * 1000))

            result = resp.json()
            if "error" in result:
                span.set_attribute("a2a.response.error", result["error"].get("message", ""))
                return None

            # Extract response text
            response_text = extract_text(result)
            span.set_attribute("a2a.response.text_length", len(response_text))
            span.set_attribute(
                "a2a.response.preview", response_text[:200] if response_text else ""
            )
            return response_text

        except Exception as e:
            span.set_attribute("a2a.error", str(e))
            span.record_exception(e)
            return None


def extract_text(result):
    """Extract text from A2A response (handles both SDK formats)."""
    r = result.get("result", {})
    # Official SDK: result.task.artifacts[].parts[].text
    task = r.get("task", r)
    for artifact in task.get("artifacts", []):
        for part in artifact.get("parts", []):
            if part.get("text"):
                return part["text"]
    # Message format
    for part in r.get("parts", []):
        if part.get("text"):
            return part["text"]
    return ""


def run_team_assessment(tracer):
    """Run a full team assessment with tracing."""
    with tracer.start_as_current_span(
        "a2a.team.assessment",
        attributes={"a2a.team.type": "full_health_check"},
    ) as span:
        print("=== A2A Team Assessment (with Phoenix Tracing) ===\n")

        # 1. Health Agent (standalone)
        print("1. Querying K8s Health Agent...")
        health_result = send_a2a_message(
            tracer,
            "K8s Health Agent",
            HEALTH_AGENT_URL,
            "Show cluster node status",
        )
        if health_result:
            print(f"   Response: {health_result[:100]}...\n")

        # 2. Team Coordinator (queries all agents)
        print("2. Querying A2A Team Coordinator...")
        team_result = send_a2a_message(
            tracer,
            "A2A Team Coordinator",
            TEAM_URL,
            "Run a full health check on the cluster",
        )
        if team_result:
            print(f"   Response: {team_result[:200]}...\n")

        # 3. Direct health agent queries
        for query in [
            "Show pods in kagent namespace",
            "Show any warning events",
            "Show deployment status in flux-system",
        ]:
            print(f"3. Querying Health Agent: {query}")
            result = send_a2a_message(tracer, "K8s Health Agent", HEALTH_AGENT_URL, query)
            if result:
                print(f"   Response: {result[:100]}...\n")

        span.set_attribute("a2a.team.queries_total", 5)
        print("=== Assessment complete. Check Phoenix UI for traces. ===")
        print(f"    {PHOENIX_URL}")


def main():
    tracer = setup_tracing()
    run_team_assessment(tracer)
    # Flush traces
    trace.get_tracer_provider().force_flush()
    print("\nTraces flushed to Phoenix.")


if __name__ == "__main__":
    main()
