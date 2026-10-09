#!/bin/bash
set -e

# Initialize local agent credentials if needed
AUTH_DIR="/var/lib/mcrflow-agent"
AUTH_FILE="${AUTH_DIR}/agent_auth.json"

mkdir -p "${AUTH_DIR}" /var/lib/mcrflow-data /var/log

if [ ! -f "${AUTH_FILE}" ]; then
    RANDOM_HEX=$(openssl rand -hex 32)
    AGENT_TOKEN="agt_sec_${RANDOM_HEX}"
    cat <<EOF > "${AUTH_FILE}"
{
  "agent_id": "embedded-local-agent",
  "pairing_token": "${AGENT_TOKEN}",
  "created_at": "$(date -u +"%Y-%m-%dT%H:%M:%SZ")",
  "version": "1.0.0"
}
EOF
    chmod 600 "${AUTH_FILE}"
fi

echo "================================================================================"
echo "  MCRFLOW PLAYOUT - ALL-IN-ONE CONTAINER BOOTED"
echo "  Web Management UI: http://localhost:3081"
echo "  Control Plane gRPC: :9090"
echo "  Embedded Agent:     :3082 (Auto-Paired locally)"
echo "================================================================================"

exec "$@"
