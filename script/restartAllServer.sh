#!/bin/bash

set -e
set -o pipefail

BASE_DIR=$(cd "$(dirname "$0")"; pwd)
SERVER_DIR="$BASE_DIR/../server"

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