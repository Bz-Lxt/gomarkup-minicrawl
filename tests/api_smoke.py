import os
import time

import pytest
import requests

API = os.environ.get("API_URL", "http://127.0.0.1:18421")


def test_health():
    r = requests.get(f"{API}/api/health", timeout=5)
    assert r.status_code == 200
    body = r.json()
    assert body["status"] == "ok"
    assert "time" in body


def test_search_requires_q():
    r = requests.get(f"{API}/api/search", timeout=5)
    assert r.status_code == 400
    assert r.json()["error"]["code"] == "VALIDATION_ERROR"


def test_crawl_search_graph_mock():
    stats = requests.get(f"{API}/api/stats", timeout=5).json()
    if stats.get("index_docs", 0) < 20:
        r = requests.post(
            f"{API}/api/crawl/tasks",
            json={"use_fixture": True, "workers": 8, "max_depth": 6, "max_pages": 80, "global_rps": 0},
            timeout=5,
        )
        if r.status_code == 409:
            tasks = requests.get(f"{API}/api/crawl/tasks", timeout=5).json()["tasks"]
            tid = next(t["id"] for t in tasks if t["status"] in ("running", "paused", "pending"))
        else:
            assert r.status_code == 201, r.text
            tid = r.json()["id"]

        deadline = time.time() + 30
        status = ""
        t = {}
        while time.time() < deadline:
            t = requests.get(f"{API}/api/crawl/tasks/{tid}", timeout=5).json()
            status = t["status"]
            if status in ("completed", "failed", "stopped"):
                break
            time.sleep(0.3)
        assert status == "completed", status
        assert t["crawled"] >= 20

    s = requests.get(f"{API}/api/search", params={"q": "minicrawl", "highlight": "true"}, timeout=5)
    assert s.status_code == 200
    hits = s.json()["hits"]
    assert hits
    assert "<mark>" in hits[0]["snippet"]

    g = requests.get(f"{API}/api/graph", timeout=5).json()
    assert len(g["nodes"]) >= 20
    assert len(g["edges"]) >= 1

    k = requests.get(f"{API}/api/keywords/top", timeout=5).json()
    assert k["keywords"]

    stats = requests.get(f"{API}/api/stats", timeout=5).json()
    assert stats["index_docs"] >= 20
