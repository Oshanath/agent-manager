# MCP Agent Sample

A LangGraph ReAct agent that dynamically loads tools, resources, and prompts from
one or more MCP (Model Context Protocol) servers and uses them when answering user
messages.

## How it works

1. At startup the agent reads the list of MCP server URLs from `BIJIRA_MCP_EVERYTHING_LOCAL_SECURE_URL`.
2. It connects to each server using the **streamable-HTTP** transport via
   `langchain-mcp-adapters`.
3. All tools exposed by the servers are loaded into a LangGraph ReAct agent.
4. The agent picks the right tool(s) for each user message and streams responses
   back through the MCP connection.

## Environment Variables

| Variable | Required | Default | Description |
|---|---|---|---|
| `BIJIRA_MCP_EVERYTHING_LOCAL_SECURE_URL` | Yes | *(empty)* | Comma-separated list of MCP proxy URLs. Injected automatically by Agent Manager when MCP proxies are bound to this agent. |
| `OPENAI_MAPPING_URL` | Yes | — | Base URL of the Agent Manager LLM proxy. Injected automatically by Agent Manager. |
| `OPENAI_MAPPING_API_KEY` | Yes | — | API key for the Agent Manager LLM proxy. Injected automatically by Agent Manager. |
| `SYSTEM_PROMPT` | No | Built-in default | Override the agent's system prompt. |

LLM calls are always routed through the Agent Manager LLM proxy — the agent does
not call OpenAI directly. When you bind MCP proxies to this agent in the Agent
Manager console, the platform automatically populates
`BIJIRA_MCP_EVERYTHING_LOCAL_SECURE_URL` with the gateway-accessible URLs of the
selected proxies.

## Example `BIJIRA_MCP_EVERYTHING_LOCAL_SECURE_URL`

```
BIJIRA_MCP_EVERYTHING_LOCAL_SECURE_URL=https://gw.example.com/my-org/search-server/v1.0/mcp,https://gw.example.com/my-org/data-server/v1.0/mcp
```

## Running locally

```bash
pip install -r requirements.txt

export BIJIRA_MCP_EVERYTHING_LOCAL_SECURE_URL="https://db720294-98fd-40f4-85a1-cc6a3b65bc9a-prod.e1-us-east-azure.choreoapis.dev/godzilla/mcp-everything-server/v1.0/mcp"
export OPENAI_MAPPING_URL="https://gw.example.com/my-org/llm-proxy/v1.0"
export OPENAI_MAPPING_API_KEY="..."

python main.py
```

Then call the agent:

```bash
curl -X POST http://localhost:8000/chat \
  -H "Content-Type: application/json" \
  -d '{"message": "What tools do you have available?"}'
```
