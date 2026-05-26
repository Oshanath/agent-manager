"""Instance-level configuration, read from env at startup.

MCP server URLs are provided as a comma-separated list in ``MCP_SERVER_URLS``.
Each URL must point to a deployed MCP proxy (e.g. the gateway-exposed endpoint).

LLM calls are always routed through the Agent Manager LLM proxy using
``OPENAI_MAPPING_URL`` and ``OPENAI_MAPPING_API_KEY``, which are injected
automatically by Agent Manager.
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
    openai_mapping_url: str
    openai_mapping_api_key: str
    system_prompt: str

    @classmethod
    def from_env(cls) -> "Config":
        raw_urls = _env("MCP_SERVER_URLS", "")
        mcp_server_urls = [u.strip() for u in raw_urls.split(",") if u.strip()]

        return cls(
            mcp_server_urls=mcp_server_urls,
            openai_mapping_url=_env("OPENAI_MAPPING_URL"),
            openai_mapping_api_key=_env("OPENAI_MAPPING_API_KEY"),
            system_prompt=_env(
                "SYSTEM_PROMPT",
                "You are a helpful assistant with access to tools, resources, and prompts "
                "provided by connected MCP servers. Use these capabilities to assist the user.",
            ),
        )
