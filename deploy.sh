#!/usr/bin/env bash
set -euo pipefail

# --- Load project config ---
if [[ ! -f "./deploy.conf" ]]; then
  echo "[ERROR] No deploy.conf found in project directory."
  exit 1
fi
source ./deploy.conf

# --- Build backend ---
echo "[INFO] Building production Go binary..."
make prod

# --- Build frontend (Vue) ---
#echo "[INFO] Preparing Vue client build..."

# Step 1: Remove symlink if it exists
#if [[ -L "$CLIENT_LIB_LINK" ]]; then
#  echo "[INFO] Removing symlink: $CLIENT_LIB_LINK"
#  rm "$CLIENT_LIB_LINK"
#fi

# Step 2: Copy actual library folder
#echo "[INFO] Copying real library from $CLIENT_LIB_REAL"
#cp -r "$CLIENT_LIB_REAL" "$CLIENT_LIB_LINK"

# --- Build frontend (Vue) ---
#echo "[INFO] Building Vue client..."
#pushd "$CLIENT_DIR" >/dev/null
#$CLIENT_BUILD_CMD
#popd >/dev/null

# Step 4: Remove the copied library folder
#if [[ -d "$CLIENT_LIB_LINK" ]]; then
#  echo "[INFO] Removing real library folder"
#  rm -rf "$CLIENT_LIB_LINK"
#fi

# Step 5: Restore symlink
#echo "[INFO] Restoring symlink"
#ln -s "$CLIENT_LIB_REAL" "$CLIENT_LIB_LINK"

# --- Get last update timestamp from DB ---
# echo "[INFO] Retrieving last update timestamp..."
# LAST_UPDATE=$(PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -t -c \
#     "SELECT COALESCE(MAX(updated_at), '1970-01-01 00:00:00') FROM ${LAST_UPDATE_TABLE};")
# LAST_UPDATE=$(echo "$LAST_UPDATE" | xargs)
# echo "[INFO] Last update: $LAST_UPDATE"

# --- Iterate over hosts ---
for HOST in "${HOSTS[@]}"; do
  echo "[INFO] Deploying to $HOST"

  ssh "$HOST" "SUDO_ASKPASS=./pass sudo -A systemctl stop ${APP_NAME}.service"

  rsync -az "$LOCAL_BUILD_DIR/$APP_NAME" "$HOST:$APP_DIR/"
  #rsync -az "$MIGRATIONS_DIR/" "$HOST:$APP_DIR/migrations/"
  #rsync -az "$SQL_DIR/" "$HOST:$APP_DIR/migrations/sql/"
  
  #rsync -az "$CLIENT_DIST_DIR/" "$HOST:$APP_DIR/dist/"

  #ssh "$HOST" "cd $APP_DIR && source <(grep DB_CONN .env) && migrate -path ./migrations -database \$DB_CONN up"

  ssh "$HOST" "SUDO_ASKPASS=./pass sudo -A systemctl start ${APP_NAME}.service"
done

echo "[INFO] Deployment completed successfully."

