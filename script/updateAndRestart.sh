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

echo "===== 🚀 开始更新 ====="
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

    if [ -d "$INNER_DIR/server" ] || [ -d "$INNER_DIR/gameConfig" ] || [ -d "$INNER_DIR/config" ]; then
        SRC_DIR="$INNER_DIR"
    else
        SRC_DIR="$TMP_DIR"
    fi
else
    SRC_DIR="$TMP_DIR"
fi

echo "使用源目录: $SRC_DIR"

# ===== 校验关键目录 =====
if [ ! -d "$SRC_DIR/server" ]; then
    echo "❌ 找不到 server 目录"
    rm -rf "$TMP_DIR"
    exit 1
fi

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

# ===== 更新前检查本地服务器版本一致性 =====
echo "===== 🔍 更新前检查本地服务器版本一致性 ====="

BASE_CODE_VERSION=""
BASE_GAME_CONFIG_VERSION=""
BASE_SERVER_NAME=""

HAS_VERSION_MISMATCH=0
HAS_VERSION_MISSING=0
HAS_ANY_SERVER=0

for dir in "$SERVER_DIR"/*; do
    if [ -d "$dir" ]; then
        SERVER_NAME=$(basename "$dir")
        HAS_ANY_SERVER=1

        CODE_VERSION_FILE="$dir/config/codeVersion"
        GAME_CONFIG_VERSION_FILE="$dir/config/gameConfigVersion"

        if [ -f "$CODE_VERSION_FILE" ]; then
            SERVER_CODE_VERSION=$(cat "$CODE_VERSION_FILE")
        else
            SERVER_CODE_VERSION="MISSING"
            HAS_VERSION_MISSING=1
        fi

        if [ -f "$GAME_CONFIG_VERSION_FILE" ]; then
            SERVER_GAME_CONFIG_VERSION=$(cat "$GAME_CONFIG_VERSION_FILE")
        else
            SERVER_GAME_CONFIG_VERSION="MISSING"
            HAS_VERSION_MISSING=1
        fi

        # 第一个服务器作为本地基准版本
        if [ -z "$BASE_SERVER_NAME" ]; then
            BASE_SERVER_NAME="$SERVER_NAME"
            BASE_CODE_VERSION="$SERVER_CODE_VERSION"
            BASE_GAME_CONFIG_VERSION="$SERVER_GAME_CONFIG_VERSION"
        fi

        VERSION_STATUS="✅ 一致"

        if [ "$SERVER_CODE_VERSION" != "$BASE_CODE_VERSION" ] || [ "$SERVER_GAME_CONFIG_VERSION" != "$BASE_GAME_CONFIG_VERSION" ]; then
            VERSION_STATUS="⚠️ 不一致"
            HAS_VERSION_MISMATCH=1
        fi

        if [ "$SERVER_CODE_VERSION" = "MISSING" ] || [ "$SERVER_GAME_CONFIG_VERSION" = "MISSING" ]; then
            VERSION_STATUS="⚠️ 版本文件缺失"
            HAS_VERSION_MISMATCH=1
        fi

        echo "----------------------------------------"
        echo "服务: $SERVER_NAME"
        echo "路径: $dir"
        echo "codeVersion: $SERVER_CODE_VERSION"
        echo "gameConfigVersion: $SERVER_GAME_CONFIG_VERSION"
        echo "状态: $VERSION_STATUS"
    fi
done

echo "----------------------------------------"

if [ "$HAS_ANY_SERVER" -eq 0 ]; then
    echo "⚠️ $SERVER_DIR 下没有找到任何服务器目录"
fi

echo "本地基准服务: ${BASE_SERVER_NAME:-NONE}"
echo "本地基准 codeVersion: ${BASE_CODE_VERSION:-NONE}"
echo "本地基准 gameConfigVersion: ${BASE_GAME_CONFIG_VERSION:-NONE}"

if [ "$HAS_VERSION_MISMATCH" -eq 1 ]; then
    echo
    echo "⚠️ 检测到本地服务器之间版本不一致或版本文件缺失。"
    echo "⚠️ 继续更新会覆盖 server、gameConfig、config，并重启所有服务。"
    echo

    read -r -p "是否继续更新？请输入 yes 继续，其他输入取消: " CONFIRM_CONTINUE

    if [ "$CONFIRM_CONTINUE" != "yes" ]; then
        echo "❌ 用户取消更新"
        rm -rf "$TMP_DIR"
        exit 1
    fi

    echo "✅ 用户确认继续更新"
else
    echo "✅ 本地所有服务器版本一致，继续更新"
fi

# ===== 删除旧 server =====
echo "👉 删除旧 server..."
rm -rf "$PROJECT_DIR/server"

# ===== 拷贝新 server =====
echo "👉 更新 server..."
cp -r "$SRC_DIR/server" "$PROJECT_DIR/"

# ===== 分发 gameConfig 和 config =====
echo "👉 分发 gameConfig 和 config（自动遍历 SERVER_DIR）..."

CONFIG_SRC="$SRC_DIR/gameConfig"
CONFIG_COMMON_SRC="$SRC_DIR/config"

for dir in "$SERVER_DIR"/*; do
    if [ -d "$dir" ]; then
        echo "处理: $dir"

        rm -rf "$dir/gameConfig"
        cp -r "$CONFIG_SRC" "$dir/"

        mkdir -p "$dir/config"
        cp -r "$CONFIG_COMMON_SRC/." "$dir/config/"
    fi
done

# ===== 构建 =====
echo "👉 执行 build.sh..."
cd "$PROJECT_DIR"

chmod +x build.sh
./build.sh

# ===== 清理 =====
echo "👉 清理临时文件..."
rm -rf "$TMP_DIR"

echo "===== ✅ 更新完成，准备启动服务器 ====="

echo "👉 重启所有服务..."

for dir in "$SERVER_DIR"/*; do
    if [ -d "$dir" ]; then
        if [ -f "$dir/booter.sh" ]; then
            echo "重启服务: $dir"

            cd "$dir"

            chmod +x booter.sh
            ./booter.sh restart

        else
            echo "⏭️ 跳过（无 booter.sh）: $dir"
        fi
    fi
done

echo "✅ 所有服务重启完成"