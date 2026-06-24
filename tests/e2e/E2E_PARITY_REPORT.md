# E2E Test Report — Three-Base Parity Verification

**Date**: 2026-06-24 22:41:24

## Summary

| Base | Passed | Failed | Total | Pass Rate |
|------|--------|--------|-------|-----------|
| go | 25 | 0 | 25 | ✅ 100.0% |
| python | 25 | 0 | 25 | ✅ 100.0% |
| java | 25 | 0 | 25 | ✅ 100.0% |

## Detailed Results

| Endpoint | Go | Python | Java | Status |
|----------|-----|--------|------|--------|
| `health` | ✅ | ✅ | ✅ | ✅ |
| `ready` | ✅ | ✅ | ✅ | ✅ |
| `metrics` | ✅ | ✅ | ✅ | ✅ |
| `list_agents` | ✅ | ✅ | ✅ | ✅ |
| `list_conversations` | ✅ | ✅ | ✅ | ✅ |
| `create_conversation` | ✅ | ✅ | ✅ | ✅ |
| `get_agent_state` | ✅ | ✅ | ✅ | ✅ |
| `set_agent_state` | ✅ | ✅ | ✅ | ✅ |
| `get_agent_state_key` | ✅ | ✅ | ✅ | ✅ |
| `get_session_messages` | ✅ | ✅ | ✅ | ✅ |
| `list_files` | ✅ | ✅ | ✅ | ✅ |
| `list_memory` | ✅ | ✅ | ✅ | ✅ |
| `add_memory` | ✅ | ✅ | ✅ | ✅ |
| `search_memory` | ✅ | ✅ | ✅ | ✅ |
| `list_workflows` | ✅ | ✅ | ✅ | ✅ |
| `list_skills` | ✅ | ✅ | ✅ | ✅ |
| `search_skills` | ✅ | ✅ | ✅ | ✅ |
| `list_tools` | ✅ | ✅ | ✅ | ✅ |
| `list_mcp_servers` | ✅ | ✅ | ✅ | ✅ |
| `admin_status` | ✅ | ✅ | ✅ | ✅ |
| `admin_reload` | ✅ | ✅ | ✅ | ✅ |
| `get_me` | ✅ | ✅ | ✅ | ✅ |
| `chat_stream` | ✅ | ✅ | ✅ | ✅ |
| `get_interrupt_state` | ✅ | ✅ | ✅ | ✅ |
| `chat_abort` | ✅ | ✅ | ✅ | ✅ |

## Parity Analysis

✅ **All endpoints have consistent behavior across all three bases.**

## Response Format Comparison

| Endpoint | Go Response | Python Response | Java Response | Match |
|----------|-------------|-----------------|---------------|-------|
| `health` | {'status'} | {'status', 'uptime_seconds', 'version', 'agents_loaded'} | {'timestamp', 'status', 'service'} | ❌ |
| `ready` | {'status'} | {'data', 'msg', 'code'} | {'checks', 'status', 'uptime_seconds'} | ❌ |
| `metrics` | set() | set() | {'message', 'status'} | ❌ |
| `list_agents` | {'agents'} | set() | {'count', 'agents'} | ❌ |
| `list_conversations` | {'msg', 'code'} | set() | {'data', 'msg', 'code'} | ❌ |
| `get_agent_state` | {'agent_id', 'state'} | {'data', 'msg', 'code'} | {'data', 'msg', 'code'} | ❌ |
| `set_agent_state` | {'key', 'agent_id', 'status'} | {'data', 'msg', 'code'} | {'data', 'msg', 'code'} | ❌ |
| `get_agent_state_key` | {'key', 'agent_id', 'value'} | {'data', 'msg', 'code'} | {'data', 'msg', 'code'} | ❌ |
| `get_session_messages` | {'messages', 'count', 'session_id'} | {'data', 'msg', 'code'} | {'data', 'msg', 'code'} | ❌ |
| `list_files` | {'count', 'files'} | {'data', 'msg', 'code'} | {'data', 'msg', 'code'} | ❌ |
| `list_memory` | {'error'} | {'data', 'msg', 'code'} | {'path', 'timestamp', 'requestId', 'error', 'status'} | ❌ |
| `add_memory` | {'id'} | {'data', 'msg', 'code'} | {'data', 'msg', 'code'} | ❌ |
| `search_memory` | {'query', 'user_id', 'count', 'results'} | {'data', 'msg', 'code'} | {'data', 'msg', 'code'} | ❌ |
| `list_workflows` | {'msg', 'code'} | {'data', 'msg', 'code'} | {'data', 'msg', 'code'} | ❌ |
| `list_skills` | {'skills'} | {'data', 'msg', 'code'} | {'data', 'msg', 'code'} | ❌ |
| `search_skills` | {'skills'} | {'data', 'msg', 'code'} | {'data', 'msg', 'code'} | ❌ |
| `list_tools` | {'tools', 'count'} | {'data', 'msg', 'code'} | {'data', 'msg', 'code'} | ❌ |
| `list_mcp_servers` | {'msg', 'code'} | {'data', 'msg', 'code'} | {'servers', 'count'} | ❌ |
| `admin_status` | {'ready', 'uptime_seconds', 'agents', 'start_time', 'readiness_checks', 'last_reload_at', 'health', 'agent_count'} | {'data', 'msg', 'code'} | {'agents_loaded', 'memory_used_mb', 'processors', 'mcp_servers', 'version', 'memory_max_mb', 'status', 'runtime'} | ❌ |
| `admin_reload` | {'message', 'agents', 'agent_count'} | {'status', 'agents_loaded'} | {'agents_reloaded', 'status', 'total_agents'} | ❌ |
| `get_me` | {'user'} | {'data', 'msg', 'code'} | {'data', 'msg', 'code'} | ❌ |
| `get_interrupt_state` | {'error'} | {'data', 'msg', 'code'} | {'status', 'session_id', 'interrupted'} | ❌ |
| `chat_abort` | {'aborted'} | {'data', 'msg', 'code'} | {'data', 'msg', 'code'} | ❌ |