"""Streaming chat example — iterate events as they arrive."""

import asyncio

from superagent import SuperagentClient, A2UIEventType


async def main() -> None:
    async with SuperagentClient(base_url="http://localhost:8888") as client:
        print("Streaming response:\n")

        stream = client.chat_stream(
            agent_id="research-agent",
            message="Explain the double-slit experiment",
            session_id="demo-session",
        )

        # Register a callback for tool_call events
        stream.on(A2UIEventType.tool_call, lambda e: print(f"\n[tool] {e.data.get('name')}"))

        async for event in stream:
            if event.event_type == A2UIEventType.text:
                print(event.text_delta, end="", flush=True)
            elif event.event_type == A2UIEventType.thinking:
                delta = event.data.get("delta", "")
                if delta:
                    print(f"\n[thinking] {delta}", end="", flush=True)
            elif event.event_type == A2UIEventType.interrupt:
                print(f"\n[interrupt] {event.data.get('reason')}")
                # In a real app you would collect user input here and call:
                # await client.resume(agent_id, session_id, user_input)
                break
            elif event.event_type == A2UIEventType.error:
                print(f"\n[error] {event.data.get('message')}")
                break
            elif event.event_type == A2UIEventType.done:
                print("\n\n[done]")

        print()


async def collect_example() -> None:
    """Alternatively, collect the entire response at once."""
    async with SuperagentClient(base_url="http://localhost:8888") as client:
        stream = client.chat_stream("research-agent", "What is the Higgs boson?")
        full_text = await stream.collect()
        print(full_text)


if __name__ == "__main__":
    asyncio.run(main())
