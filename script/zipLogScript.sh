#!/usr/bin/env bash
set -euo pipefail

BASE_DIR=$(cd "$(dirname "$0")"; pwd)
SERVER_BASE="$BASE_DIR/../server"
BACKUP_DIR="/data/logs"
DATE=$(date -d "yesterday" +"%Y-%m-%d")

mkdir -p "$BACKUP_DIR"

echo "==> archive start, target date: $DATE"

for SERVER_PATH in "$SERVER_BASE"/*; do
    [ -d "$SERVER_PATH" ] || continue

    SERVER_NAME=$(basename "$SERVER_PATH")
    LOG_DIR="$SERVER_PATH/logs"
    OUTPUT_FILE="${BACKUP_DIR}/${SERVER_NAME}_${DATE}.tar.gz"

    if [ ! -d "$LOG_DIR" ]; then
        echo "skip ${SERVER_NAME}, no logs dir"
        continue
    fi

    if [ -f "$OUTPUT_FILE" ]; then
        echo "skip ${SERVER_NAME}, archive already exists: $OUTPUT_FILE"
        continue
    fi

    TMP_FILE=$(mktemp)

    find "$LOG_DIR" -maxdepth 1 -type f \( -name "*-${DATE}.log" -o -name "*-${DATE}.log.*" \) > "$TMP_FILE"

    if [ ! -s "$TMP_FILE" ]; then
        echo "skip ${SERVER_NAME}, no log files for ${DATE}"
        rm -f "$TMP_FILE"
        continue
    fi

    echo "==> archiving ${SERVER_NAME} -> ${OUTPUT_FILE}"

    if tar -czf "$OUTPUT_FILE" -C "$LOG_DIR" --files-from="$TMP_FILE"; then
        while IFS= read -r file; do
            rm -f "$file"
        done < "$TMP_FILE"
        echo "==> archive success and deleted source logs for ${SERVER_NAME}"
    else
        echo "==> archive failed for ${SERVER_NAME}, source logs kept"
    fi

    rm -f "$TMP_FILE"
done

echo "==> archive finished"