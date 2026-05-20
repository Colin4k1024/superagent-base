"""Basic chat example — send a message and print the response."""

import asyncio

from superagent import SuperagentClient


async def main() -> None:
    async with SuperagentClient(base_url="http://localhost:8888") as client:
        response = await client.chat("research-agent", "What is quantum computing?")
        print(response)


if __name__ == "__main__":
    asyncio.run(main())
