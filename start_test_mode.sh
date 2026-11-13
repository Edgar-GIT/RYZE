#!/usr/bin/env bash
#
# Starts the RYZE local test stack and provisions the documented test accounts.
#
# This script deliberately uses the existing migrations, exercise importer and
# administrative HTTP APIs. It never writes application tables directly.
# PostgreSQL is kept running after Ctrl-C; the local Go/Vite processes stop.

set -Eeuo pipefail

ROOT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
RUNTIME_DIR="${ROOT_DIR}/.runtime/ryze-test"
BACKEND_DIR="${ROOT_DIR}/main/src/backend"
EXERCISE_DATASET="${ROOT_DIR}/main/src/resources/exercises/exercises-dataset-main/data/exercises.json"
API_BASE="${RYZE_TEST_API_BASE:-http://localhost:8080/api/v1}"
FRONTEND_URL="${RYZE_TEST_FRONTEND_URL:-http://localhost:5173}"
BACKEND_URL="${RYZE_TEST_BACKEND_URL:-http://localhost:8080}"

BACKEND_PID=""
FRONTEND_PID=""
COMPOSE_OVERRIDE=""
HTTP_STATUS=""

usage() {
  cat <<'EOF'
Usage: ./start_test_mode.sh

Starts or reuses PostgreSQL, applies migrations, imports the exercise catalogue,
starts the local Go API and Vite frontend, then creates/reuses the configured
test client and approved trainer through the existing API.

The database is left running. Press Ctrl-C to stop the local API and frontend.
Logs are written under .runtime/ryze-test/.
EOF
}

die() {
  printf 'ERROR: %s\n' "$*" >&2
  exit 1
}

require_command() {
  command -v "$1" >/dev/null 2>&1 || die "Comando obrigatório não encontrado: $1"
}

cleanup() {
  local exit_code=$?
  trap - EXIT INT TERM

  if [[ -n "${FRONTEND_PID}" ]] && kill -0 "${FRONTEND_PID}" 2>/dev/null; then
    kill "${FRONTEND_PID}" 2>/dev/null || true
  fi
  if [[ -n "${BACKEND_PID}" ]] && kill -0 "${BACKEND_PID}" 2>/dev/null; then
    kill "${BACKEND_PID}" 2>/dev/null || true
  fi

  if [[ -n "${COMPOSE_OVERRIDE}" && -f "${COMPOSE_OVERRIDE}" ]]; then
    rm -f -- "${COMPOSE_OVERRIDE}"
  fi
  rm -f -- "${RUNTIME_DIR}"/*.json
  rm -rf -- "${RUNTIME_DIR}/cookies"

  if [[ "${exit_code}" -ne 0 ]]; then
    printf 'Os logs permanecem em %s\n' "${RUNTIME_DIR}" >&2
  fi
  exit "${exit_code}"
}

wait_for_url() {
  local url="$1"
  local label="$2"
  local attempts=60

  while (( attempts > 0 )); do
    if curl -fsS --max-time 2 "$url" >/dev/null 2>&1; then
      return 0
    fi
    sleep 1
    attempts=$((attempts - 1))
  done

  die "${label} não ficou disponível: ${url}"
}

json_file() {
  local output="$1"
  local kind="$2"

  case "${kind}" in
    admin-login)
      python3 - "${output}" <<'PY'
import json, os, sys
json.dump({"username": os.environ["RYZE_ADMIN_USERNAME"], "password": os.environ["RYZE_ADMIN_PASSWORD"]}, open(sys.argv[1], "w"), separators=(",", ":"))
PY
      ;;
    admin-verify)
      python3 - "${output}" <<'PY'
import json, os, sys
json.dump({"access_code": os.environ["RYZE_ADMIN_ACCESS_CODE"]}, open(sys.argv[1], "w"), separators=(",", ":"))
PY
      ;;
    register-client)
      python3 - "${output}" <<'PY'
import json, os, sys
json.dump({
    "email": os.environ["TEST_USER_EMAIL"],
    "password": os.environ["TEST_USER_PASSWORD"],
    "first_name": "Test",
    "last_name": "Client",
    "language": "pt",
}, open(sys.argv[1], "w"), separators=(",", ":"))
PY
      ;;
    create-user)
      python3 - "${output}" <<'PY'
import json, os, sys
json.dump({
    "email": os.environ["TEST_USER_EMAIL"],
    "password": os.environ["TEST_USER_PASSWORD"],
    "first_name": "Test",
    "last_name": "Client",
}, open(sys.argv[1], "w"), separators=(",", ":"))
PY
      ;;
    reset-client)
      python3 - "${output}" <<'PY'
import json, os, sys
json.dump({"new_password": os.environ["TEST_USER_PASSWORD"]}, open(sys.argv[1], "w"), separators=(",", ":"))
PY
      ;;
    create-trainer)
      python3 - "${output}" <<'PY'
import json, os, sys
json.dump({
    "email": os.environ["TEST_TRAINER_EMAIL"],
    "password": os.environ["TEST_TRAINER_PASSWORD"],
    "first_name": os.environ["TEST_TRAINER_FIRST_NAME"],
    "last_name": os.environ["TEST_TRAINER_LAST_NAME"],
}, open(sys.argv[1], "w"), separators=(",", ":"))
PY
      ;;
    reset-trainer)
      python3 - "${output}" <<'PY'
import json, os, sys
json.dump({"new_password": os.environ["TEST_TRAINER_PASSWORD"]}, open(sys.argv[1], "w"), separators=(",", ":"))
PY
      ;;
    test-mode-client|test-mode-trainer)
      local persona="${kind#test-mode-}"
      python3 - "${output}" "${persona}" <<'PY'
import json, sys
json.dump({"persona": sys.argv[2], "return_path": "/"}, open(sys.argv[1], "w"), separators=(",", ":"))
PY
      ;;
    *)
      die "Tipo de JSON desconhecido: ${kind}"
      ;;
  esac
}

request() {
  local method="$1"
  local path="$2"
  local body="$3"
  local response="$4"
  local cookie_jar="${5:-}"
  local -a curl_args=(
    -sS
    -o "${response}"
    -w "%{http_code}"
    -X "${method}"
    "${API_BASE}${path}"
    -H "Origin: http://localhost:5173"
  )

  if [[ -n "${body}" ]]; then
    curl_args+=( -H 'Content-Type: application/json' --data-binary "@${body}" )
  fi
  if [[ -n "${cookie_jar}" ]]; then
    curl_args+=( --cookie "${cookie_jar}" --cookie-jar "${cookie_jar}" )
  fi

  HTTP_STATUS="$(curl "${curl_args[@]}" 2>"${RUNTIME_DIR}/curl-errors.log" || true)"
  [[ "${HTTP_STATUS}" =~ ^[0-9]{3}$ ]] || HTTP_STATUS="000"
}

response_id() {
  python3 - "$1" <<'PY'
import json, sys
payload = json.load(open(sys.argv[1]))
print(payload.get("data", {}).get("id", ""))
PY
}

find_collection_id() {
  local response="$1"
  local collection="$2"
  local email="$3"
  python3 - "${response}" "${collection}" "${email}" <<'PY'
import json, sys
payload = json.load(open(sys.argv[1]))
for item in payload.get("data", {}).get(sys.argv[2], []):
    if str(item.get("email", "")).strip().lower() == sys.argv[3].strip().lower():
        print(item.get("id", ""))
        break
PY
}

find_trainer_user_id() {
  local response="$1"
  local email="$2"
  python3 - "${response}" "${email}" <<'PY'
import json, sys
payload = json.load(open(sys.argv[1]))
for item in payload.get("data", {}).get("trainers", []):
    if str(item.get("email", "")).strip().lower() == sys.argv[2].strip().lower():
        print(item.get("user_id", ""))
        break
PY
}

admin_login() {
  local admin_id="$1"
  local username_var="ADMIN_${admin_id}_USERNAME"
  local password_var="ADMIN_${admin_id}_PASSWORD"
  local access_var="ADMIN_${admin_id}_ACCESS_CODE"
  local jar="${RUNTIME_DIR}/cookies/admin-${admin_id}.jar"
  local request_body="${RUNTIME_DIR}/admin-${admin_id}-request.json"
  local response="${RUNTIME_DIR}/admin-${admin_id}-response.json"

  export RYZE_ADMIN_USERNAME="${!username_var}"
  export RYZE_ADMIN_PASSWORD="${!password_var}"
  export RYZE_ADMIN_ACCESS_CODE="${!access_var}"

  json_file "${request_body}" admin-login
  request POST /admin/auth/login "${request_body}" "${response}" "${jar}"
  [[ "${HTTP_STATUS}" == "200" ]] || die "Login de ${admin_id} falhou (HTTP ${HTTP_STATUS})"

  json_file "${request_body}" admin-verify
  request POST /admin/auth/verify "${request_body}" "${response}" "${jar}"
  [[ "${HTTP_STATUS}" == "200" ]] || die "Verificação de ${admin_id} falhou (HTTP ${HTTP_STATUS})"
  printf 'ADMIN_%s autenticado.\n' "${admin_id}"
}

reset_user_password() {
  local user_id="$1"
  local kind="$2"
  local response="${RUNTIME_DIR}/reset-${kind}.json"
  local request_body="${RUNTIME_DIR}/reset-${kind}-request.json"
  json_file "${request_body}" "reset-${kind}"
  request POST "/admin/users/${user_id}/password" "${request_body}" "${response}" "${RUNTIME_DIR}/cookies/admin-1.jar"
  [[ "${HTTP_STATUS}" == "200" ]] || die "Não foi possível atualizar a password do ${kind} (HTTP ${HTTP_STATUS})"
}

ensure_client() {
  local response="${RUNTIME_DIR}/client-response.json"
  local request_body="${RUNTIME_DIR}/client-request.json"
  local users_response="${RUNTIME_DIR}/users.json"
  local user_id=""

  json_file "${request_body}" register-client
  request POST /auth/register "${request_body}" "${response}"
  if [[ "${HTTP_STATUS}" == "201" ]]; then
    printf 'Cliente normal criado: %s\n' "${TEST_USER_EMAIL}"
    return 0
  fi
  [[ "${HTTP_STATUS}" == "409" ]] || die "Registo do cliente falhou (HTTP ${HTTP_STATUS})"

  request GET '/admin/users?page=1&limit=100' '' "${users_response}" "${RUNTIME_DIR}/cookies/admin-1.jar"
  [[ "${HTTP_STATUS}" == "200" ]] || die "Não foi possível procurar o cliente existente (HTTP ${HTTP_STATUS})"
  user_id="$(find_collection_id "${users_response}" users "${TEST_USER_EMAIL}")"
  if [[ -z "${user_id}" ]]; then
    json_file "${request_body}" create-user
    request POST /admin/users "${request_body}" "${response}" "${RUNTIME_DIR}/cookies/admin-1.jar"
    if [[ "${HTTP_STATUS}" == "201" ]]; then
      user_id="$(response_id "${response}")"
    fi
  fi
  [[ -n "${user_id}" ]] || die "O email do cliente já existe, mas não foi possível obter a conta pelo endpoint administrativo"
  reset_user_password "${user_id}" client
  printf 'Cliente normal reutilizado: %s\n' "${TEST_USER_EMAIL}"
}

ensure_trainer() {
  local response="${RUNTIME_DIR}/trainer-response.json"
  local request_body="${RUNTIME_DIR}/trainer-request.json"
  local trainers_response="${RUNTIME_DIR}/trainers.json"
  local trainer_id=""
  local user_id=""
  local status=""

  request GET '/admin/trainers?page=1&limit=100' '' "${trainers_response}" "${RUNTIME_DIR}/cookies/admin-1.jar"
  [[ "${HTTP_STATUS}" == "200" ]] || die "Não foi possível procurar treinadores existentes (HTTP ${HTTP_STATUS})"
  trainer_id="$(find_collection_id "${trainers_response}" trainers "${TEST_TRAINER_EMAIL}")"
  if [[ -n "${trainer_id}" ]]; then
    user_id="$(find_trainer_user_id "${trainers_response}" "${TEST_TRAINER_EMAIL}")"
    status="$(python3 - "${trainers_response}" "${TEST_TRAINER_EMAIL}" <<'PY'
import json, sys
for item in json.load(open(sys.argv[1])).get("data", {}).get("trainers", []):
    if item.get("email", "").lower() == sys.argv[2].lower():
        print(item.get("status", ""))
        break
PY
)"
    [[ "${status}" == "active" ]] || die "A conta do treinador existe, mas não está ativa (status: ${status})"
    reset_user_password "${user_id}" trainer
    printf 'Treinador aprovado reutilizado: %s\n' "${TEST_TRAINER_EMAIL}"
    return 0
  fi

  request GET '/admin/trainers/deleted?page=1&limit=100' '' "${trainers_response}" "${RUNTIME_DIR}/cookies/admin-1.jar"
  [[ "${HTTP_STATUS}" == "200" ]] || die "Não foi possível procurar treinadores arquivados (HTTP ${HTTP_STATUS})"
  trainer_id="$(find_collection_id "${trainers_response}" trainers "${TEST_TRAINER_EMAIL}")"
  if [[ -n "${trainer_id}" ]]; then
    request POST "/admin/trainers/${trainer_id}/reactivate" '' "${response}" "${RUNTIME_DIR}/cookies/admin-1.jar"
    [[ "${HTTP_STATUS}" == "200" ]] || die "Não foi possível reativar o treinador existente (HTTP ${HTTP_STATUS})"
    request GET '/admin/trainers?page=1&limit=100' '' "${trainers_response}" "${RUNTIME_DIR}/cookies/admin-1.jar"
    [[ "${HTTP_STATUS}" == "200" ]] || die "Não foi possível confirmar o treinador reativado (HTTP ${HTTP_STATUS})"
    user_id="$(find_trainer_user_id "${trainers_response}" "${TEST_TRAINER_EMAIL}")"
    [[ -n "${user_id}" ]] || die "Treinador reativado sem utilizador associado"
    reset_user_password "${user_id}" trainer
    printf 'Treinador aprovado reativado: %s\n' "${TEST_TRAINER_EMAIL}"
    return 0
  fi

  json_file "${request_body}" create-trainer
  request POST /admin/trainers "${request_body}" "${response}" "${RUNTIME_DIR}/cookies/admin-1.jar"
  if [[ "${HTTP_STATUS}" == "201" ]]; then
    printf 'Treinador aprovado criado: %s\n' "${TEST_TRAINER_EMAIL}"
    return 0
  fi
  die "Não foi possível criar o treinador (HTTP ${HTTP_STATUS}); não foi criada uma conta duplicada"
}

ensure_test_mode_personas() {
  local persona="$1"
  local request_body="${RUNTIME_DIR}/test-mode-${persona}-request.json"
  local response="${RUNTIME_DIR}/test-mode-${persona}-response.json"
  json_file "${request_body}" "test-mode-${persona}"
  request POST /admin/auth/test-mode "${request_body}" "${response}" "${RUNTIME_DIR}/cookies/admin-1.jar"
  [[ "${HTTP_STATUS}" == "200" ]] || die "Test Mode (${persona}) falhou ao iniciar (HTTP ${HTTP_STATUS})"
  request POST /admin/auth/test-mode/exit '' "${response}" "${RUNTIME_DIR}/cookies/admin-1.jar"
  [[ "${HTTP_STATUS}" == "200" ]] || die "Test Mode (${persona}) falhou ao terminar (HTTP ${HTTP_STATUS})"
  printf 'Persona Test Mode pronta: %s\n' "${persona}"
}

start_database() {
  local host="${DB_HOST:-127.0.0.1}"
  local port="${DB_PORT:-5432}"

  if pg_isready -h "${host}" -p "${port}" -U "${DB_USER}" -d "${DB_NAME}" >/dev/null 2>&1; then
    printf 'PostgreSQL já disponível em %s:%s.\n' "${host}" "${port}"
    return 0
  fi

  require_command docker
  docker compose version >/dev/null 2>&1 || die 'Docker Compose não está disponível'
  COMPOSE_OVERRIDE="$(mktemp "${RUNTIME_DIR}/compose.XXXXXX.yml")"
  cat >"${COMPOSE_OVERRIDE}" <<EOF
services:
  db:
    ports:
      - "${port}:5432"
EOF
  docker compose -f "${ROOT_DIR}/docker-compose.yml" -f "${COMPOSE_OVERRIDE}" up -d db
  export DB_HOST="127.0.0.1"

  local attempts=60
  while (( attempts > 0 )); do
    if pg_isready -h "${DB_HOST}" -p "${DB_PORT}" -U "${DB_USER}" -d "${DB_NAME}" >/dev/null 2>&1; then
      printf 'PostgreSQL iniciado através do Docker em %s:%s.\n' "${DB_HOST}" "${DB_PORT}"
      return 0
    fi
    sleep 1
    attempts=$((attempts - 1))
  done
  die 'PostgreSQL não ficou pronto após iniciar o serviço Docker'
}

main() {
  if [[ "${1:-}" == "--help" || "${1:-}" == "-h" ]]; then
    usage
    return 0
  fi
  [[ -z "${1:-}" ]] || die "Argumento desconhecido: $1 (use --help)"

  require_command curl
  require_command go
  require_command npm
  require_command python3
  require_command pg_isready
  [[ -f "${ROOT_DIR}/scripts/env_loader.sh" ]] || die 'scripts/env_loader.sh não encontrado'

  mkdir -p "${RUNTIME_DIR}/cookies"
  umask 077
  chmod 700 "${RUNTIME_DIR}" "${RUNTIME_DIR}/cookies"
  # shellcheck disable=SC1091
  source "${ROOT_DIR}/scripts/env_loader.sh"
  load_project_env "${ROOT_DIR}"

  : "${DB_USER:?DB_USER não definido no .env}"
  : "${DB_NAME:?DB_NAME não definido no .env}"
  : "${DB_PASSWORD:?DB_PASSWORD não definido no .env}"
  : "${ADMIN_1_USERNAME:?ADMIN_1_USERNAME não definido no .env}"
  : "${ADMIN_1_PASSWORD:?ADMIN_1_PASSWORD não definido no .env}"
  : "${ADMIN_1_ACCESS_CODE:?ADMIN_1_ACCESS_CODE não definido no .env}"
  : "${ADMIN_2_USERNAME:?ADMIN_2_USERNAME não definido no .env}"
  : "${ADMIN_2_PASSWORD:?ADMIN_2_PASSWORD não definido no .env}"
  : "${ADMIN_2_ACCESS_CODE:?ADMIN_2_ACCESS_CODE não definido no .env}"
  : "${TEST_USER_EMAIL:?TEST_USER_EMAIL não definido no .env}"
  : "${TEST_USER_PASSWORD:?TEST_USER_PASSWORD não definido no .env}"
  : "${TEST_TRAINER_EMAIL:?TEST_TRAINER_EMAIL não definido no .env}"
  : "${TEST_TRAINER_PASSWORD:?TEST_TRAINER_PASSWORD não definido no .env}"
  : "${TEST_TRAINER_FIRST_NAME:?TEST_TRAINER_FIRST_NAME não definido no .env}"
  : "${TEST_TRAINER_LAST_NAME:?TEST_TRAINER_LAST_NAME não definido no .env}"

  start_database

  (cd "${BACKEND_DIR}" && go run . migrate up) >"${RUNTIME_DIR}/migrations.log" 2>&1 || die "As migrações falharam; consulte ${RUNTIME_DIR}/migrations.log"
  if [[ -f "${EXERCISE_DATASET}" ]]; then
    (cd "${BACKEND_DIR}" && go run . import-exercises --file "${EXERCISE_DATASET}") >"${RUNTIME_DIR}/exercise-import.log" 2>&1 || die "A importação de exercícios falhou; consulte ${RUNTIME_DIR}/exercise-import.log"
  else
    printf 'AVISO: dataset de exercícios não encontrado; o catálogo existente será preservado.\n' >&2
  fi

  mkdir -p "${RUNTIME_DIR}"
  (cd "${BACKEND_DIR}" && \
    PORT=8080 COOKIE_SECURE=false TEST_MODE_ENABLED=true \
    EXERCISE_MEDIA_ENABLED=true EXERCISE_MEDIA_BASE_PATH=/dev-exercise-media \
    go run .) >"${RUNTIME_DIR}/backend.log" 2>&1 &
  BACKEND_PID=$!
  wait_for_url "${BACKEND_URL}/api/v1/health" 'Backend'

  (cd "${ROOT_DIR}" && \
    VITE_API_BASE_URL="${API_BASE}" npm run dev -- --host 127.0.0.1 --port 5173) >"${RUNTIME_DIR}/frontend.log" 2>&1 &
  FRONTEND_PID=$!
  wait_for_url "${FRONTEND_URL}" 'Frontend'

  admin_login 1
  admin_login 2
  ensure_client
  ensure_trainer
  ensure_test_mode_personas client
  ensure_test_mode_personas trainer

  printf '\nRYZE Test Mode pronto.\n'
  printf 'Frontend: %s\n' "${FRONTEND_URL}"
  printf 'Backend:  %s\n' "${BACKEND_URL}"
  printf 'Credenciais de teste: .env (não são impressas pelo script).\n'
  printf 'Ctrl-C termina frontend/backend e mantém PostgreSQL ativo.\n\n'

  while kill -0 "${BACKEND_PID}" 2>/dev/null && kill -0 "${FRONTEND_PID}" 2>/dev/null; do
    sleep 2
  done
  die 'Frontend ou backend terminou inesperadamente; consulte os logs'
}

trap cleanup EXIT INT TERM
main "$@"
