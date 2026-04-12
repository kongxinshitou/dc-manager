#!/bin/sh
set -e

DATA_DIR="/app/data"
DB_FILE="${DB_PATH:-/app/data/dc_manager.db}"
UPLOAD_DIR="${UPLOAD_DIR:-/app/uploads}"

# 首次启动时，将初始数据库复制到持久化 volume
mkdir -p "$DATA_DIR"
mkdir -p "$UPLOAD_DIR/inspections"
if [ ! -f "$DB_FILE" ]; then
    echo "[entrypoint] 首次启动，初始化数据库..."
    cp /app/dc_manager.db.init "$DB_FILE"
    echo "[entrypoint] 数据库初始化完成（1350 条设备数据）"
else
    echo "[entrypoint] 使用已有数据库: $DB_FILE"
fi

exec ./dcmanager
