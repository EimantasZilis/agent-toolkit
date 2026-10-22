# Shared catalog contract: stable name, category, and source key (tab-separated).

catalog_records() {
  local source key
  while IFS= read -r source; do
    key=$(skill_key "$source")
    printf '%s\t%s\t%s\n' "${source##*/}" "${key%%/*}" "$key"
  done <<< "$SKILL_CATALOG"
}

bundle_names() {
  {
    printf 'all\n'
    catalog_records | cut -f2
    awk '!/^#/ && NF { print $1 }' "$AGENT_TOOLKIT_HOME/.agents/bundles.txt"
  } | sort -u
}

bundle_members() {
  local bundle=$1
  case "$bundle" in
    all) catalog_records | cut -f1 ;;
    *)
      if catalog_records | cut -f2 | grep -Fxq -- "$bundle"; then
        catalog_records | awk -F '\t' -v category="$bundle" '$2 == category { print $1 }'
      else
        awk -v bundle="$bundle" '$1 == bundle { for (i = 2; i <= NF; i++) print $i }' \
          "$AGENT_TOOLKIT_HOME/.agents/bundles.txt"
      fi
      ;;
  esac
}

validate_catalog() {
  local name category key members member seen=''
  local -a items
  while IFS=$'\t' read -r name category key; do
    [[ "$name" =~ ^[a-z0-9]+(-[a-z0-9]+)*$ && "$category" =~ ^[a-z0-9]+(-[a-z0-9]+)*$ ]] || {
      printf 'Error: invalid catalog identity: %s\n' "$key" >&2; return 1;
    }
    skill_source "$name" >/dev/null || return 1
  done < <(catalog_records)

  [[ -r "$AGENT_TOOLKIT_HOME/.agents/bundles.txt" ]] || {
    printf 'Error: missing bundle registry\n' >&2; return 1;
  }
  while read -r name members; do
    [[ -z "$name" || "$name" == \#* ]] && continue
    if [[ ! "$name" =~ ^[a-z0-9]+(-[a-z0-9]+)*$ || "$name" == all ]] ||
      catalog_records | cut -f2 | grep -Fxq -- "$name" ||
      grep -Fxq -- "$name" <<< "$seen"; then
      printf 'Error: invalid or duplicate bundle: %s\n' "$name" >&2; return 1
    fi
    [[ -n "$members" ]] || { printf 'Error: empty bundle: %s\n' "$name" >&2; return 1; }
    seen+="$name"$'\n'
    read -r -a items <<< "$members"
    for member in "${items[@]}"; do
      skill_source "$member" >/dev/null || return 1
    done
  done < "$AGENT_TOOLKIT_HOME/.agents/bundles.txt"
  grep -Fxq starter <<< "$seen" || { printf 'Error: missing starter bundle\n' >&2; return 1; }
}
