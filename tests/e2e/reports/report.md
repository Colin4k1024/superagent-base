# Superagent-Base E2E Test Report

**Generated**: 2026-05-12 10:12:39
**Environment**: localhost:8888 (backend) + localhost:8000 (LLM)
**Model**: Qwen3-Coder-Next-4bit

---

## Summary

| Metric | Value |
|--------|-------|
| Total Tests | 31 |
| Passed | 30 |
| Failed | 0 |
| Skipped | 1 |
| Pass Rate | 96.8% |

**pytest summary**: 30 passed, 1 skipped

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

### 20260512_101146_llm_models

```
{"object":"list","data":[{"id":"Qwen3-Coder-Next-4bit","object":"model","created":1778551906,"owned_by":"omlx"},{"id":"Qwen3.6-27B-4bit","object":"model","created":1778551906,"owned_by":"omlx"}]}
```

### 20260512_101147_backend_agents_list

```
{"agents":[{"description":"You are a helpful test assistant. Keep responses concise (under 50 words).\nAlways respond in English.\n","name":"e2e-basic"},{"description":"You are a helpful research assistant. You answer questions clearly, concisely,\nand accurately. When you don't know something, you say so.\n","name":"research-agent"},{"description":"You are a project manager. You coordinate sub-agents to complete complex tasks.\nWhen the user asks something, decide which sub-agent is best suited
```

### 20260512_101147_basic_single_message

```
Question: What is 2 + 2?
Response (359ms, 7 tokens):
2 + 2 = 4.
```

### 20260512_101147_llm_chat_completion

```
 {"id":"chatcmpl-cb492311","object":"chat.completion","created":1778551907,"model":"Qwen3-Coder-Next-4bit","choices":[{"index":0,"message":{"role":"assistant","content":"Hello!","reasoning_content":null,"tool_calls":null},"finish_reason":"stop"}],"usage":{"prompt_tokens":14,"completion_tokens":2,"total_tokens":16,"input_tokens":14,"output_tokens":2,"prompt_tokens_details":{"cached_tokens":0,"audio_tokens":null},"model_load_duration":null,"time_to_first_token":null,"total_time":null,"prompt_eval_
```

### 20260512_101147_loaded_agents

```
Agents: ['e2e-basic', 'research-agent', 'project-manager', 'e2e-sequential', 'e2e-interrupt', 'e2e-supervisor', 'research-workflow', 'approval-agent', 'e2e-workflow', 'e2e-parallel']
```

### 20260512_101148_basic_streaming_tokens

```
Token count: 34
First 5 tokens: ['Python', ' is', ' a', ' high', '-level']
Full response: Python is a high-level, interpreted programming language known for its readability and simplicity. It supports multiple programming paradigms, including procedural, object-oriented, and functional programming.
```

### 20260512_101149_basic_multi_turn

```
Turn 1: My name is Alice. Remember this.
Response 1: Got it, Alice! I’ll remember that. How can I help you today?

Turn 2: What is my name?
Response 2: Alice.
```

### 20260512_101154_workflow_basic_execution

```
Input: artificial intelligence
Pipeline: research -> summarize
Elapsed: 3447ms
Output:
AI simulates human intelligence via subfields like machine learning and computer vision, enabled by big data, powerful computing, and deep learning, driving transformative applications across industries while raising ethical concerns about bias, privacy, and employment.
```

### 20260512_101200_workflow_different_topics

```
Topic 1: quantum computing
Output 1: Qubits leverage superposition and entanglement to enable exponential speedups for specific problems like factoring and search, revolutionizing computation beyond classical limits.

Topic 2: ocean biology
Output 2: Phytoplankton produce ~50% of Earth’s oxygen and anchor marine food webs; zooplankton transfer their energy upward; deep-sea species survive extreme conditions via bioluminescence, gigantism, and low 
```

### 20260512_101203_workflow_performance

```
Nodes: 2 (research -> summarize)
Total time: 3065ms
Avg per node: 1532ms
```

### 20260512_101204_supervisor_basic

```
Agent: e2e-supervisor
Question: What are the benefits of exercise?
Elapsed: 886ms
Response:
- e2e-basic: Exercise improves physical health (e.g., cardiovascular fitness, strength), boosts mental health (reduces anxiety, depression), enhances sleep, and supports weight management.
```

### 20260512_101205_supervisor_delegation

```
Delegation test
Question: What is the capital of France?
Response: Paris.
```

### 20260512_101206_sequential_basic

```
Agent: e2e-sequential
Pipeline: e2e-basic (single step)
Input: Explain what a database is.
Elapsed: 989ms
Output:
A database is an organized collection of structured data, stored and accessed electronically. It enables efficient data management, retrieval, and updating through systems like databases management systems (DBMS), supporting applications such as websites, banking, and analytics.
```

### 20260512_101208_sequential_complex

```
Complex input test
Response length: 495 chars
Response: Key microservices patterns include:  1. **Service Decomposition** (by business capability or domain)  2. **API Gateway** (unified entry point)  3. **Database per Service** (data isolation)  4. **Event-Driven Architecture** (asynchronous messaging)  5. **Circuit Breaker** (fault tolerance)  6. **Serv
```

### 20260512_101209_parallel_basic

```
Agent: e2e-parallel
Sub-agents: e2e-basic (parallel)
Input: What is cloud computing?
Elapsed: 934ms
Output:

--- e2e-basic ---
Cloud computing delivers on-demand computing services—like storage, servers, databases, networking, and software—over the internet (“the cloud”) instead of using local servers or personal devices. It enables scalable, flexible, and cost-effective access to resources.
```

### 20260512_101210_management_agent_fields

```
Sample agent structure: {'description': 'Sequential multi-agent pipeline for e2e testing', 'name': 'e2e-sequential'}
```

### 20260512_101210_management_all_types

```
Expected agents: ['e2e-basic', 'e2e-workflow']
Found: ['e2e-basic', 'e2e-workflow']
All agents: ['project-manager', 'e2e-sequential', 'e2e-interrupt', 'e2e-supervisor', 'research-workflow', 'approval-agent', 'e2e-workflow', 'e2e-parallel', 'e2e-basic', 'research-agent']
```

### 20260512_101210_management_list_200

```
Status: 200
Agent count: 10
Agents: ['e2e-sequential', 'e2e-interrupt', 'e2e-supervisor', 'research-workflow', 'approval-agent', 'e2e-workflow', 'e2e-parallel', 'e2e-basic', 'research-agent', 'project-manager']
```

### 20260512_101210_parallel_performance

```
Performance: 1086ms
Content: 
--- e2e-basic ---
A REST API (Representational State Transfer Application Programming Interface) is a web service design style that uses HTTP methods (GET, POST, PUT, DELETE) to perform operations on
```

### 20260512_101213_management_hotreload_add

```
Added: e2e-hotreload-add
Path: /Users/ailabuser1/Desktop/gitcode/superagent-base/backend/configs/agents/e2e-hotreload-add.yaml
Detected: YES
Current agents: ['e2e-basic', 'research-agent', 'project-manager', 'e2e-sequential', 'e2e-interrupt', 'e2e-supervisor', 'research-workflow', 'approval-agent', 'e2e-workflow', 'e2e-parallel', 'e2e-hotreload-add']
```


---

*Report generated by `generate_report.py`*
