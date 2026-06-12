#!/bin/bash

set -e
set -o pipefail

BASE_DIR=$(pwd)
SERVER_DIR="$BASE_DIR/../server"

# 需要跳过的服务
SKIP_SERVICES=("httpServer" "backendServer")

is_skip() {
    local service_name=$1

    for skip in "${SKIP_SERVICES[@]}"; do
        if [[ "$service_name" == "$skip" ]]; then
            return 0
        fi
    done

    return 1
}

echo "👉 开始停止服务..."

for dir in "$SERVER_DIR"/*; do
    [ -d "$dir" ] || continue

    service_name=$(basename "$dir")

    # 跳过指定服务
    if is_skip "$service_name"; then
        echo "⏭️  跳过服务: $service_name"
        continue
    fi

    if [ -f "$dir/booter.sh" ]; then
        echo "🛑 停止服务: $service_name"

        cd "$dir"

        chmod +x booter.sh

        if ./booter.sh stop; then
            echo "✅ $service_name 停止成功"
        else
            echo "❌ $service_name 停止失败"
        fi
    else
        echo "⏭️  跳过（无 booter.sh）: $service_name"
    fi
done

echo "🎉 服务停止完成"