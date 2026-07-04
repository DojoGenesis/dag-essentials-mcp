"""Interactive stdio smoke test for dag-essentials-mcp.

Behaves like a real MCP client: holds stdin open and reads each response
before sending the next request. Run: python smoke_test.py
Exits non-zero if any check fails.
"""
import json
import subprocess
import sys

EXE = r"C:\Users\cruzr\dag-essentials-mcp\dag-essentials-mcp.exe"


def main() -> int:
    p = subprocess.Popen(
        [EXE],
        stdin=subprocess.PIPE,
        stdout=subprocess.PIPE,
        stderr=subprocess.DEVNULL,
        bufsize=0,
    )

    def send(obj):
        p.stdin.write((json.dumps(obj) + "\n").encode("utf-8"))
        p.stdin.flush()

    def recv():
        line = p.stdout.readline()
        if not line:
            raise RuntimeError("server closed stdout with no response")
        return json.loads(line.decode("utf-8"))

    send({"jsonrpc": "2.0", "id": 1, "method": "initialize",
          "params": {"protocolVersion": "2025-06-18", "capabilities": {},
                     "clientInfo": {"name": "smoke", "version": "0"}}})
    init = recv()
    send({"jsonrpc": "2.0", "method": "notifications/initialized"})

    send({"jsonrpc": "2.0", "id": 2, "method": "tools/list"})
    tools = recv()

    send({"jsonrpc": "2.0", "id": 3, "method": "tools/call",
          "params": {"name": "validate",
                     "arguments": {"edges": [["a", "b"], ["b", "c"], ["c", "a"]]}}})
    val = recv()

    send({"jsonrpc": "2.0", "id": 4, "method": "tools/call",
          "params": {"name": "width",
                     "arguments": {"nodes": ["a", "b", "c", "d"],
                                   "edges": [["a", "b"], ["a", "c"], ["b", "d"], ["c", "d"]]}}})
    wid = recv()

    send({"jsonrpc": "2.0", "id": 5, "method": "tools/call",
          "params": {"name": "frontier",
                     "arguments": {"nodes": ["a", "b", "c", "d"],
                                   "edges": [["a", "b"], ["a", "c"], ["b", "d"], ["c", "d"]],
                                   "completed": ["a"]}}})
    fro = recv()

    p.stdin.close()
    try:
        p.wait(timeout=5)
    except subprocess.TimeoutExpired:
        p.kill()

    names = sorted(t["name"] for t in tools["result"]["tools"])
    val_sc = val["result"].get("structuredContent", {})
    wid_sc = wid["result"].get("structuredContent", {})
    fro_sc = fro["result"].get("structuredContent", {})

    print("protocol      :", init["result"].get("protocolVersion"))
    print("server        :", init["result"].get("serverInfo", {}))
    print("tools (%d)     : %s" % (len(names), names))
    print("validate cycle:", val_sc)
    print("width diamond :", wid_sc)
    print("frontier {a}  :", fro_sc)

    ok = True
    ok &= len(names) == 9
    ok &= val_sc.get("is_dag") is False and len(val_sc.get("cycle") or []) >= 3
    ok &= wid_sc.get("width") == 2
    ok &= fro_sc.get("ready") == ["b", "c"]
    print("\nRESULT:", "PASS" if ok else "FAIL")
    return 0 if ok else 1


if __name__ == "__main__":
    sys.exit(main())
