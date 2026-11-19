#!/usr/bin/env bash
#
# Stops the local Go/Vite processes started by start_test_mode.sh.
# PostgreSQL is left running by default because the start script may have
# reused an existing database. Use --database to stop the compose database too.

set -Eeuo pipefail

ROOT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
RUNTIME_DIR="${ROOT_DIR}/.runtime/ryze-test"

usage() {
  cat <<'EOF'
Usage: ./stop_test_mode.sh [--database]

Stops the backend and frontend processes recorded by start_test_mode.sh.
The PostgreSQL data and Docker volume are never deleted.

--database  Also stops the PostgreSQL service from this repository's
            docker-compose.yml. It does not remove the database volume.
EOF
}

process_command() {
  local pid="$1"
  [[ -r "/proc/${pid}/cmdline" ]] || return 0
  tr '\0' ' ' <"/proc/${pid}/cmdline" 2>/dev/null || true
}

matches_expected_process() {
  local name="$1"
  local pid="$2"
  local command
  command="$(process_command "${pid}")"

  case "${name}" in
    backend)
      [[ "${command}" == *"${RUNTIME_DIR}/ryze-backend"* ]]
      ;;
    frontend)
      [[ "${command}" == *"node_modules/.bin/vite"* && "${command}" == *"--port 5173"* ]]
      ;;
    *)
      return 1
      ;;
  esac
}

stop_process() {
  local name="$1"
  local pid_file="${RUNTIME_DIR}/${name}.pid"
  local pid command attempt

  if [[ ! -f "${pid_file}" ]]; then
    printf '%s não está registado.\n' "${name}"
    return 0
  fi

  pid="$(<"${pid_file}")"
  if [[ ! "${pid}" =~ ^[0-9]+$ ]] || ! kill -0 "${pid}" 2>/dev/null; then
    printf 'PID antigo removido: %s.\n' "${pid_file}"
    rm -f -- "${pid_file}"
    return 0
  fi

  command="$(process_command "${pid}")"
  if ! matches_expected_process "${name}" "${pid}"; then
    printf 'PID %s não corresponde ao processo esperado de %s; não foi terminado.\n' "${pid}" "${name}" >&2
    printf 'Processo encontrado: %s\n' "${command:-indisponível}" >&2
    rm -f -- "${pid_file}"
    return 1
  fi

  printf 'A parar %s (pid %s)...\n' "${name}" "${pid}"
  kill -TERM "${pid}" 2>/dev/null || true
  for attempt in {1..50}; do
    if ! kill -0 "${pid}" 2>/dev/null || ! matches_expected_process "${name}" "${pid}"; then
      rm -f -- "${pid_file}"
      printf '%s parado.\n' "${name}"
      return 0
    fi
    sleep 0.2
  done

  printf '%s não terminou após 10 segundos; a enviar SIGKILL.\n' "${name}" >&2
  if matches_expected_process "${name}" "${pid}"; then
    kill -KILL "${pid}" 2>/dev/null || true
  fi
  rm -f -- "${pid_file}"
}

stop_database() {
  command -v docker >/dev/null 2>&1 || {
    printf 'Docker não está disponível; a base de dados não foi parada.\n' >&2
    return 1
  }
  docker compose version >/dev/null 2>&1 || {
    printf 'Docker Compose não está disponível; a base de dados não foi parada.\n' >&2
    return 1
  }

  printf 'A parar o serviço PostgreSQL do compose...\n'
  docker compose --project-directory "${ROOT_DIR}" stop db
}

main() {
  local stop_db=false
  case "${1:-}" in
    '') ;;
    --database) stop_db=true ;;
    --help|-h) usage; return 0 ;;
    *) printf 'Argumento desconhecido: %s\n\n' "$1" >&2; usage >&2; return 2 ;;
  esac

  local failed=false
  stop_process backend || failed=true
  stop_process frontend || failed=true

  if [[ "${stop_db}" == true ]]; then
    stop_database || failed=true
  else
    printf 'PostgreSQL mantido em execução. Use --database para o parar também.\n'
  fi

  [[ "${failed}" == false ]]
}

main "$@"
