# Superagent-Base E2E Test Report

**Generated**: 2026-05-12 18:15:10
**Environment**: localhost:8888 (backend) + localhost:8000 (LLM)
**Model**: Qwen3-Coder-Next-4bit

---

## Summary

| Metric | Value |
|--------|-------|
| Total Tests | 52 |
| Passed | 47 |
| Failed | 0 |
| Skipped | 5 |
| Pass Rate | 90.4% |

**pytest summary**: 47 passed, 5 skipped

---

## Test Results by Category

### Health Check [PASS]

- [x] test_01_health.py::TestHealthCheck::test_llm_service_reachable
- [x] test_01_health.py::TestHealthCheck::test_llm_chat_completion
- [x] test_01_health.py::TestHealthCheck::test_backend_service_reachable
- [x] test_01_health.py::TestHealthCheck::test_agents_loaded
- [x] test_01_health.py::TestHealthCheck::test_e2e_test_agents_loaded

### Basic Agent [PASS]

- [x] test_02_basic_agent.py::TestBasicAgent::test_single_message_response
- [x] test_02_basic_agent.py::TestBasicAgent::test_streaming_produces_multiple_tokens
- [x] test_02_basic_agent.py::TestBasicAgent::test_multi_turn_conversation
- [x] test_02_basic_agent.py::TestBasicAgent::test_long_message_handling
- [x] test_02_basic_agent.py::TestBasicAgent::test_empty_like_message

### Workflow Agent [PASS]

- [x] test_03_workflow.py::TestWorkflowAgent::test_workflow_executes_and_returns_result
- [x] test_03_workflow.py::TestWorkflowAgent::test_workflow_processes_different_topics
- [x] test_03_workflow.py::TestWorkflowAgent::test_workflow_performance

### Multi-Agent [PASS]

- [x] test_04_multi_agent.py::TestSupervisorAgent::test_supervisor_responds
- [x] test_04_multi_agent.py::TestSupervisorAgent::test_supervisor_delegation
- [x] test_04_multi_agent.py::TestSequentialAgent::test_sequential_pipeline_completes
- [x] test_04_multi_agent.py::TestSequentialAgent::test_sequential_handles_complex_input
- [x] test_04_multi_agent.py::TestParallelAgent::test_parallel_execution_completes
- [x] test_04_multi_agent.py::TestParallelAgent::test_parallel_performance_reasonable

### Management [PASS]

- [x] test_05_management.py::TestAgentListing::test_list_agents_returns_200
- [x] test_05_management.py::TestAgentListing::test_list_agents_contains_expected_fields
- [x] test_05_management.py::TestAgentListing::test_list_includes_all_types
- [x] test_05_management.py::TestHotReload::test_add_new_agent_yaml
- [x] test_05_management.py::TestHotReload::test_remove_agent_yaml
- [x] test_05_management.py::TestHotReload::test_modify_agent_yaml
- [x] test_05_management.py::TestAgentValidation::test_invalid_agent_not_loaded
- [x] test_05_management.py::TestAgentValidation::test_chat_with_nonexistent_agent_returns_error

### Interrupt/Resume [PASS]

- [x] test_06_interrupt.py::TestInterruptResume::test_interrupt_agent_loaded
- [x] test_06_interrupt.py::TestInterruptResume::test_interrupt_agent_chat
- [x] test_06_interrupt.py::TestInterruptResume::test_resume_without_interrupt_returns_error
- [ ] test_06_interrupt.py::TestInterruptResume::test_get_interrupt_state_no_pending _(skipped)_

---

## Test Evidence (Screenshots)

### 20260512_181416_llm_models

```
{"object":"list","data":[{"id":"Qwen3-Coder-Next-4bit","object":"model","created":1778580856,"owned_by":"omlx"},{"id":"Qwen3.6-27B-4bit","object":"model","created":1778580856,"owned_by":"omlx"}]}
```

### 20260512_181417_backend_agents_list

```
{"agents":[{"description":"","name":"research-workflow"},{"description":"You are a safety-aware assistant. When the user asks you to perform\npotentially dangerous operations (delete files, send emails, make purchases),\nyou must first ask for explicit confirmation before proceeding.\nAlways say \"Please confirm: do you want me to proceed with [action]?\"\nand wait for the user's response before taking any action.\n","name":"approval-agent"},{"description":"Sequential multi-agent pipeline for e2
```

### 20260512_181417_basic_single_message

```
Question: What is 2 + 2?
Response (329ms, 7 tokens):
2 + 2 = 4.
```

### 20260512_181417_llm_chat_completion

```
 {"id":"chatcmpl-c5461d8f","object":"chat.completion","created":1778580857,"model":"Qwen3-Coder-Next-4bit","choices":[{"index":0,"message":{"role":"assistant","content":"Hello!","reasoning_content":null,"tool_calls":null},"finish_reason":"stop"}],"usage":{"prompt_tokens":14,"completion_tokens":2,"total_tokens":16,"input_tokens":14,"output_tokens":2,"prompt_tokens_details":{"cached_tokens":0,"audio_tokens":null},"model_load_duration":null,"time_to_first_token":null,"total_time":null,"prompt_eval_
```

### 20260512_181417_loaded_agents

```
Agents: ['research-workflow', 'approval-agent', 'e2e-sequential', 'e2e-supervisor', 'research-agent', 'project-manager', 'e2e-basic', 'e2e-parallel', 'e2e-interrupt', 'e2e-workflow']
```

### 20260512_181418_basic_multi_turn

```
Turn 1: My name is Alice. Remember this.
Response 1: Got it, Alice! 😊

Turn 2: What is my name?
Response 2: Alice!
```

### 20260512_181418_basic_streaming_tokens

```
Token count: 50
First 5 tokens: ['Python', ' is', ' a', ' high', '-level']
Full response: Python is a high-level, interpreted programming language known for its simplicity, readability, and versatility. It supports multiple programming paradigms—including procedural, object-oriented, and functional—making it suitable for web development, data science, automation, and more.
```

### 20260512_181423_workflow_basic_execution

```
Input: artificial intelligence
Pipeline: research -> summarize
Elapsed: 3053ms
Output:
AI simulates human intelligence via algorithms, big data, and computing, using machine learning and deep learning for task improvement; it transforms industries like healthcare and finance but raises ethical issues like bias and job displacement.
```

### 20260512_181429_workflow_different_topics

```
Topic 1: quantum computing
Output 1: Qubits leverage superposition and entanglement for massive parallelism and correlated operations, enabling exponential (Shor) or quadratic (Grover) speedups for specific problems—though large-scale qu

Topic 2: ocean biology
Output 2: Phytoplankton drive marine food webs and produce ~50% of Earth’s oxygen; zooplankton transfer this energy upward; symbiosis enables ecosystems like coral reefs and deep-sea vents.
```

### 20260512_181433_supervisor_basic

```
Agent: e2e-supervisor
Question: What are the benefits of exercise?
Elapsed: 743ms
Response:
Exercise improves physical health (e.g., heart, weight), boosts mental health (reduces anxiety/depression), enhances sleep, increases energy, and supports cognitive function.
```

### 20260512_181433_workflow_performance

```
Nodes: 2 (research -> summarize)
Total time: 3498ms
Avg per node: 1749ms
```

### 20260512_181434_sequential_basic

```
Agent: e2e-sequential
Pipeline: e2e-basic (single step)
Input: Explain what a database is.
Elapsed: 827ms
Output:
A database is an organized collection of structured data, stored electronically in a computer system. It allows efficient data retrieval, management, and updates, typically using a Database Management System (DBMS) like MySQL or PostgreSQL.
```

### 20260512_181434_supervisor_delegation

```
Delegation test
Question: What is the capital of France?
Response: Paris
```

### 20260512_181436_sequential_complex

```
Complex input test
Response length: 458 chars
Response: Key microservices patterns include:  - **API Gateway** (single entry point for clients)  - **Service Discovery** (dynamic service lookup)  - **Circuit Breaker** (prevent cascading failures)  - **Event-Driven Architecture** (asynchronous communication via events)  - **Sidecar** (auxiliary container f
```

### 20260512_181437_parallel_basic

```
Agent: e2e-parallel
Sub-agents: e2e-basic (parallel)
Input: What is cloud computing?
Elapsed: 1001ms
Output:

--- e2e-basic ---
Cloud computing delivers on-demand IT resources—like servers, storage, databases, and software—over the internet, eliminating the need for on-premises hardware. It enables scalability, cost efficiency, and remote access to services via providers like AWS, Azure, or Google Cloud.
```

### 20260512_181438_management_agent_fields

```
Sample agent structure: {'description': "You are a helpful research assistant. You answer questions clearly, concisely,\nand accurately. When you don't know something, you say so.\n", 'name': 'research-agent'}
```

### 20260512_181438_management_all_types

```
Expected agents: ['e2e-basic', 'e2e-workflow']
Found: ['e2e-basic', 'e2e-workflow']
All agents: ['e2e-basic', 'e2e-parallel', 'e2e-interrupt', 'e2e-workflow', 'research-workflow', 'approval-agent', 'e2e-sequential', 'e2e-supervisor', 'research-agent', 'project-manager']
```

### 20260512_181438_management_list_200

```
Status: 200
Agent count: 10
Agents: ['e2e-supervisor', 'research-agent', 'project-manager', 'e2e-basic', 'e2e-parallel', 'e2e-interrupt', 'e2e-workflow', 'research-workflow', 'approval-agent', 'e2e-sequential']
```

### 20260512_181438_parallel_performance

```
Performance: 736ms
Content: 
--- e2e-basic ---
REST API is an architectural style for web services using HTTP methods (GET, POST, PUT, DELETE) to interact with resources, following stateless, cacheable, and uniform interface pri
```

### 20260512_181441_management_hotreload_add

```
Added: e2e-hotreload-add
Path: /Users/ailabuser1/Desktop/gitcode/superagent-base/backend/configs/agents/e2e-hotreload-add.yaml
Detected: YES
Current agents: ['research-agent', 'project-manager', 'e2e-basic', 'e2e-parallel', 'e2e-interrupt', 'e2e-workflow', 'research-workflow', 'approval-agent', 'e2e-sequential', 'e2e-supervisor', 'e2e-hotreload-add']
```


---

*Report generated by `generate_report.py`*
