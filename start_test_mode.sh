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
Usage: ./start_test_mode.sh [--accounts-only]

Starts or reuses PostgreSQL, applies migrations, imports the exercise catalogue,
starts the local Go API and Vite frontend, then creates/reuses the configured
test client and approved trainer through the existing API.

--accounts-only  Only checks (and, when needed, repairs) the test accounts
                 against an API that is already running, then exits.

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
  rm -f -- "${RUNTIME_DIR}"/*.json "${RUNTIME_DIR}"/*.pid
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

url_responds() {
  curl -fsS --max-time 2 "$1" >/dev/null 2>&1
}

# stop_previous_run stops servers left by an earlier run of this script that
# did not shut down cleanly (their PIDs are kept under .runtime/).
stop_previous_run() {
  local name pid_file pid
  for name in backend frontend; do
    pid_file="${RUNTIME_DIR}/${name}.pid"
    [[ -f "${pid_file}" ]] || continue
    pid="$(cat -- "${pid_file}")"
    if [[ "${pid}" =~ ^[0-9]+$ ]] && kill -0 "${pid}" 2>/dev/null; then
      kill "${pid}" 2>/dev/null || true
      printf 'Parado %s de uma execução anterior (pid %s).\n' "${name}" "${pid}"
    fi
    rm -f -- "${pid_file}"
  done
  sleep 1
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
    login-client)
      python3 - "${output}" <<'PY'
import json, os, sys
json.dump({"email": os.environ["TEST_USER_EMAIL"], "password": os.environ["TEST_USER_PASSWORD"]}, open(sys.argv[1], "w"), separators=(",", ":"))
PY
      ;;
    login-trainer)
      python3 - "${output}" <<'PY'
import json, os, sys
json.dump({"email": os.environ["TEST_TRAINER_EMAIL"], "password": os.environ["TEST_TRAINER_PASSWORD"]}, open(sys.argv[1], "w"), separators=(",", ":"))
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

# find_account walks every page of an administrative listing (the API caps a
# page at 100 rows and has no email search) and prints "<id> <user_id> <status>"
# for the row whose email matches. It returns 1 when no page has the email.
find_account() {
  local path="$1"
  local collection="$2"
  local email="$3"
  local response="${RUNTIME_DIR}/lookup.json"
  local page=1
  local total_pages=1
  local match=""

  while (( page <= total_pages )); do
    request GET "${path}?page=${page}&limit=100" '' "${response}" "${RUNTIME_DIR}/cookies/admin-1.jar"
    [[ "${HTTP_STATUS}" == "200" ]] || die "Não foi possível consultar ${path} (HTTP ${HTTP_STATUS})"
    match="$(python3 - "${response}" "${collection}" "${email}" <<'PY'
import json, sys
data = json.load(open(sys.argv[1])).get("data", {})
for item in data.get(sys.argv[2], []):
    if str(item.get("email", "")).strip().lower() == sys.argv[3].strip().lower():
        print(item.get("id", ""), item.get("user_id") or "-", item.get("status") or "-")
        break
else:
    print("pages", data.get("pagination", {}).get("total_pages", 1))
PY
)"
    if [[ "${match}" == pages* ]]; then
      total_pages="${match#pages }"
      page=$((page + 1))
      continue
    fi
    printf '%s\n' "${match}"
    return 0
  done
  return 1
}

# account_login signs a test account in (client or trainer) with the .env
# password and keeps its session in a cookie jar. Returns 1 on any failure.
account_login() {
  local kind="$1"
  local request_body="${RUNTIME_DIR}/login-${kind}-request.json"
  local response="${RUNTIME_DIR}/login-${kind}-response.json"
  rm -f -- "${RUNTIME_DIR}/cookies/${kind}.jar"
  json_file "${request_body}" "login-${kind}"
  request POST /auth/login "${request_body}" "${response}" "${RUNTIME_DIR}/cookies/${kind}.jar"
  [[ "${HTTP_STATUS}" == "200" ]]
}

# trainer_is_approved reports whether the signed-in trainer account has an
# active trainer profile.
trainer_is_approved() {
  local response="${RUNTIME_DIR}/trainer-access.json"
  request GET /trainer/access '' "${response}" "${RUNTIME_DIR}/cookies/trainer.jar"
  [[ "${HTTP_STATUS}" == "200" ]] || return 1
  python3 - "${response}" <<'PY'
import json, sys
sys.exit(0 if json.load(open(sys.argv[1])).get("data", {}).get("is_trainer") is True else 1)
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

# ensure_client makes the .env client password work. An account that already
# signs in is left untouched (its browser sessions stay valid); otherwise the
# account is registered, or its password is reset by ADMIN_1.
ensure_client() {
  local response="${RUNTIME_DIR}/client-response.json"
  local request_body="${RUNTIME_DIR}/client-request.json"
  local found=""

  if account_login client; then
    printf 'Cliente normal OK (login confirmado): %s\n' "${TEST_USER_EMAIL}"
    return 0
  fi

  json_file "${request_body}" register-client
  request POST /auth/register "${request_body}" "${response}"
  if [[ "${HTTP_STATUS}" == "201" ]]; then
    account_login client || die "Cliente criado, mas o login falhou (HTTP ${HTTP_STATUS})"
    printf 'Cliente normal criado: %s\n' "${TEST_USER_EMAIL}"
    return 0
  fi
  [[ "${HTTP_STATUS}" == "409" ]] || die "Registo do cliente falhou (HTTP ${HTTP_STATUS})"

  found="$(find_account /admin/users users "${TEST_USER_EMAIL}")" ||
    die "O email do cliente já existe mas não está entre as contas ativas (conta desativada?)"
  reset_user_password "${found%% *}" client
  account_login client || die "A password do cliente foi atualizada, mas o login continua a falhar (HTTP ${HTTP_STATUS})"
  printf 'Cliente normal reutilizado (password do .env aplicada): %s\n' "${TEST_USER_EMAIL}"
}

# approve_own_application submits a trainer application as the signed-in
# trainer account and approves it as ADMIN_1, through the normal review flow.
approve_own_application() {
  local response="${RUNTIME_DIR}/apply-response.json"
  local application_id=""

  request POST /trainer/apply '' "${response}" "${RUNTIME_DIR}/cookies/trainer.jar"
  [[ "${HTTP_STATUS}" == "201" || "${HTTP_STATUS}" == "409" ]] || die "Candidatura de treinador falhou (HTTP ${HTTP_STATUS})"
  request GET /trainer/access '' "${response}" "${RUNTIME_DIR}/cookies/trainer.jar"
  application_id="$(python3 - "${response}" <<'PY'
import json, sys
application = json.load(open(sys.argv[1])).get("data", {}).get("application") or {}
print(application.get("id", "") if application.get("status") == "PENDING" else "")
PY
)"
  [[ -n "${application_id}" ]] || die "Não foi encontrada uma candidatura pendente para aprovar"
  request POST "/admin/trainer-applications/${application_id}/approve" '' "${response}" "${RUNTIME_DIR}/cookies/admin-1.jar"
  [[ "${HTTP_STATUS}" == "200" ]] || die "Aprovação da candidatura falhou (HTTP ${HTTP_STATUS})"
}

# ensure_trainer makes the .env trainer sign in as an approved trainer: it is
# reused when it already works, otherwise reset, reactivated, approved through
# the application flow (existing client account) or created by ADMIN_1.
ensure_trainer() {
  local response="${RUNTIME_DIR}/trainer-response.json"
  local request_body="${RUNTIME_DIR}/trainer-request.json"
  local found=""
  local user_id=""
  local status=""

  if account_login trainer && trainer_is_approved; then
    printf 'Treinador aprovado OK (login confirmado): %s\n' "${TEST_TRAINER_EMAIL}"
    return 0
  fi

  if found="$(find_account /admin/trainers trainers "${TEST_TRAINER_EMAIL}")"; then
    read -r _ user_id status <<<"${found}"
    [[ "${status}" == "active" ]] || die "A conta do treinador existe, mas não está ativa (status: ${status})"
    reset_user_password "${user_id}" trainer
    printf 'Treinador aprovado reutilizado (password do .env aplicada): %s\n' "${TEST_TRAINER_EMAIL}"
  elif found="$(find_account /admin/trainers/deleted trainers "${TEST_TRAINER_EMAIL}")"; then
    read -r _ user_id _ <<<"${found}"
    request POST "/admin/trainers/${found%% *}/reactivate" '' "${response}" "${RUNTIME_DIR}/cookies/admin-1.jar"
    [[ "${HTTP_STATUS}" == "200" ]] || die "Não foi possível reativar o treinador existente (HTTP ${HTTP_STATUS})"
    reset_user_password "${user_id}" trainer
    printf 'Treinador aprovado reativado: %s\n' "${TEST_TRAINER_EMAIL}"
  elif found="$(find_account /admin/users users "${TEST_TRAINER_EMAIL}")"; then
    # The email belongs to a plain account: make it a trainer through the
    # normal application and approval flow.
    reset_user_password "${found%% *}" trainer
    account_login trainer || die "Login da conta do treinador falhou (HTTP ${HTTP_STATUS})"
    approve_own_application
    printf 'Treinador aprovado através de candidatura: %s\n' "${TEST_TRAINER_EMAIL}"
  else
    json_file "${request_body}" create-trainer
    request POST /admin/trainers "${request_body}" "${response}" "${RUNTIME_DIR}/cookies/admin-1.jar"
    [[ "${HTTP_STATUS}" == "201" ]] || die "Não foi possível criar o treinador (HTTP ${HTTP_STATUS}); não foi criada uma conta duplicada"
    printf 'Treinador aprovado criado: %s\n' "${TEST_TRAINER_EMAIL}"
  fi

  account_login trainer || die "O login do treinador continua a falhar (HTTP ${HTTP_STATUS})"
  trainer_is_approved || die "A conta do treinador entra, mas não tem acesso de treinador"
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

provision_accounts() {
  admin_login 1
  admin_login 2
  ensure_client
  ensure_trainer
  ensure_test_mode_personas client
  ensure_test_mode_personas trainer
}

main() {
  if [[ "${1:-}" == "--help" || "${1:-}" == "-h" ]]; then
    usage
    return 0
  fi
  local accounts_only=false
  if [[ "${1:-}" == "--accounts-only" ]]; then
    accounts_only=true
    shift
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

  if [[ "${accounts_only}" == true ]]; then
    url_responds "${BACKEND_URL}/api/v1/health" || die "O backend não está a responder em ${BACKEND_URL}"
    provision_accounts
    printf '\nContas de teste prontas.\n'
    return 0
  fi

  start_database

  (cd "${BACKEND_DIR}" && go run . migrate up) >"${RUNTIME_DIR}/migrations.log" 2>&1 || die "As migrações falharam; consulte ${RUNTIME_DIR}/migrations.log"
  if [[ -f "${EXERCISE_DATASET}" ]]; then
    (cd "${BACKEND_DIR}" && go run . import-exercises --file "${EXERCISE_DATASET}") >"${RUNTIME_DIR}/exercise-import.log" 2>&1 || die "A importação de exercícios falhou; consulte ${RUNTIME_DIR}/exercise-import.log"
  else
    printf 'AVISO: dataset de exercícios não encontrado; o catálogo existente será preservado.\n' >&2
  fi

  mkdir -p "${RUNTIME_DIR}"
  stop_previous_run

  # The API is built and run as a binary (not `go run`), so the PID kept here
  # is the server itself and Ctrl-C really stops it.
  if url_responds "${BACKEND_URL}/api/v1/health"; then
    die "Já existe um backend na porta 8080 que não foi iniciado por este script. Pare-o (por exemplo: fuser -k 8080/tcp) e volte a correr ./start_test_mode.sh"
  fi
  (cd "${BACKEND_DIR}" && go build -o "${RUNTIME_DIR}/ryze-backend" .) >"${RUNTIME_DIR}/build.log" 2>&1 ||
    die "A compilação do backend falhou; consulte ${RUNTIME_DIR}/build.log"
  (cd "${BACKEND_DIR}" && \
    PORT=8080 COOKIE_SECURE=false TEST_MODE_ENABLED=true \
    EXERCISE_MEDIA_ENABLED=true EXERCISE_MEDIA_BASE_PATH=/dev-exercise-media \
    exec "${RUNTIME_DIR}/ryze-backend") >"${RUNTIME_DIR}/backend.log" 2>&1 &
  BACKEND_PID=$!
  printf '%s\n' "${BACKEND_PID}" >"${RUNTIME_DIR}/backend.pid"
  wait_for_url "${BACKEND_URL}/api/v1/health" 'Backend'

  if url_responds "${FRONTEND_URL}"; then
    # A Vite dev server started from this repository reads the same .env
    # (exercise media preview included), so it is reused as is.
    printf 'Frontend já a correr em %s; a reutilizar.\n' "${FRONTEND_URL}"
  else
    (cd "${ROOT_DIR}" && \
      VITE_API_BASE_URL="${API_BASE}" \
      EXERCISE_MEDIA_ENABLED=true EXERCISE_MEDIA_BASE_PATH=/dev-exercise-media \
      exec ./node_modules/.bin/vite --host 127.0.0.1 --port 5173 --strictPort) >"${RUNTIME_DIR}/frontend.log" 2>&1 &
    FRONTEND_PID=$!
    printf '%s\n' "${FRONTEND_PID}" >"${RUNTIME_DIR}/frontend.pid"
    wait_for_url "${FRONTEND_URL}" 'Frontend'
  fi

  provision_accounts

  printf '\nRYZE Test Mode pronto.\n'
  printf 'Frontend: %s\n' "${FRONTEND_URL}"
  printf 'Backend:  %s\n' "${BACKEND_URL}"
  printf 'Contas (passwords e códigos de acesso no .env, nunca impressos):\n'
  printf '  Cliente:   %s  -> %s/login\n' "${TEST_USER_EMAIL}" "${FRONTEND_URL}"
  printf '  Treinador: %s  -> %s/login, depois /trainer\n' "${TEST_TRAINER_EMAIL}" "${FRONTEND_URL}"
  printf '  ADMIN_1 e ADMIN_2 -> %s/admin/login (utilizador, password e código de acesso)\n' "${FRONTEND_URL}"
  printf 'Ctrl-C termina o backend (e o frontend, se foi iniciado aqui) e mantém PostgreSQL ativo.\n\n'

  while kill -0 "${BACKEND_PID}" 2>/dev/null && { [[ -z "${FRONTEND_PID}" ]] || kill -0 "${FRONTEND_PID}" 2>/dev/null; }; do
    sleep 2
  done
  die 'Frontend ou backend terminou inesperadamente; consulte os logs'
}

trap cleanup EXIT INT TERM
main "$@"
