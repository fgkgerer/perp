#!/usr/bin/env bash
# MetaNode 后端进程管理：start | stop | restart | redeploy | status
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$ROOT_DIR"

APP_NAME="metanode"
BIN="${ROOT_DIR}/bin/${APP_NAME}"
PID_FILE="${ROOT_DIR}/run/${APP_NAME}.pid"
LOG_DIR="${ROOT_DIR}/logs"
LOG_FILE="${LOG_DIR}/${APP_NAME}.log"

if [[ -n "${METANODE_CONFIG:-}" ]]; then
  CONFIG="${METANODE_CONFIG}"
elif [[ -f "${ROOT_DIR}/etc/metanode.prod.yaml" ]]; then
  CONFIG="${ROOT_DIR}/etc/metanode.prod.yaml"
else
  CONFIG="${ROOT_DIR}/etc/metanode.yaml"
fi

mkdir -p "${ROOT_DIR}/run" "${ROOT_DIR}/bin" "${LOG_DIR}"

log() {
  echo "[$(date '+%Y-%m-%d %H:%M:%S')] $*"
}

die() {
  log "ERROR: $*"
  exit 1
}

git_root() {
  git -C "${ROOT_DIR}" rev-parse --show-toplevel 2>/dev/null || true
}

is_running() {
  [[ -f "${PID_FILE}" ]] || return 1
  local pid
  pid="$(cat "${PID_FILE}")"
  [[ -n "${pid}" ]] && kill -0 "${pid}" 2>/dev/null
}

running_pid() {
  if is_running; then
    cat "${PID_FILE}"
  fi
}

build() {
  log "编译 ${APP_NAME}..."
  if ! command -v go >/dev/null 2>&1; then
    die "未找到 go，请先安装 Go 或将已编译的 bin/metanode 放到 ${BIN}"
  fi
  CGO_ENABLED=0 go build -ldflags="-s -w" -o "${BIN}" metanode.go
  log "编译完成: ${BIN}"
}

ensure_binary() {
  if [[ ! -x "${BIN}" ]]; then
    build
  fi
}

health_check() {
  local url="http://127.0.0.1:28888/api/v1/markets"
  if command -v curl >/dev/null 2>&1; then
    curl -fsS --max-time 5 "${url}" >/dev/null 2>&1
  else
    return 0
  fi
}

cmd_start() {
  if is_running; then
    log "${APP_NAME} 已在运行 (pid=$(cat "${PID_FILE}"))"
    return 0
  fi

  [[ -f "${CONFIG}" ]] || die "配置文件不存在: ${CONFIG}"

  ensure_binary

  log "启动 ${APP_NAME}，配置: ${CONFIG}"
  nohup "${BIN}" -f "${CONFIG}" >>"${LOG_FILE}" 2>&1 &
  echo $! >"${PID_FILE}"
  sleep 1

  if ! is_running; then
    rm -f "${PID_FILE}"
    die "启动失败，请查看日志: ${LOG_FILE}"
  fi

  log "${APP_NAME} 已启动 (pid=$(cat "${PID_FILE}"))，日志: ${LOG_FILE}"

  if health_check; then
    log "健康检查通过: /api/v1/markets"
  else
    log "WARN: 健康检查未通过，服务可能仍在初始化，请稍后 curl http://127.0.0.1:28888/api/v1/markets"
  fi
}

cmd_stop() {
  if ! is_running; then
    rm -f "${PID_FILE}"
    log "${APP_NAME} 未在运行"
    return 0
  fi

  local pid
  pid="$(cat "${PID_FILE}")"
  log "停止 ${APP_NAME} (pid=${pid})..."

  kill "${pid}" 2>/dev/null || true

  local i
  for i in $(seq 1 30); do
    if ! kill -0 "${pid}" 2>/dev/null; then
      rm -f "${PID_FILE}"
      log "${APP_NAME} 已停止"
      return 0
    fi
    sleep 1
  done

  log "WARN: 进程未在 30s 内退出，发送 SIGKILL"
  kill -9 "${pid}" 2>/dev/null || true
  rm -f "${PID_FILE}"
  log "${APP_NAME} 已强制停止"
}

cmd_restart() {
  cmd_stop
  cmd_start
}

cmd_redeploy() {
  if [[ "${SKIP_GIT_PULL:-0}" != "1" ]]; then
    local repo
    repo="$(git_root)"
    if [[ -n "${repo}" ]]; then
      log "拉取最新代码: ${repo}"
      git -C "${repo}" pull --ff-only
    else
      log "WARN: 非 git 仓库，跳过 git pull（可设 SKIP_GIT_PULL=1 静默跳过）"
    fi
  fi

  build
  cmd_restart
  log "redeploy 完成"
}

cmd_status() {
  if is_running; then
    log "${APP_NAME} 运行中 (pid=$(cat "${PID_FILE}"))，配置: ${CONFIG}"
    if health_check; then
      log "健康检查: OK"
    else
      log "健康检查: 失败或未就绪"
    fi
  else
    log "${APP_NAME} 未运行"
    rm -f "${PID_FILE}"
    return 1
  fi
}

usage() {
  cat <<EOF
用法: $(basename "$0") <command>

命令:
  start     后台启动（缺二进制时自动编译）
  stop      停止进程
  restart   重启
  redeploy  git pull + 编译 + 重启
  status    查看运行状态

环境变量:
  METANODE_CONFIG   配置文件路径（默认 etc/metanode.prod.yaml 或 etc/metanode.yaml）
  SKIP_GIT_PULL=1   redeploy 时跳过 git pull

示例:
  ./app.sh start
  METANODE_CONFIG=/etc/metanode/metanode.yaml ./app.sh redeploy
  tail -f logs/metanode.log
EOF
}

main() {
  local cmd="${1:-}"
  case "${cmd}" in
    start) cmd_start ;;
    stop) cmd_stop ;;
    restart) cmd_restart ;;
    redeploy) cmd_redeploy ;;
    status) cmd_status ;;
    -h|--help|help|"") usage ;;
    *) die "未知命令: ${cmd}（执行 $(basename "$0") help 查看用法）" ;;
  esac
}

main "$@"
