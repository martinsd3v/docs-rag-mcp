#!/bin/bash

export DB_PATH=./data/docs.db
export OPENAI_API_KEY=token-marcelo
export OPENAI_BASE_URL=https://ai.v3m.ai/v1

# Send requests to server
{
    echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"1.0"}}}'
    sleep 0.5
    echo '{"jsonrpc":"2.0","id":2,"method":"tools/list"}'
    sleep 0.5
} | ./bin/docs-rag-mcp-server 2>&1
