import json
import urllib.request
import urllib.error
import subprocess
import time
import sys

BASE_URL = "http://localhost:3081"

def http_req(path, method="GET", body=None, token=None):
    url = f"{BASE_URL}{path}"
    headers = {}
    if token:
        headers["Authorization"] = f"Bearer {token}"
    data = None
    if body is not None:
        headers["Content-Type"] = "application/json"
        data = json.dumps(body).encode("utf-8")
    
    req = urllib.request.Request(url, data=data, headers=headers, method=method)
    try:
        with urllib.request.urlopen(req, timeout=10) as resp:
            content_type = resp.headers.get("Content-Type", "")
            raw = resp.read()
            if "application/json" in content_type:
                return resp.status, json.loads(raw.decode("utf-8")), raw
            else:
                return resp.status, raw.decode("utf-8", errors="replace"), raw
    except urllib.error.HTTPError as e:
        raw = e.read()
        try:
            parsed = json.loads(raw.decode("utf-8"))
        except Exception:
            parsed = raw.decode("utf-8", errors="replace")
        return e.code, parsed, raw

def run():
    results = {}
    print("--- 1. Health & Setup Status ---")
    status, body, _ = http_req("/api/v1/health")
    print(f"GET /api/v1/health -> {status}: {body}")
    assert status == 200, f"Health check failed: {status}"
    results["health"] = {"status": status, "body": body}

    status, body, _ = http_req("/api/v1/auth/setup-status")
    print(f"GET /api/v1/auth/setup-status -> {status}: {body}")
    assert status == 200, f"Setup status failed: {status}"
    results["setup_status"] = {"status": status, "body": body}
    needs_setup = body.get("needs_setup", False)

    # Verify unauthenticated API access is strictly blocked with 401 Unauthorized
    unauth_status, _, _ = http_req("/api/v1/channels")
    print(f"GET /api/v1/channels without token -> {unauth_status} (Expected 401 Unauthorized)")
    assert unauth_status == 401, f"Expected 401 Unauthorized without token, got: {unauth_status}"

    print("\n--- 2. Auth Setup / Login ---")
    if needs_setup:
        print("Database requires first-launch setup. Testing /api/v1/auth/setup...")
        setup_payload = {
            "username": "admin",
            "password": "admin123",
            "full_name": "Root Administrator",
            "email": "admin@mcrflow.tv"
        }
        status, body, _ = http_req("/api/v1/auth/setup", method="POST", body=setup_payload)
        print(f"POST /api/v1/auth/setup -> {status}: token present={('token' in body)}")
        assert status == 200 and "token" in body, f"Setup failed: {status} {body}"
        token = body["token"]
        user = body["user"]

        # Crucial security test: Assert repeated /api/v1/auth/setup is strictly rejected with 403 Forbidden!
        repeat_status, repeat_body, _ = http_req("/api/v1/auth/setup", method="POST", body=setup_payload)
        print(f"POST /api/v1/auth/setup (repeated attempt) -> {repeat_status} (Expected 403 Forbidden)")
        assert repeat_status == 403, f"Repeated setup should be forbidden, got: {repeat_status}"
    else:
        status, body, _ = http_req("/api/v1/auth/login", method="POST", body={"username": "admin", "password": "admin123"})
        print(f"POST /api/v1/auth/login -> {status}: token present={('token' in body)}")
        assert status == 200 and "token" in body, f"Login failed: {status} {body}"
        token = body["token"]
        user = body["user"]

        # Also verify /api/v1/auth/setup is strictly forbidden when users exist
        setup_status, _, _ = http_req("/api/v1/auth/setup", method="POST", body={"username": "hacker", "password": "pwd"})
        assert setup_status == 403, f"Setup should be forbidden when users exist, got: {setup_status}"

    results["login"] = {"token_obtained": True, "user": user}

    print("\n--- 3. Channels CRUD & Status ---")
    # List channels
    status, body, _ = http_req("/api/v1/channels", token=token)
    print(f"GET /api/v1/channels -> {status}, count={len(body)}")
    assert status == 200 and len(body) > 0, "Failed to list channels"
    results["channels_list"] = {"status": status, "count": len(body), "sample": body[0]}

    # Get single channel
    status, body, _ = http_req("/api/v1/channels/ch-01", token=token)
    print(f"GET /api/v1/channels/ch-01 -> {status}, name={body.get('name')}")
    assert status == 200 and body.get("id") == "ch-01", "Failed to get channel ch-01"
    channel_ch01 = body
    results["channel_get"] = {"status": status, "channel": channel_ch01}

    # Create temporary channel
    new_ch = {
        "id": "ch-qa-temp",
        "name": "QA Test Channel",
        "call_sign": "QA-TEST",
        "resolution_id": "res-in-1080i50",
        "logo_path": "/logos/test.png",
        "ad_template_id": "tmpl-news-standard",
        "primary_agent_id": "agent-local-01",
        "destinations": [
            {"type": "udp", "enabled": True, "url": "udp://239.255.0.99", "port": 5000}
        ],
        "is_active": True
    }
    status, body, _ = http_req("/api/v1/channels", method="POST", body=new_ch, token=token)
    print(f"POST /api/v1/channels -> {status}, created id={body.get('id') if isinstance(body, dict) else body}")
    assert status == 201, f"Failed to create channel: {status} {body}"

    # Update temporary channel
    new_ch["name"] = "QA Test Channel Updated"
    status, body, _ = http_req("/api/v1/channels/ch-qa-temp", method="PUT", body=new_ch, token=token)
    print(f"PUT /api/v1/channels/ch-qa-temp -> {status}, updated name={body.get('name') if isinstance(body, dict) else body}")
    assert status == 200, f"Failed to update channel: {status}"

    # Delete temporary channel
    status, body, _ = http_req("/api/v1/channels/ch-qa-temp", method="DELETE", token=token)
    print(f"DELETE /api/v1/channels/ch-qa-temp -> {status}: {body}")
    assert status == 200, f"Failed to delete channel: {status}"
    results["channel_crud"] = "PASSED"

    print("\n--- 4. Playout Controls & FFmpeg Command ---")
    # Playout start
    status, body, _ = http_req("/api/v1/channels/ch-01/playout/start", method="POST", token=token)
    print(f"POST /api/v1/channels/ch-01/playout/start -> {status}: {body}")
    assert status == 200, f"Failed to start playout: {status} {body}"
    results["playout_start"] = {"status": status, "body": body}

    # Playout status
    status, body, _ = http_req("/api/v1/channels/ch-01/playout/status", token=token)
    print(f"GET /api/v1/channels/ch-01/playout/status -> {status}: {body}")
    assert status == 200, f"Failed to get playout status: {status}"
    results["playout_status"] = {"status": status, "body": body}

    # FFmpeg command generation
    status, body, _ = http_req("/api/v1/channels/ch-01/ffmpeg-cmd", token=token)
    print(f"GET /api/v1/channels/ch-01/ffmpeg-cmd -> {status}: {body.get('command')}")
    assert status == 200 and "ffmpeg" in body.get("command", ""), f"Failed to get ffmpeg cmd: {status}"
    results["ffmpeg_cmd"] = {"status": status, "command": body.get("command")}

    print("\n--- 5. Resolutions & Ad Templates ---")
    status, body, _ = http_req("/api/v1/resolutions", token=token)
    print(f"GET /api/v1/resolutions -> {status}, count={len(body)}")
    assert status == 200 and len(body) > 0, f"Failed to get resolutions: {status}"
    results["resolutions"] = {"status": status, "count": len(body), "resolutions": [r.get("name") for r in body]}

    status, body, _ = http_req("/api/v1/ad-templates", token=token)
    print(f"GET /api/v1/ad-templates -> {status}, count={len(body)}")
    assert status == 200 and len(body) > 0, f"Failed to get ad templates: {status}"
    results["ad_templates"] = {"status": status, "count": len(body)}

    print("\n--- 6. Media Browsing & Probing ---")
    status, body, _ = http_req("/api/v1/storage/browse", token=token)
    print(f"GET /api/v1/storage/browse -> {status}, files={len(body) if isinstance(body, list) else body}")
    assert status == 200, f"Failed to browse storage: {status}"
    results["storage_browse"] = {"status": status}

    print("\n--- 7. Agents & Bots ---")
    status, body, _ = http_req("/api/v1/agents", token=token)
    print(f"GET /api/v1/agents -> {status}, count={len(body)}")
    assert status == 200 and len(body) > 0, f"Failed to get agents: {status}"
    results["agents"] = {"status": status, "count": len(body), "agents": [a.get("name") for a in body]}

    status, body, _ = http_req("/api/v1/bots", token=token)
    print(f"GET /api/v1/bots -> {status}, count={len(body)}")
    assert status == 200 and len(body) > 0, f"Failed to get bots: {status}"
    results["bots"] = {"status": status, "count": len(body), "bots": [b.get("name") for b in body]}

    # Test NLP bot command
    status, body, _ = http_req("/api/v1/bot/nlp-command", method="POST", body={"command": "status ch-01", "channel_id": "ch-01"}, token=token)
    print(f"POST /api/v1/bot/nlp-command -> {status}: {body}")
    assert status == 200 and "reply" in body, f"Failed NLP bot command: {status} {body}"
    results["bot_nlp"] = {"status": status, "reply": body.get("reply")}

    print("\n--- 8. Schedules CRUD ---")
    now_ts = int(time.time())
    sched_item = {
        "id": "qa-sched-01",
        "channel_id": "ch-01",
        "title": "Evening News Special",
        "description": "Live broadcast testing segment",
        "media_path": "/media/storage/sample.mp4",
        "start_time": "2026-10-10T20:00:00Z",
        "end_time": "2026-10-10T20:30:00Z",
        "duration_seconds": 1800,
        "item_type": "primary",
        "status": "scheduled",
        "is_live": False
    }
    status, body, _ = http_req("/api/v1/schedules", method="POST", body=sched_item, token=token)
    print(f"POST /api/v1/schedules -> {status}, created={body.get('id') if isinstance(body, dict) else body}")
    assert status == 201, f"Failed to create schedule: {status} {body}"

    status, body, _ = http_req("/api/v1/schedules/qa-sched-01", token=token)
    print(f"GET /api/v1/schedules/qa-sched-01 -> {status}, title={body.get('title') if isinstance(body, dict) else body}")
    assert status == 200, f"Failed to get schedule: {status}"

    sched_item["title"] = "Evening News Special (Updated)"
    status, body, _ = http_req("/api/v1/schedules/qa-sched-01", method="PUT", body=sched_item, token=token)
    print(f"PUT /api/v1/schedules/qa-sched-01 -> {status}")
    assert status == 200, f"Failed to update schedule: {status}"

    status, body, _ = http_req("/api/v1/schedules/qa-sched-01", method="DELETE", token=token)
    print(f"DELETE /api/v1/schedules/qa-sched-01 -> {status}")
    assert status == 200, f"Failed to delete schedule: {status}"
    results["schedules_crud"] = "PASSED"

    print("\n--- 8b. Schedule Conflict Checking API ---")
    conflict_check_payload = {
        "channel_id": "ch-01",
        "start_time": "2026-10-10T14:00:00Z",
        "duration_seconds": 7200
    }
    status, body, _ = http_req("/api/v1/schedule/check-conflicts", method="POST", body=conflict_check_payload, token=token)
    print(f"POST /api/v1/schedule/check-conflicts -> {status}: has_conflicts={body.get('has_conflicts')}")
    assert status == 200, f"Failed conflict check: {status}"
    results["conflict_check"] = body

    print("\n--- 8c. Edge Agent Test Connection & Strict Reachability ---")
    # Test valid connection endpoint
    test_conn_payload = {
        "ip_address": "127.0.0.1",
        "port": 3082,
        "token": "agt_sec_test_valid_reachability"
    }
    status, body, _ = http_req("/api/v1/agents/test-connection", method="POST", body=test_conn_payload, token=token)
    print(f"POST /api/v1/agents/test-connection -> {status}: reachable={body.get('reachable')}, latency={body.get('latency_ms')}ms")
    assert status == 200, f"Failed test-connection: {status}"

    # Test agent ping
    status, body, _ = http_req("/api/v1/agents/agent-local-01/ping", method="POST", token=token)
    print(f"POST /api/v1/agents/agent-local-01/ping -> {status}: reachable={body.get('reachable')}")
    assert status == 200, f"Failed ping agent: {status}"

    # Verify pairing rejection with invalid token format
    bad_pair_payload = {
        "hostname": "unreachable-agent",
        "ip_address": "192.0.2.1",
        "port": 9999,
        "token": "bad_token"
    }
    status, body, _ = http_req("/api/v1/agents/pair", method="POST", body=bad_pair_payload, token=token)
    print(f"POST /api/v1/agents/pair (bad token) -> {status} (Expected 400 rejection)")
    assert status == 400, f"Agent pairing should reject bad token: {status}"

    print("\n--- 9. EPG XMLTV Export ---")
    status, xmltv, _ = http_req("/epg/ch-01.xml?token=epg_sec_dd1_xml_2026")
    print(f"GET /epg/ch-01.xml -> {status}, xml snippet:\n{xmltv[:250]}...")
    assert status == 200 and "<tv" in xmltv, f"Failed to get EPG XMLTV: {status}"
    results["epg_xmltv"] = {"status": status, "length": len(xmltv)}

    print("\n--- 10. Web UI Serving ---")
    status, html, _ = http_req("/")
    print(f"GET / -> {status}, html snippet:\n{html[:200]}...")
    assert status == 200 and "<!DOCTYPE html>" in html, f"Failed to serve SPA root: {status}"
    results["ui_root"] = {"status": status, "length": len(html)}

    # Dynamically extract asset filenames from HTML
    import re
    js_match = re.search(r'src=["\'](/assets/index-[^"\']+\.js)["\']', html)
    css_match = re.search(r'href=["\'](/assets/index-[^"\']+\.css)["\']', html)
    js_path = js_match.group(1) if js_match else "/assets/index-Cd21Wl-u.js"
    css_path = css_match.group(1) if css_match else "/assets/index-DJjAR1XV.css"

    status, js, _ = http_req(js_path)
    print(f"GET {js_path} -> {status}, len={len(js)}")
    assert status == 200 and len(js) > 1000, f"Failed to serve JS bundle: {status}"
    results["ui_js"] = {"status": status, "length": len(js)}

    status, css, _ = http_req(css_path)
    print(f"GET {css_path} -> {status}, len={len(css)}")
    assert status == 200 and len(css) > 100, f"Failed to serve CSS bundle: {status}"
    results["ui_css"] = {"status": status, "length": len(css)}

    # Save summary results
    with open("qa_api_results.json", "w") as f:
        json.dump(results, f, indent=2)

    print("\nAPI & UI E2E verification successfully passed!")

if __name__ == "__main__":
    run()
