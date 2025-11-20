#!/bin/bash
# OpenShift commands to set environment variables for whatsapp-mcp-server deployment

# LlamaStack Configuration
oc set env deployment/whatsapp-mcp-server \
  LLAMASTACK_BASE_URL="http://ragathon-team-1-ragathon-team-1.apps.llama-rag-pool-b84hp.aws.rh-ods.com/" \
  LLAMASTACK_API_KEY="your_api_key_here" \
  LLAMASTACK_MODEL="vllm-inference/llama-4-scout-17b-16e-w4a16" \
  LLAMASTACK_TEMPERATURE="0.7" \
  LLAMASTACK_MAX_TOKENS="200"

# Vector Store Configuration
oc set env deployment/whatsapp-mcp-server \
  VECTOR_STORE_NAME="redbank-kb-vector-store"

# MCP Tool Group Configuration
# Note: The MCP server must be pre-registered as a tool group first
# Register it using: llama-stack-client toolgroups register <toolgroup_id> --mcp-config <config>
# Then reference it here:
oc set env deployment/whatsapp-mcp-server \
  LLAMASTACK_MCP_TOOL_GROUP="mcp::redbank-financials"

echo "Environment variables set successfully!"
echo "Note: Update LLAMASTACK_API_KEY with your actual API key"

