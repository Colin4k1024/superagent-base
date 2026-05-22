# gRPC API

## 服务端口

默认监听 `:50051`，支持 gRPC reflection。

## 服务定义

### AgentService

```protobuf
service AgentService {
  rpc CreateAgent(CreateAgentRequest) returns (CreateAgentResponse);
  rpc GetAgent(GetAgentRequest) returns (GetAgentResponse);
  rpc ListAgents(ListAgentsRequest) returns (ListAgentsResponse);
  rpc UpdateAgent(UpdateAgentRequest) returns (UpdateAgentResponse);
  rpc DeleteAgent(DeleteAgentRequest) returns (DeleteAgentResponse);
  rpc LoadAgentFromYAML(LoadAgentFromYAMLRequest) returns (LoadAgentFromYAMLResponse);
}
```

### ConversationService

```protobuf
service ConversationService {
  rpc Chat(ChatRequest) returns (stream ChatResponse);      // 流式对话
  rpc CreateConversation(CreateConversationRequest) returns (CreateConversationResponse);
  rpc GetConversation(GetConversationRequest) returns (GetConversationResponse);
  rpc ListConversations(ListConversationsRequest) returns (ListConversationsResponse);
}
```

### ModelService

```protobuf
service ModelService {
  rpc ListModels(ListModelsRequest) returns (ListModelsResponse);
  rpc GetModel(GetModelRequest) returns (GetModelResponse);
  rpc CreateModel(CreateModelRequest) returns (CreateModelResponse);
  rpc TestModel(TestModelRequest) returns (TestModelResponse);
}
```

### ToolService

```protobuf
service ToolService {
  rpc ListTools(ListToolsRequest) returns (ListToolsResponse);
  rpc GetTool(GetToolRequest) returns (GetToolResponse);
  rpc InvokeTool(InvokeToolRequest) returns (InvokeToolResponse);
}
```

## 使用示例

```bash
# 列出服务
grpcurl -plaintext localhost:50051 list

# 列出 Agent
grpcurl -plaintext localhost:50051 superagent.agent.v1.AgentService/ListAgents

# 流式对话
grpcurl -plaintext -d '{
  "session_id": "s1",
  "message": "Hello",
  "extra": {"fields": {"agent_name": {"string_value": "research-agent"}}}
}' localhost:50051 superagent.conversation.v1.ConversationService/Chat
```

## Proto 文件位置

```
api/proto/
├── agent/v1/agent.proto
├── conversation/v1/conversation.proto
├── model/v1/model.proto
└── tool/v1/tool.proto
```
