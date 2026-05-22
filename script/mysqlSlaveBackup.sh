#!/bin/bash
set -euo pipefail

LOGIN_PATH="backup3307"
BACKUP_DIR="/data/backup/mysql"
LOG_DIR="/data/logs/mysql_backup"
DATE="$(date +%F_%H%M%S)"
HOST_TAG="slave3307"
RETENTION_DAYS=7

BACKUP_FILE="${BACKUP_DIR}/mysql_${HOST_TAG}_${DATE}.sql.gz"
LOG_FILE="${LOG_DIR}/mysql_backup_${DATE}.log"
LATEST_LINK="${BACKUP_DIR}/latest.sql.gz"

mkdir -p "${BACKUP_DIR}"
mkdir -p "${LOG_DIR}"

log() {
    echo "[$(date '+%F %T')] $*" | tee -a "${LOG_FILE}"
}

cleanup_on_error() {
    log "备份失败，删除不完整文件: ${BACKUP_FILE}"
    rm -f "${BACKUP_FILE}"
}
trap cleanup_on_error ERR

log "开始执行 MySQL 从库备份"

if ! mysql --login-path="${LOGIN_PATH}" -e "SELECT 1;" >/dev/null 2>&1; then
    log "错误：无法连接到 MySQL"
    exit 1
fi
log "MySQL 连接检查通过"

READ_ONLY_VAL=$(mysql --login-path="${LOGIN_PATH}" -Nse "SHOW VARIABLES LIKE 'read_only';" | awk '{print $2}')
SUPER_READ_ONLY_VAL=$(mysql --login-path="${LOGIN_PATH}" -Nse "SHOW VARIABLES LIKE 'super_read_only';" | awk '{print $2}')

log "read_only=${READ_ONLY_VAL:-unknown}, super_read_only=${SUPER_READ_ONLY_VAL:-unknown}"

log "记录复制状态"
mysql --login-path="${LOGIN_PATH}" -e "SHOW REPLICA STATUS\G" >> "${LOG_FILE}" 2>&1 || true

mysqldump --login-path="${LOGIN_PATH}" \
  --all-databases \
  --single-transaction \
  --routines \
  --events \
  --triggers \
  --set-gtid-purged=OFF \
  --no-tablespaces \
  --hex-blob \
  --default-character-set=utf8mb4 \
  --skip-comments \
  | gzip > "${BACKUP_FILE}"

log "备份完成: ${BACKUP_FILE}"

if [[ ! -s "${BACKUP_FILE}" ]]; then
    log "错误：备份文件不存在或为空"
    exit 1
fi

gzip -t "${BACKUP_FILE}"
log "gzip 完整性校验通过"

ln -sfn "${BACKUP_FILE}" "${LATEST_LINK}"
log "已更新最新备份软链接: ${LATEST_LINK}"

find "${BACKUP_DIR}" -maxdepth 1 -type f -name "mysql_${HOST_TAG}_*.sql.gz" -mtime +${RETENTION_DAYS} -print -delete | tee -a "${LOG_FILE}" || true
log "已清理 ${RETENTION_DAYS} 天前的旧备份"

ls -lh "${BACKUP_FILE}" | tee -a "${LOG_FILE}"
log "MySQL 备份流程结束"
