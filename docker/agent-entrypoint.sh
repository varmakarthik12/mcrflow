#!/bin/bash
set -e

AUTH_DIR="/var/lib/playout-agent"
AUTH_FILE="${AUTH_DIR}/agent_auth.json"

mkdir -p "${AUTH_DIR}"

if [ ! -f "${AUTH_FILE}" ]; then
    echo "[OMNISTREAM AGENT] Initializing fresh edge playout agent instance..."
    
    # Generate cryptographically secure token (32 random bytes as hex string)
    RANDOM_HEX=$(openssl rand -hex 32)
    AGENT_TOKEN="agt_sec_${RANDOM_HEX}"
    AGENT_ID="${AGENT_HOSTNAME:-agent-node-$(cat /proc/sys/kernel/random/uuid 2>/dev/null | cut -c 1-8 || date +%s)}"
    
    # Save to persistent storage with strict permissions
    cat <<EOF > "${AUTH_FILE}"
{
  "agent_id": "${AGENT_ID}",
  "pairing_token": "${AGENT_TOKEN}",
  "created_at": "$(date -u +"%Y-%m-%dT%H:%M:%SZ")",
  "version": "1.0.0"
}
EOF
    chmod 600 "${AUTH_FILE}"
    
    echo ""
    echo "================================================================================"
    echo "  OMNISTREAM EDGE PLAYOUT AGENT - CRYPTOGRAPHIC PAIRING REQUIRED"
    echo "================================================================================"
    echo "  Agent ID:         ${AGENT_ID}"
    echo "  Listen Port:      :9095 (gRPC / mTLS)"
    echo "  Persistent Token: ${AGENT_TOKEN}"
    echo ""
    echo "  👉 Copy and paste the token above into OmniStream Web UI:"
    echo "     Settings > Edge Agents > Pair New Agent"
    echo "================================================================================"
    echo ""
else
    AGENT_ID=$(jq -r '.agent_id' "${AUTH_FILE}" 2>/dev/null || echo "edge-agent")
    AGENT_TOKEN=$(jq -r '.pairing_token' "${AUTH_FILE}" 2>/dev/null || echo "UNKNOWN")
    
    echo ""
    echo "================================================================================"
    echo "  OMNISTREAM EDGE PLAYOUT AGENT - EXISTING PAIRING RESTORED"
    echo "================================================================================"
    echo "  Agent ID:         ${AGENT_ID}"
    echo "  Token Status:     Active (Preserved across container restart)"
    echo "  Persistent Token: ${AGENT_TOKEN}"
    echo "================================================================================"
    echo ""
fi

# Execute playout agent daemon
exec "$@"
