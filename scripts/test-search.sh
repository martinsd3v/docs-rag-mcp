#!/bin/bash

export DB_PATH=./data/docs.db
export OPENAI_API_KEY=token-marcelo
export OPENAI_BASE_URL=https://ai.v3m.ai/v1

# Send requests to server
{
    echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"1.0"}}}'
    sleep 0.5
    echo '{"jsonrpc":"2.0","method":"initialized"}'
    sleep 0.5
    echo '{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"search_docs","arguments":{"query":"microservice architecture patterns","top_k":3}}}'
    sleep 3
} | ./bin/docs-rag-mcp-server 2>&1
