#!/bin/sh
set -e

# Fix ownership on the bind-mounted data dir so the unprivileged
# app user can write app.db and icons/. Idempotent — a no-op when
# ownership is already correct.
chown -R app:app /data

exec gosu app "$@"
