#!/usr/bin/env bash
set -euo pipefail

BASE_DIR=$(cd "$(dirname "$0")"; pwd)
PROJECT_SERVER_DIR="$BASE_DIR/server/main"
DEPLOY_BASE="$BASE_DIR/../server"

GOOS=linux
GOARCH=amd64
CGO_ENABLED=0

BUILD_TIME="$(date +%s)"

TRIMPATH="-trimpath"
GCFLAGS=""
ASMFLAGS=""
RACE_FLAG=""
BUILD_TAGS=""
LDFLAGS=""

usage() {
    echo "用法: $0"
    echo "说明: 自动读取 $DEPLOY_BASE 下所有子目录 config/nodeConfig.yaml 的 environment，且要求全部一致"
    exit 1
}

extract_env_from_yaml() {
    local yaml_file="$1"

    awk -F ': *' '
        /^[[:space:]]*environment[[:space:]]*:/ {
            val=$2
            gsub(/^[[:space:]]+|[[:space:]]+$/, "", val)
            gsub(/"/, "", val)
            gsub(/\r/, "", val)
            print val
            exit
        }
    ' "$yaml_file"
}

detect_build_env() {
    local -a yaml_files=()
    local -a env_pairs=()
    local file=""
    local env=""
    local first_env=""

    while IFS= read -r -d '' file; do
        yaml_files+=("$file")
    done < <(find "$DEPLOY_BASE" -mindepth 2 -maxdepth 3 -type f -path "*/config/nodeConfig.yaml" -print0 2>/dev/null | sort -z)

    if [[ ${#yaml_files[@]} -eq 0 ]]; then
        echo "错误：未找到任何 nodeConfig.yaml，查找路径：$DEPLOY_BASE/*/config/nodeConfig.yaml" >&2
        exit 1
    fi

    for file in "${yaml_files[@]}"; do
        env="$(extract_env_from_yaml "$file")"
        if [[ -z "$env" ]]; then
            echo "错误：$file 中未找到 environment 配置" >&2
            exit 1
        fi
        env_pairs+=("$file => $env")
    done

    first_env="$(extract_env_from_yaml "${yaml_files[0]}")"

    for file in "${yaml_files[@]}"; do
        env="$(extract_env_from_yaml "$file")"
        if [[ "$env" != "$first_env" ]]; then
            echo "错误：检测到多个不一致的 environment 配置：" >&2
            for pair in "${env_pairs[@]}"; do
                echo "  $pair" >&2
            done
            exit 1
        fi
    done

    echo "$first_env"
}

BUILD_ENV="$(detect_build_env)"

echo "==> 自动检测到环境: $BUILD_ENV"

case "$BUILD_ENV" in
    dev)
        echo "==> 构建环境: dev（开发环境）"
        CGO_ENABLED=1
        TRIMPATH=""
        GCFLAGS='all=-N -l'
        RACE_FLAG="-race"
        ;;
    test)
        echo "==> 构建环境: test（测试环境）"
        CGO_ENABLED=1
        TRIMPATH="-trimpath"
        RACE_FLAG="-race"
        LDFLAGS=""
        ;;
    press)
        echo "==> 构建环境: press（压测环境）"
        CGO_ENABLED=0
        TRIMPATH="-trimpath"
        LDFLAGS="-s -w"
        ;;
    audit)
        echo "==> 构建环境: audit（审核环境）"
        CGO_ENABLED=0
        TRIMPATH="-trimpath"
        LDFLAGS="-s -w"
        ;;
    stage)
        echo "==> 构建环境: stage（预发布环境）"
        CGO_ENABLED=0
        TRIMPATH="-trimpath"
        LDFLAGS="-s -w"
        ;;
    prod)
        echo "==> 构建环境: prod（线上环境）"
        CGO_ENABLED=0
        TRIMPATH="-trimpath"
        LDFLAGS="-s -w"
        ;;
    *)
        echo "错误：不支持的环境 $BUILD_ENV" >&2
        usage
        ;;
esac

build() {
    local NAME="$1"
    local MAIN_FILE="$2"
    local OUT_DIR="$DEPLOY_BASE/$NAME"

    mkdir -p "$OUT_DIR"

    echo "==> Building $NAME -> $OUT_DIR/$NAME"
    echo "    MAIN_FILE   : $MAIN_FILE"
    echo "    BUILD_ENV   : $BUILD_ENV"
    echo "    BUILD_TIME  : $BUILD_TIME"
    echo "    CGO_ENABLED : $CGO_ENABLED"
    echo "    GOOS/GOARCH : $GOOS/$GOARCH"

    local CMD=(go build)

    if [[ -n "$TRIMPATH" ]]; then
        CMD+=("$TRIMPATH")
    fi

    if [[ -n "$GCFLAGS" ]]; then
        CMD+=(-gcflags "$GCFLAGS")
    fi

    if [[ -n "$ASMFLAGS" ]]; then
        CMD+=(-asmflags "$ASMFLAGS")
    fi

    if [[ -n "$RACE_FLAG" ]]; then
        CMD+=("$RACE_FLAG")
    fi

    if [[ -n "$BUILD_TAGS" ]]; then
        CMD+=(-tags "$BUILD_TAGS")
    fi

    if [[ -n "$(echo "$LDFLAGS" | xargs)" ]]; then
        CMD+=(-ldflags "$(echo "$LDFLAGS" | xargs)")
    fi

    CMD+=(-o "$OUT_DIR/$NAME" "$PROJECT_SERVER_DIR/$MAIN_FILE")

    CGO_ENABLED="$CGO_ENABLED" GOOS="$GOOS" GOARCH="$GOARCH" "${CMD[@]}"
}

build gameServer001 gameMain.go
build gatewayServer gatewayMain.go
build httpServer httpMain.go
build rankBoardServer rankBoardMain.go
build socialServer socialMain.go
build backendServer backendMain.go

echo "==> Build finished"