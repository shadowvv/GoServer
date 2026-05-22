#!/usr/bin/env bash

# 👉 当前脚本所在目录
APP_DIR="$(cd "$(dirname "$0")" && pwd)"

# 👉 自动取目录名作为服务名（关键）
APP_NAME="$(basename "$APP_DIR")"

BIN="${APP_DIR}/${APP_NAME}"
PID_FILE="${APP_DIR}/${APP_NAME}.pid"

start() {
    if [ -f "${PID_FILE}" ] && kill -0 $(cat "${PID_FILE}") 2>/dev/null; then
        echo "[${APP_NAME}] already running, pid=$(cat ${PID_FILE})"
        return
    fi

    echo "[${APP_NAME}] starting..."

    cd "${APP_DIR}"

    chmod +x "${BIN}" 2>/dev/null

    nohup "${BIN}" > "${APP_DIR}/nohup.out" 2>&1 &

    echo $! > "${PID_FILE}"

    sleep 1

    if kill -0 $(cat "${PID_FILE}") 2>/dev/null; then
        echo "[${APP_NAME}] started, pid=$(cat ${PID_FILE})"
    else
        echo "[${APP_NAME}] start failed"
        rm -f "${PID_FILE}"
    fi
}

stop() {
    if [ ! -f "${PID_FILE}" ]; then
        echo "[${APP_NAME}] not running"
        return
    fi

    PID=$(cat "${PID_FILE}")

    echo "[${APP_NAME}] stopping pid=${PID}"

    kill "${PID}" 2>/dev/null || true

    sleep 2

    if kill -0 "${PID}" 2>/dev/null; then
        echo "[${APP_NAME}] force killing..."
        kill -9 "${PID}" 2>/dev/null || true
    fi

    rm -f "${PID_FILE}"
    echo "[${APP_NAME}] stopped"
}

status() {
    if [ -f "${PID_FILE}" ] && kill -0 $(cat "${PID_FILE}") 2>/dev/null; then
        echo "[${APP_NAME}] running pid=$(cat ${PID_FILE})"
    else
        echo "[${APP_NAME}] not running"
    fi
}

restart() {
    stop
    start
}

case "$1" in
    start)
        start
        ;;
    stop)
        stop
        ;;
    restart)
        restart
        ;;
    status)
        status
        ;;
    *)
        echo "Usage: $0 {start|stop|restart|status}"
        exit 1
        ;;
esac
