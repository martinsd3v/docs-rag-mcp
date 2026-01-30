#!/bin/bash
# Test MCP server

set -e

SERVER_BIN="./bin/docs-rag-mcp-server"
DB_PATH="./data/docs.db"

export DB_PATH OPENAI_API_KEY=token-marcelo OPENAI_BASE_URL=https://ai.v3m.ai/v1

echo "=== Testing MCP Server ==="
echo ""

# Create a temp file for communication
INPUT_FILE=$(mktemp)
OUTPUT_FILE=$(mktemp)

cleanup() {
    rm -f "$INPUT_FILE" "$OUTPUT_FILE"
    # Kill any background processes
    jobs -p | xargs -r kill 2>/dev/null || true
}
trap cleanup EXIT

# Test 1: Initialize
echo "Test 1: Initialize"
echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"1.0"}}}' > "$INPUT_FILE"

timeout 5 "$SERVER_BIN" < "$INPUT_FILE" > "$OUTPUT_FILE" 2>/dev/null || true
cat "$OUTPUT_FILE"
echo ""

# Test 2: Tools list
echo "Test 2: Tools list"
cat > "$INPUT_FILE" << 'EOF'
{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"1.0"}}}
{"jsonrpc":"2.0","id":2,"method":"tools/list"}
EOF

timeout 5 "$SERVER_BIN" < "$INPUT_FILE" > "$OUTPUT_FILE" 2>/dev/null || true
cat "$OUTPUT_FILE"
echo ""

echo "=== Tests Complete ==="
