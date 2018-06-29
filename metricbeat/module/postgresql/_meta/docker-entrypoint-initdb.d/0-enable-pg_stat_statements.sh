#!/usr/bin/env bash
echo -e "shared_preload_libraries = 'pg_stat_statements'\npg_stat_statements.max = 10000\npg_stat_statements.track = all" >> $PGDATA/postgresql.conf