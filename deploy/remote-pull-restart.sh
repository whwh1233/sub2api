#!/bin/bash
# remote-pull-restart.sh
# VPS 端：git pull 最新代码（含新二进制），然后重启 systemd 服务
# 用法（在 VPS 上作为 root）：
#   bash /root/sub2api/deploy/remote-pull-restart.sh

set -euo pipefail

REPO_DIR="/root/sub2api"
BINARY="$REPO_DIR/backend/sub2api-linux"
SERVICE="sub2api"

# 失败时尝试把服务恢复成可运行状态
rollback_on_fail() {
    local rc=$?
    if [ $rc -ne 0 ]; then
        echo ""
        echo "!! 部署失败，尝试恢复服务..." >&2
        systemctl start "$SERVICE" 2>/dev/null || true
    fi
}
trap rollback_on_fail EXIT

cd "$REPO_DIR"

echo "==== [1/4] 停止服务 ===="
systemctl stop "$SERVICE"

echo "==== [2/4] git pull ===="
git pull origin main

echo "==== [3/4] 校验二进制 ===="
[ -f "$BINARY" ] || { echo "ERROR: 二进制缺失: $BINARY"; exit 1; }
chmod +x "$BINARY"
echo "  路径: $BINARY"
echo "  大小: $(ls -lh "$BINARY" | awk '{print $5}')"

echo "==== [4/4] 启动服务 ===="
systemctl start "$SERVICE"
sleep 2

if systemctl is-active --quiet "$SERVICE"; then
    echo "  ✓ 服务运行中"
    trap - EXIT
else
    echo "  !! 服务启动失败"
    exit 1
fi

echo ""
echo "=========================="
systemctl status "$SERVICE" --no-pager | head -12
echo "=========================="
echo ""
echo "✓ 部署完成。查看日志: journalctl -u $SERVICE -n 30 -f"
