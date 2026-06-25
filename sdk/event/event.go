package event

type EventType string

const (
	TextDelta    EventType = "text_delta"
	ToolCall     EventType = "tool_call"
	ToolResult   EventType = "tool_result"
	Thinking     EventType = "thinking"
	Error        EventType = "error"
	Done         EventType = "done"
	Progress     EventType = "progress"
	AgentSwitch  EventType = "agent_switch"
)

type Event struct {
	Type    EventType `json:"type"`
	Content string    `json:"content,omitempty"`
	Delta   string    `json:"delta,omitempty"`
	Name    string    `json:"name,omitempty"`
	Error   string    `json:"error,omitempty"`
}
