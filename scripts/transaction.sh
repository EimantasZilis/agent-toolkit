# Filesystem transactions shared by additive and managed installation.

path_exists() { [[ -e "$1" || -L "$1" ]]; }

remove_path() {
  if [[ -d "$1" && ! -L "$1" ]]; then
    rm -rf -- "$1"
  else
    rm -f -- "$1"
  fi
}

transaction_begin() {
  transaction_root="$(mktemp -d "${TMPDIR:-/tmp}/agent-toolkit-install.XXXXXX")"
  transaction_active=yes
  backup_count=0
  backup_paths=()
  backup_slots=()
  backup_modes=()
  backup_states=()
  trap transaction_cleanup EXIT
  trap 'exit 130' INT
  trap 'exit 143' TERM
}

backup_path() {
  local path=$1 mode=${2:-move} slot="$transaction_root/$backup_count"
  backup_paths[$backup_count]=$path
  backup_slots[$backup_count]=$slot
  backup_modes[$backup_count]=$mode
  backup_states[$backup_count]=pending
  if path_exists "$path"; then
    if [[ "$mode" == copy ]]; then
      cp -p -- "$path" "$slot"
    else
      mv -- "$path" "$slot"
    fi
    backup_states[$backup_count]=present
  else
    backup_states[$backup_count]=absent
  fi
  backup_count=$((backup_count + 1))
}

rollback_transaction() {
  local restore_failed=no i path slot mode state
  for ((i = backup_count - 1; i >= 0; i--)); do
    path=${backup_paths[$i]}
    slot=${backup_slots[$i]}
    mode=${backup_modes[$i]}
    state=${backup_states[$i]}
    [[ "$state" == pending ]] && continue
    if [[ "$state" == present ]]; then
      if ! path_exists "$slot" || ! remove_path "$path"; then
        restore_failed=yes
        continue
      fi
      if [[ "$mode" == copy ]]; then
        cp -p -- "$slot" "$path" || restore_failed=yes
      else
        mv -- "$slot" "$path" || restore_failed=yes
      fi
    else
      remove_path "$path" || restore_failed=yes
    fi
  done
  if [[ "$restore_failed" == yes ]]; then
    printf 'Error: rollback incomplete; recovery backups retained at %s\n' "$transaction_root" >&2
    return 1
  fi
  rm -rf -- "$transaction_root"
}

transaction_cleanup() {
  local status=$?
  trap - EXIT INT TERM
  if [[ "${transaction_active:-no}" == yes ]]; then
    if ! rollback_transaction; then
      [[ "$status" -ne 0 ]] || status=1
    fi
  fi
  if [[ "${managed_lock_held:-no}" == yes ]]; then
    if ! rmdir -- "$managed_state_dir/apply.lock"; then
      [[ "$status" -ne 0 ]] || status=1
    fi
  fi
  if declare -F managed_protocol_failure >/dev/null; then managed_protocol_failure "$status"; fi
  return "$status"
}

transaction_commit() {
  transaction_active=no
  rm -rf -- "$transaction_root"
  if [[ "${managed_lock_held:-no}" == yes ]]; then
    rmdir -- "$managed_state_dir/apply.lock"
    managed_lock_held=no
  fi
}
