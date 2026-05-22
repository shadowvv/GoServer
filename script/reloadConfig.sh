#!/bin/bash

set -e
set -o pipefail

BASE_DIR=$(cd "$(dirname "$0")"; pwd)
PROJECT_DIR="$BASE_DIR/../project"
SERVER_DIR="$BASE_DIR/../server"
TMP_DIR="$BASE_DIR/tmp_update_$(date +%s)"
ZIP_FILE=$1

if [ -z "$ZIP_FILE" ]; then
    echo "❌ 请传入压缩包"
    exit 1
fi

if [ ! -f "$ZIP_FILE" ]; then
    echo "❌ 文件不存在: $ZIP_FILE"
    exit 1
fi

echo "===== 🚀 开始更新（热更新模式） ====="
echo "压缩包: $ZIP_FILE"

# ===== 解压 =====
echo "👉 解压文件..."
mkdir -p "$TMP_DIR"

if [[ "$ZIP_FILE" == *.zip ]]; then
    unzip -q "$ZIP_FILE" -d "$TMP_DIR"
elif [[ "$ZIP_FILE" == *.tar.gz ]]; then
    tar -xzf "$ZIP_FILE" -C "$TMP_DIR"
elif [[ "$ZIP_FILE" == *.tar ]]; then
    tar -xf "$ZIP_FILE" -C "$TMP_DIR"
else
    echo "❌ 不支持的压缩格式"
    exit 1
fi

# ===== 自动识别结构 =====
echo "👉 判断压缩结构..."

SUB_DIR_COUNT=$(find "$TMP_DIR" -mindepth 1 -maxdepth 1 -type d | wc -l)

if [ "$SUB_DIR_COUNT" -eq 1 ]; then
    INNER_DIR=$(find "$TMP_DIR" -mindepth 1 -maxdepth 1 -type d)
    if [ -d "$INNER_DIR/server" ]; then
        SRC_DIR="$INNER_DIR"
    else
        SRC_DIR="$TMP_DIR"
    fi
else
    SRC_DIR="$TMP_DIR"
fi

echo "使用源目录: $SRC_DIR"

# ===== 校验 =====
if [ ! -d "$SRC_DIR/gameConfig" ]; then
    echo "❌ 找不到 gameConfig 目录"
    exit 1
fi

# ===== 分发 gameConfig =====
echo "👉 分发 gameConfig（热更新）..."

CONFIG_SRC="$SRC_DIR/gameConfig"

for dir in "$SERVER_DIR"/*; do
    if [ -d "$dir" ]; then
        echo "处理: $dir"

        rm -rf "$dir/gameConfig"
        cp -r "$CONFIG_SRC" "$dir/"
    fi
done

# ===== 清理 =====
echo "👉 清理临时文件..."
rm -rf "$TMP_DIR"

echo "===== ✅ 配置更新完成 ====="

# ===== 热更新（SIGHUP）=====
echo "👉 发送 SIGHUP 热更新配置..."

for dir in "$SERVER_DIR"/*/; do
    [ -d "$dir" ] || continue

    app_name=$(basename "$dir")
    pid_file="${dir}${app_name}.pid"

    echo "-----------------------------------"
    echo "[INFO] app: $app_name"

    if [ ! -f "$pid_file" ]; then
        echo "[WARN] pid file not found: $pid_file"
        continue
    fi

    pid=$(tr -d '[:space:]' < "$pid_file")

    if ! ps -p "$pid" > /dev/null 2>&1; then
        echo "[WARN] process not running (pid=$pid)"
        continue
    fi

    echo "[INFO] reload $app_name (pid=$pid)"
    kill -SIGHUP "$pid"

    # 可选：确认进程还活着
    sleep 0.2
    if ps -p "$pid" > /dev/null 2>&1; then
        echo "[OK] reload signal sent"
    else
        echo "[ERROR] process died after reload!"
    fi

done

echo "-----------------------------------"
echo "✅ 所有服务热更新完成"