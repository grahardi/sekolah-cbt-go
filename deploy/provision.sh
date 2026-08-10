#!/usr/bin/env bash
# Provisions one sekolah's cbt-server instance: allocates a free port,
# generates a JWT secret, creates the schema + runs migrations, and drops a
# ready .env + binary copy into cbt-instances/{sekolah_id}/.
#
# Called by Laravel via a Process call (same pattern as the existing
# Extraordinary CBT auto-provisioning), not meant to be run interactively.
#
# Deliberately does NOT touch systemd or Nginx — those stay manual via the
# sekolah.co.id panel for now, same as Extraordinary CBT's Run/Stop today.
#
# Usage:
#   DB_PASSWORD=... ./provision.sh <sekolah_id> "<sekolah_nama>" <db_host> <db_port> <db_user> <db_name>
#
# DB_PASSWORD is read from the environment, not a CLI arg, so it never
# shows up in `ps aux` or shell history — same principle already used for
# the Postgres root password in the Extraordinary CBT provisioning flow:
# passed in at call time, never stored in this script.
#
# On success, prints a single line of JSON to stdout:
#   {"sekolah_id":"...","port":13001,"jwt_secret":"...","schema":"cbt_...","instance_dir":"..."}
# Laravel is expected to persist port + jwt_secret + schema against the
# sekolah record — this script itself doesn't write them anywhere else.
# jwt_secret in particular matters: it's what Laravel signs admin JWTs with
# for this sekolah's /admin/* API from now on, so it has to be captured.

set -euo pipefail

SEKOLAH_ID="${1:?usage: provision.sh <sekolah_id> <sekolah_nama> <db_host> <db_port> <db_user> <db_name>}"
SEKOLAH_NAMA="${2:?}"
DB_HOST="${3:?}"
DB_PORT="${4:?}"
DB_USER="${5:?}"
DB_NAME="${6:?}"
DB_PASSWORD="${DB_PASSWORD:?DB_PASSWORD env var wajib diisi}"

# sekolah_id jadi bagian dari nama schema Postgres dan path direktori, jadi
# divalidasi ketat: huruf kecil, angka, underscore saja.
if ! [[ "$SEKOLAH_ID" =~ ^[a-z0-9_]+$ ]]; then
  echo "error: sekolah_id harus huruf kecil/angka/underscore saja, dapat: $SEKOLAH_ID" >&2
  exit 1
fi

SRC_DIR="/www/wwwroot/sekolah.co.id/cbt-src/sekolah-cbt-go"
INSTANCES_DIR="/www/wwwroot/sekolah.co.id/cbt-instances"
INSTANCE_DIR="$INSTANCES_DIR/$SEKOLAH_ID"
DB_SCHEMA="cbt_${SEKOLAH_ID}"
PORT_RANGE_START=13000
PORT_RANGE_END=14000

echo "provisioning $SEKOLAH_NAMA ($SEKOLAH_ID)..." >&2

if [ -d "$INSTANCE_DIR" ]; then
  echo "error: instance $SEKOLAH_ID sudah ada di $INSTANCE_DIR" >&2
  exit 1
fi

if [ ! -x "$SRC_DIR/cbt-server" ]; then
  echo "error: binary belum di-build, jalankan 'make build' di $SRC_DIR dulu" >&2
  exit 1
fi

# cari port bebas: kumpulin PORT yang udah dipakai instance lain, pilih yang
# pertama kosong di range
used_ports=$(grep -h '^PORT=' "$INSTANCES_DIR"/*/.env 2>/dev/null | cut -d= -f2 || true)
port=""
for candidate in $(seq "$PORT_RANGE_START" "$PORT_RANGE_END"); do
  if ! grep -qx "$candidate" <<< "$used_ports"; then
    port="$candidate"
    break
  fi
done
if [ -z "$port" ]; then
  echo "error: tidak ada port bebas di range $PORT_RANGE_START-$PORT_RANGE_END" >&2
  exit 1
fi

jwt_secret=$(openssl rand -hex 32)

mkdir -p "$INSTANCE_DIR"
cp "$SRC_DIR/cbt-server" "$INSTANCE_DIR/cbt-server"

cat > "$INSTANCE_DIR/.env" <<ENVEOF
DB_HOST=$DB_HOST
DB_PORT=$DB_PORT
DB_USER=$DB_USER
DB_PASSWORD=$DB_PASSWORD
DB_NAME=$DB_NAME
DB_SCHEMA=$DB_SCHEMA
PORT=$port
JWT_SECRET=$jwt_secret
SEKOLAH_ID=$SEKOLAH_ID
ENVEOF
chmod 600 "$INSTANCE_DIR/.env"

echo "running migrations for $DB_SCHEMA..." >&2
(cd "$INSTANCE_DIR" && ./cbt-server migrate) >&2

printf '{"sekolah_id":"%s","port":%s,"jwt_secret":"%s","schema":"%s","instance_dir":"%s"}\n' \
  "$SEKOLAH_ID" "$port" "$jwt_secret" "$DB_SCHEMA" "$INSTANCE_DIR"
