#!/usr/bin/env bash
#
# Loads the repository .env file without overwriting values already exported by
# the caller. Recovery procedures must be able to target a throwaway database
# with explicit environment variables even when the repository .env points at
# the real one.

load_project_env() {
  local root_dir="$1"
  shift

  local env_file="${root_dir}/.env"
  if [[ ! -f "${env_file}" ]]; then
    return 0
  fi

  local -A preserved=()
  local key
  for key in "$@"; do
    if [[ -n "${!key+x}" ]]; then
      preserved["${key}"]="${!key}"
    fi
  done

  set -a
  # shellcheck disable=SC1091
  . "${env_file}"
  set +a

  for key in "${!preserved[@]}"; do
    export "${key}=${preserved[${key}]}"
  done
}
