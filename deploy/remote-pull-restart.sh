#!/bin/bash
# remote-pull-restart.sh
# VPS 端：备份当前二进制 → git pull → 重启 → 失败自动回滚
# 用法（VPS 上作为 root）：
#   bash /root/sub2api/deploy/remote-pull-restart.sh
#
# 如果新版本有问题，手动回滚：
#   bash /root/sub2api/deploy/remote-pull-restart.sh rollback

set -euo pipefail

REPO_DIR="/root/sub2api"
BINARY="$REPO_DIR/backend/sub2api-linux"
BACKUP="$BINARY.prev"
SERVICE="sub2api"

cd "$REPO_DIR"

# ---- rollback 模式 ----
if [ "${1:-}" = "rollback" ]; then
    if [ ! -f "$BACKUP" ]; then
        echo "ERROR: 没有备份文件 $BACKUP，无法回滚" >&2
        exit 1
    fi
    echo "==== 回滚到上一版本 ===="
    systemctl stop "$SERVICE"
    rm -f "$BINARY"
    mv "$BACKUP" "$BINARY"
    chmod +x "$BINARY"
    systemctl start "$SERVICE"
    sleep 2
    systemctl status "$SERVICE" --no-pager | head -10
    echo ""
    echo "✓ 已回滚。当前跑的是备份版本，如需重新部署，再跑一次本脚本（不带 rollback 参数）"
    exit 0
fi

# ---- 部署模式 ----

# 失败时自动从 .prev 回滚
rollback_on_fail() {
    local rc=$?
    if [ $rc -ne 0 ]; then
        echo ""
        echo "!! 部署失败，尝试从备份回滚..." >&2
        if [ -f "$BACKUP" ]; then
            rm -f "$BINARY"
            mv "$BACKUP" "$BINARY"
            chmod +x "$BINARY"
            systemctl start "$SERVICE" 2>/dev/null || true
            echo "   已恢复为备份版本" >&2
        else
            echo "   没有备份可用，服务状态未知" >&2
        fi
    fi
}
trap rollback_on_fail EXIT

echo "==== [1/5] 备份当前二进制 ===="
if [ -f "$BINARY" ]; then
    # 如果已有 .prev，先删掉（每次只保留最近一版）
    [ -f "$BACKUP" ] && rm -f "$BACKUP"
    mv "$BINARY" "$BACKUP"
    echo "  备份到: $BACKUP ($(ls -lh "$BACKUP" | awk '{print $5}'))"
else
    echo "  未发现当前二进制，跳过备份"
fi

echo "==== [2/5] git pull ===="
git pull origin main

echo "==== [3/5] 校验新二进制 ===="
[ -f "$BINARY" ] || { echo "ERROR: pull 后未找到 $BINARY" >&2; exit 1; }
chmod +x "$BINARY"
echo "  大小: $(ls -lh "$BINARY" | awk '{print $5}')"

echo "==== [4/5] 重启服务 ===="
systemctl restart "$SERVICE"
sleep 2

echo "==== [5/5] 健康检查 ===="
if systemctl is-active --quiet "$SERVICE"; then
    echo "  ✓ 服务运行中"
    trap - EXIT
else
    echo "  !! 服务未启动" >&2
    exit 1
fi

echo ""
echo "=========================="
systemctl status "$SERVICE" --no-pager | head -12
echo "=========================="
echo ""
echo "✓ 部署完成"
echo ""
echo "备份文件: $BACKUP (下次部署会覆盖)"
echo "手动回滚: bash $0 rollback"
echo "查看日志: journalctl -u $SERVICE -n 30 -f"
