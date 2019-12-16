#!/bin/bash

set -e

declare -a as_opts

# Always bind to 0.0.0.0 under docker
(env | grep --quiet ^app_search.listen_host) || as_opts+=("app_search.listen_host: 0.0.0.0")

# Loop through environment and append matching keys and values to our options array
while IFS='=' read -r envvar_key envvar_value
do
  if [[ "$envvar_key" =~ ^(app_search|email|elasticsearch)\.[a-z0-9_]+ || "$envvar_key" =~ ^(allow_es_settings_modification|diagnostic_report_directory|disable_es_settings_checks|filebeat_log_directory|hide_version_info|log_directory|log_level|secret_session_key) ]]; then
    if [[ -n $envvar_value ]]; then
      opt="${envvar_key}: ${envvar_value}"
      as_opts+=("${opt}")
    fi
  fi
done < <(env)

# Only override config file if it doesn't exist or it's all commented lines (the default).
# This is so if a user mounts their own config file, we will leave it alone
if [[ ! -f /usr/share/app-search/config/app-search.yml ]] || ! grep -q -v '^\s*#' /usr/share/app-search/config/app-search.yml; then
  printf '%s\n' "${as_opts[@]}" | sort > /usr/share/app-search/config/app-search.yml
fi

until curl -f "http://elasticsearch:9200/_license"; do
  echo "Elasticsearch not available yet".
  sleep 1
done

/usr/share/app-search/bin/app-search "$@"
