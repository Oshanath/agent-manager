"""Instance-level configuration, read from env at startup.

MCP server URLs can be provided as a comma-separated list in
``BIJIRA_MCP_EVERYTHING_URL``.
Each URL must point to a deployed MCP proxy (e.g. the gateway-exposed endpoint).
MCP proxy authentication uses ``BIJIRA_MCP_EVERYTHING_API_KEY`` when MCP
configuration is present.

LLM calls are always routed through the Agent Manager LLM proxy using
``OPENAI_URL`` and ``OPENAI_API_KEY``, which are injected automatically by
Agent Manager.
"""

from __future__ import annotations

import os
from dataclasses import dataclass


def _env(name: str, default: str | None = None) -> str:
    val = os.environ.get(name, default)
    if val is None:
        raise RuntimeError(f"Missing required env var: {name}")
    return val


@dataclass(frozen=True)
class Config:
    mcp_server_urls: list[str]
    mcp_api_key: str | None
    openai_url: str
    openai_api_key: str
    system_prompt: str

    @classmethod
    def from_env(cls) -> "Config":
        raw_urls = _env("BIJIRA_MCP_EVERYTHING_URL", "")
        mcp_server_urls = [u.strip() for u in raw_urls.split(",") if u.strip()]
        mcp_api_key = _env("BIJIRA_MCP_EVERYTHING_API_KEY", "").strip() or None

        return cls(
            mcp_server_urls=mcp_server_urls,
            mcp_api_key=mcp_api_key,
            openai_url=_env("OPENAI_URL"),
            openai_api_key=_env("OPENAI_API_KEY"),
            system_prompt=_env(
                "SYSTEM_PROMPT",
                "You are a helpful assistant with access to tools, resources, and prompts "
                "provided by connected MCP servers. Use these capabilities to assist the user.",
            ),
        )
