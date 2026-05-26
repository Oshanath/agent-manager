"""Programmatic entrypoint for the MCP agent.

Used as the AM build's run command: ``python main.py``.
"""

from __future__ import annotations

import uvicorn
from dotenv import load_dotenv

load_dotenv()

from app import app


def main() -> None:
    uvicorn.run(app, host="0.0.0.0", port=8000)


if __name__ == "__main__":
    main()
