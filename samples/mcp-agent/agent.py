"""LangGraph ReAct agent with MCP tool integration.

Connects to one or more MCP servers listed in config.mcp_server_urls using
the streamable-HTTP transport, loads all available tools, and builds a
LangGraph ReAct agent. Resources and prompts exposed by the servers are
also available through the MCP client context.

The MCP client is intentionally kept open for the lifetime of the process
so tools can stream responses without re-negotiating sessions on each call.
"""

from __future__ import annotations

import asyncio
import logging
from typing import Any

from langchain_core.messages import AIMessage
from langchain_mcp_adapters.client import MultiServerMCPClient
from langchain_openai import ChatOpenAI
from langgraph.prebuilt import create_react_agent

from config import Config

log = logging.getLogger("mcp-agent")

MODEL = "gpt-4o-mini"


def _build_llm(cfg: Config) -> ChatOpenAI:
    return ChatOpenAI(
        model=MODEL,
        temperature=0,
        base_url=cfg.openai_mapping_url,
        api_key="not-used",
        default_headers={
            "API-Key": cfg.openai_mapping_api_key,
            "Authorization": "",
        },
    )


def _build_server_configs(cfg: Config) -> dict[str, dict[str, Any]]:
    """Map each MCP server URL to a named streamable-HTTP connection."""
    return {
        f"mcp_server_{i}": {
            "url": url,
            "transport": "streamable_http",
        }
        for i, url in enumerate(cfg.mcp_server_urls)
    }


def _build_system_prompt(base_prompt: str, tools: list[Any]) -> str:
    """Append a tool inventory to the base system prompt so the model knows exactly what it has."""
    if not tools:
        return base_prompt
    lines = [base_prompt, "", "You have access to the following tools:"]
    for t in tools:
        desc = getattr(t, "description", "") or ""
        first_line = desc.splitlines()[0] if desc else ""
        lines.append(f"- {t.name}: {first_line}" if first_line else f"- {t.name}")
    return "\n".join(lines)


class MCPAgent:
    """Async-aware wrapper that owns the MCP client lifecycle."""

    def __init__(self, cfg: Config) -> None:
        self._cfg = cfg
        self._llm = _build_llm(cfg)
        self._client: MultiServerMCPClient | None = None
        self._agent: Any = None
        self._tools: list[Any] = []

    @property
    def tool_names(self) -> list[str]:
        return [t.name for t in self._tools]

    async def start(self) -> None:
        """Open MCP connections and build the agent. Call once at startup."""
        print(f"MCP server URLs: {self._cfg.mcp_server_urls}")
        server_configs = _build_server_configs(self._cfg)

        if not server_configs:
            log.warning("No MCP server URLs configured — agent will have no MCP tools")
            self._agent = create_react_agent(
                model=self._llm,
                tools=[],
                prompt=self._cfg.system_prompt,
            )
            return

        self._client = MultiServerMCPClient(server_configs)
        self._tools = await self._client.get_tools()
        log.info(
            "Loaded %d MCP tools from %d server(s): %s",
            len(self._tools),
            len(server_configs),
            [t.name for t in self._tools],
        )

        self._agent = create_react_agent(
            model=self._llm,
            tools=self._tools,
            prompt=_build_system_prompt(self._cfg.system_prompt, self._tools),
        )

    async def stop(self) -> None:
        """Close MCP connections. Call on shutdown."""
        self._client = None

    async def ainvoke(self, message: str) -> str:
        """Invoke the agent and return the final AI message as a string."""
        if self._agent is None:
            raise RuntimeError("MCPAgent not started — call await agent.start() first")

        result = await self._agent.ainvoke({"messages": [{"role": "user", "content": message}]})

        for m in reversed(result.get("messages", [])):
            if isinstance(m, AIMessage):
                content = m.content
                if isinstance(content, list):
                    return "\n".join(
                        part.get("text", "") if isinstance(part, dict) else str(part)
                        for part in content
                    )
                return str(content)

        return "(no response)"
