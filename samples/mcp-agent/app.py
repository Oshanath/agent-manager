"""FastAPI entrypoint for the MCP agent.

Implements the AM chat-agent contract: ``POST /chat`` on port 8000 accepting
``{session_id, message, context}`` and returning ``{response, session_id}``.
``GET /health`` is provided for liveness checks.

The MCP client connections are kept open for the process lifetime using
FastAPI's lifespan context manager.
"""

from __future__ import annotations

import logging
from contextlib import asynccontextmanager
from typing import Any

from fastapi import FastAPI, HTTPException
from pydantic import BaseModel

from agent import MCPAgent
from config import Config

logging.basicConfig(level=logging.INFO)
log = logging.getLogger("mcp-agent")

CONFIG = Config.from_env()
_agent = MCPAgent(CONFIG)


@asynccontextmanager
async def lifespan(app: FastAPI):  # noqa: ANN001
    log.info(
        "MCP agent starting (servers=%d, llm_provider=agent-manager-proxy)",
        len(CONFIG.mcp_server_urls),
    )
    await _agent.start()
    yield
    await _agent.stop()
    log.info("MCP agent stopped")


class ChatRequest(BaseModel):
    message: str
    session_id: str | None = None
    context: dict[str, Any] | None = None


class ChatResponse(BaseModel):
    response: str
    session_id: str | None = None


app = FastAPI(title="MCP Agent", version="0.1.0", lifespan=lifespan)


@app.get("/health")
def health() -> dict[str, Any]:
    return {
        "status": "ok",
        "mcp_servers": len(CONFIG.mcp_server_urls),
        "tools_loaded": len(_agent.tool_names),
        "tools": _agent.tool_names,
    }


@app.post("/chat", response_model=ChatResponse)
async def chat(req: ChatRequest) -> ChatResponse:
    try:
        response = await _agent.ainvoke(req.message)
    except Exception as exc:
        log.exception("agent invocation failed")
        raise HTTPException(status_code=500, detail=str(exc)) from exc

    return ChatResponse(response=response, session_id=req.session_id)
