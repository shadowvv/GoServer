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

# ===== 校验关键目录 =====
if [ ! -d "$SRC_DIR/server" ]; then
    echo "❌ 找不到 server 目录"
    exit 1
fi

if [ ! -d "$SRC_DIR/gameConfig" ]; then
    echo "❌ 找不到 gameConfig 目录"
    exit 1
fi

if [ ! -d "$SRC_DIR/config" ]; then
    echo "❌ 找不到 config 目录"
    exit 1
fi

# ===== 删除旧 server =====
echo "👉 删除旧 server..."
rm -rf "$PROJECT_DIR/server"

# ===== 拷贝新 server =====
echo "👉 更新 server..."
cp -r "$SRC_DIR/server" "$PROJECT_DIR/"

# ===== 分发 gameConfig =====
echo "👉 分发 gameConfig（自动遍历 SERVER_DIR）..."

CONFIG_SRC="$SRC_DIR/gameConfig"
CONFIG_CAP_SRC="$SRC_DIR/config"

for dir in "$SERVER_DIR"/*; do
    if [ -d "$dir" ]; then
        echo "处理: $dir"

        rm -rf "$dir/gameConfig"
        cp -r "$CONFIG_SRC" "$dir/"

        cp -r "$CONFIG_CAP_SRC/." "$dir/config/"
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

echo "===== ✅ 更新完成 ====="