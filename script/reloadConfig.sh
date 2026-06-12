#!/bin/bash

set -e
set -o pipefail

BASE_DIR=$(cd "$(dirname "$0")"; pwd)
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
    rm -rf "$TMP_DIR"
    exit 1
fi

# ===== 自动识别结构 =====
echo "👉 判断压缩结构..."

SUB_DIR_COUNT=$(find "$TMP_DIR" -mindepth 1 -maxdepth 1 -type d | wc -l)

if [ "$SUB_DIR_COUNT" -eq 1 ]; then
    INNER_DIR=$(find "$TMP_DIR" -mindepth 1 -maxdepth 1 -type d)

    if [ -d "$INNER_DIR/gameConfig" ] || [ -d "$INNER_DIR/config" ]; then
        SRC_DIR="$INNER_DIR"
    else
        SRC_DIR="$TMP_DIR"
    fi
else
    SRC_DIR="$TMP_DIR"
fi

echo "使用源目录: $SRC_DIR"

# ===== 校验更新包 =====
if [ ! -d "$SRC_DIR/gameConfig" ]; then
    echo "❌ 找不到 gameConfig 目录"
    rm -rf "$TMP_DIR"
    exit 1
fi

if [ ! -d "$SRC_DIR/config" ]; then
    echo "❌ 找不到 config 目录"
    rm -rf "$TMP_DIR"
    exit 1
fi

if [ ! -f "$SRC_DIR/config/gameConfigVersion" ]; then
    echo "❌ 找不到 config/gameConfigVersion"
    rm -rf "$TMP_DIR"
    exit 1
fi

PACKAGE_GAME_CONFIG_VERSION=$(cat "$SRC_DIR/config/gameConfigVersion")

echo "===== 📦 本次热更包版本 ====="
echo "gameConfigVersion: $PACKAGE_GAME_CONFIG_VERSION"

# ===== 更新前检查本地 gameConfig 版本一致性 =====
echo "===== 🔍 更新前检查本地服务器 gameConfig 版本一致性 ====="

BASE_GAME_CONFIG_VERSION=""
BASE_SERVER_NAME=""

HAS_VERSION_MISMATCH=0
HAS_ANY_SERVER=0

for dir in "$SERVER_DIR"/*; do
    if [ -d "$dir" ]; then
        SERVER_NAME=$(basename "$dir")
        HAS_ANY_SERVER=1

        GAME_CONFIG_VERSION_FILE="$dir/config/gameConfigVersion"

        if [ -f "$GAME_CONFIG_VERSION_FILE" ]; then
            SERVER_GAME_CONFIG_VERSION=$(cat "$GAME_CONFIG_VERSION_FILE")
        else
            SERVER_GAME_CONFIG_VERSION="MISSING"
            HAS_VERSION_MISMATCH=1
        fi

        # 第一个服务器作为本地基准版本
        if [ -z "$BASE_SERVER_NAME" ]; then
            BASE_SERVER_NAME="$SERVER_NAME"
            BASE_GAME_CONFIG_VERSION="$SERVER_GAME_CONFIG_VERSION"
        fi

        VERSION_STATUS="✅ 一致"

        if [ "$SERVER_GAME_CONFIG_VERSION" != "$BASE_GAME_CONFIG_VERSION" ]; then
            VERSION_STATUS="⚠️ 不一致"
            HAS_VERSION_MISMATCH=1
        fi

        if [ "$SERVER_GAME_CONFIG_VERSION" = "MISSING" ]; then
            VERSION_STATUS="⚠️ 版本文件缺失"
            HAS_VERSION_MISMATCH=1
        fi

        echo "----------------------------------------"
        echo "服务: $SERVER_NAME"
        echo "路径: $dir"
        echo "gameConfigVersion: $SERVER_GAME_CONFIG_VERSION"
        echo "状态: $VERSION_STATUS"
    fi
done

echo "----------------------------------------"

if [ "$HAS_ANY_SERVER" -eq 0 ]; then
    echo "❌ $SERVER_DIR 下没有找到任何服务器目录"
    rm -rf "$TMP_DIR"
    exit 1
fi

echo "本地基准服务: ${BASE_SERVER_NAME:-NONE}"
echo "本地基准 gameConfigVersion: ${BASE_GAME_CONFIG_VERSION:-NONE}"

if [ "$HAS_VERSION_MISMATCH" -eq 1 ]; then
    echo
    echo "⚠️ 检测到本地服务器之间 gameConfigVersion 不一致或版本文件缺失。"
    echo "⚠️ 本次热更包 gameConfigVersion: $PACKAGE_GAME_CONFIG_VERSION"
    echo "⚠️ 继续更新会覆盖所有服务器的 gameConfig 和 config，并发送 SIGHUP 热更新。"
    echo

    read -r -p "是否继续更新？请输入 yes 继续，其他输入取消: " CONFIRM_CONTINUE

    if [ "$CONFIRM_CONTINUE" != "yes" ]; then
        echo "❌ 用户取消更新"
        rm -rf "$TMP_DIR"
        exit 1
    fi

    echo "✅ 用户确认继续更新"
else
    echo "✅ 本地所有服务器 gameConfigVersion 一致，继续更新"
fi

# ===== 分发 gameConfig + config =====
echo "👉 分发 gameConfig 和 config（热更新）..."

GAME_CONFIG_SRC="$SRC_DIR/gameConfig"
CONFIG_SRC="$SRC_DIR/config"

for dir in "$SERVER_DIR"/*; do
    if [ -d "$dir" ]; then
        echo "处理: $dir"

        rm -rf "$dir/gameConfig"
        cp -r "$GAME_CONFIG_SRC" "$dir/"

        mkdir -p "$dir/config"
        cp -r "$CONFIG_SRC/." "$dir/config/"
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

    sleep 0.2

    if ps -p "$pid" > /dev/null 2>&1; then
        echo "[OK] reload signal sent"
    else
        echo "[ERROR] process died after reload!"
    fi
done

echo "-----------------------------------"
echo "✅ 所有服务热更新完成"