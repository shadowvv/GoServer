#!/bin/bash

# 数据库配置
MYSQL_USER="root"
MYSQL_PASSWORD="123456"

# 备份目录
BACKUP_DIR="/data/backup/mysql"

# 时间戳
DATE=$(date +"%Y%m%d_%H%M%S")

# 创建备份目录
mkdir -p "${BACKUP_DIR}"

# 需要备份的数据库
DATABASES=(
    "game"
    "server"
    "rankBoard"
)

echo "========== Backup Start: $(date) =========="

for DB in "${DATABASES[@]}"
do
    FILE="${BACKUP_DIR}/${DB}_${DATE}.sql"

    echo "Backing up ${DB}..."

    mysqldump \
        -u${MYSQL_USER} \
        -p${MYSQL_PASSWORD} \
        --single-transaction \
        --routines \
        --triggers \
        --events \
        ${DB} > "${FILE}"

    if [ $? -eq 0 ]; then
        echo "[SUCCESS] ${DB} -> ${FILE}"
    else
        echo "[FAILED] ${DB}"
    fi
done

echo "========== Backup End: $(date) =========="