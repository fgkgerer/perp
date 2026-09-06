#!/usr/bin/env bash
# Sepolia：MetaNode 一键部署 +（可选）Etherscan 验证
# 用法：在项目根目录执行 ./script/run_deploy_sepolia.sh
# 依赖：.env 或环境变量 MetaNode_DEPLOYER_PK、SEPOLIA_RPC_URL；验证需 ETHERSCAN_API_KEY
# 建议：`foundryup` 安装 stable，并让 ~/.foundry/bin 优先于旧版（如 cargo）的 forge，否则 --verify 可能因 Etherscan API 失效。

set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

if [[ -x "${HOME}/.foundry/bin/forge" ]]; then
  PATH="${HOME}/.foundry/bin:${PATH}"
  export PATH
fi

if [[ -f "$ROOT/.env" ]]; then
  set -a
  # shellcheck source=/dev/null
  source "$ROOT/.env"
  set +a
fi
# 父环境或 CI 可设 SEPOLIA_RPC_URL_OVERRIDE，覆盖 .env（备用节点，避免单个 RPC TLS/限流）
if [[ -n "${SEPOLIA_RPC_URL_OVERRIDE:-}" ]]; then
  export SEPOLIA_RPC_URL="$SEPOLIA_RPC_URL_OVERRIDE"
fi

export FOUNDRY_PROFILE="${FOUNDRY_PROFILE:-sepolia}"
: "${MetaNode_DEPLOYER_PK:?请先设置 MetaNode_DEPLOYER_PK（部署私钥）}"
: "${SEPOLIA_RPC_URL:?请先设置 SEPOLIA_RPC_URL}"

ADDR="$(cast wallet address --private-key "$MetaNode_DEPLOYER_PK")"
BAL="$(cast balance "$ADDR" --rpc-url "$SEPOLIA_RPC_URL")"
echo "Deployer: $ADDR"
echo "Balance:  $BAL wei"

ARGS=(--rpc-url "$SEPOLIA_RPC_URL" --broadcast -vvvv)
if [[ -n "${ETHERSCAN_API_KEY:-}" ]]; then
  ARGS+=(--verify --etherscan-api-key "$ETHERSCAN_API_KEY")
  echo "Etherscan verification: enabled"
else
  echo "ETHERSCAN_API_KEY 未设置，跳过链上验证（部署仍会执行）"
fi

exec forge script script/DeploySepolia.s.sol:DeploySepoliaScript "${ARGS[@]}"
