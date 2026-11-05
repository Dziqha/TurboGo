#!/usr/bin/env bash
set -e
echo "==> init cluster jika belum ada"
if ! sudo -u postgres test -f /var/lib/postgres/data/PG_VERSION; then
  sudo -u postgres initdb --locale=C.UTF-8 -E UTF8 -D /var/lib/postgres/data
fi
echo "==> init cluster + service"
sudo systemctl start postgresql
sudo -u postgres createuser -s "$USER" || true
echo "==> setup DB (benchmark/hello_world)"
sudo -u postgres psql -f "$(dirname "$0")/setup.sql" || true
echo "==> verify"
pg_isready -h localhost
psql "postgres://benchmark:benchmark@localhost:5432/hello_world" -c "SELECT count(*) FROM world; SELECT count(*) FROM fortune;"