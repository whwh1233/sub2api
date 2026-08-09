#!/bin/bash
# remote-pull-restart.sh
# VPS 端：备份当前二进制 → git pull → 校验并解压 → 原子替换 → 重启
# 用法（VPS 上作为 root）：
#   bash /root/sub2api/deploy/remote-pull-restart.sh
#
# 如果新版本有问题，手动回滚：
#   bash /root/sub2api/deploy/remote-pull-restart.sh rollback

set -euo pipefail

REPO_DIR="${REPO_DIR:-/root/sub2api}"
BINARY="$REPO_DIR/backend/sub2api-linux"
BACKUP="$BINARY.prev"
BACKUP_STAGED="$BACKUP.new"
ARTIFACT="$REPO_DIR/backend/sub2api-linux.gz"
CHECKSUM="$REPO_DIR/backend/sub2api-linux.sha256"
STAGED="$REPO_DIR/backend/sub2api-linux.new"
SERVICE="${SERVICE:-sub2api}"
REPLACED=0

cd "$REPO_DIR"

# ---- rollback 模式 ----
if [ "${1:-}" = "rollback" ]; then
    if [ ! -f "$BACKUP" ]; then
        echo "ERROR: 没有备份文件 $BACKUP，无法回滚" >&2
        exit 1
    fi
    echo "==== 回滚到上一版本 ===="
    cp -a "$BACKUP" "$STAGED"
    chmod +x "$STAGED"
    mv -f "$STAGED" "$BINARY"
    systemctl restart "$SERVICE"
    sleep 2
    systemctl status "$SERVICE" --no-pager | sed -n '1,10p'
    echo ""
    echo "✓ 已回滚。当前跑的是备份版本，如需重新部署，再跑一次本脚本（不带 rollback 参数）"
    exit 0
fi

# ---- 部署模式 ----

# 失败时：替换前恢复路径但不重启；替换后从 .prev 原子回滚并重启。
rollback_on_fail() {
    local rc=$?
    rm -f "$BACKUP_STAGED"
    if [ $rc -ne 0 ]; then
        echo ""
        echo "!! 部署失败，处理本地二进制状态..." >&2
        if [ -f "$BACKUP" ]; then
            cp -a "$BACKUP" "$STAGED"
            chmod +x "$STAGED"
            mv -f "$STAGED" "$BINARY"
            if [ "$REPLACED" -eq 1 ]; then
                systemctl restart "$SERVICE" 2>/dev/null || true
                echo "   已恢复并重启备份版本" >&2
            else
                echo "   已恢复二进制路径；原进程未中断" >&2
            fi
        else
            echo "   没有备份可用，服务状态未知" >&2
        fi
    fi
}
trap rollback_on_fail EXIT

echo "==== [1/5] 备份当前二进制 ===="
if [ -f "$BINARY" ]; then
    # 先写入暂存路径并校验，再原子替换 .prev，避免留下部分备份。
    rm -f "$BACKUP_STAGED"
    cp -a "$BINARY" "$BACKUP_STAGED"
    live_hash="$(sha256sum "$BINARY" | awk '{print $1}')"
    backup_hash="$(sha256sum "$BACKUP_STAGED" | awk '{print $1}')"
    [ "$backup_hash" = "$live_hash" ] || {
        echo "ERROR: 当前二进制备份 SHA256 不匹配" >&2
        exit 1
    }
    mv -f "$BACKUP_STAGED" "$BACKUP"
    echo "  备份到: $BACKUP ($(ls -lh "$BACKUP" | awk '{print $5}'))"
    echo "  SHA256: $backup_hash"
else
    echo "  未发现当前二进制，跳过备份"
fi

echo "==== [2/5] git pull ===="
git pull --ff-only origin main

echo "==== [3/5] 校验并解压新二进制 ===="
[ -s "$ARTIFACT" ] || { echo "ERROR: pull 后未找到 $ARTIFACT" >&2; exit 1; }
[ -s "$CHECKSUM" ] || { echo "ERROR: pull 后未找到 $CHECKSUM" >&2; exit 1; }
gzip -t "$ARTIFACT"
rm -f "$STAGED"
gzip -dc "$ARTIFACT" > "$STAGED"
expected_hash="$(awk 'NR == 1 {print $1}' "$CHECKSUM")"
actual_hash="$(sha256sum "$STAGED" | awk '{print $1}')"
[ -n "$expected_hash" ] || { echo "ERROR: checksum 文件为空" >&2; exit 1; }
[ "$actual_hash" = "$expected_hash" ] || {
    echo "ERROR: 解压后二进制 SHA256 不匹配" >&2
    exit 1
}
chmod +x "$STAGED"
mv -f "$STAGED" "$BINARY"
REPLACED=1
echo "  大小: $(ls -lh "$BINARY" | awk '{print $5}')"
echo "  SHA256: $actual_hash"

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
systemctl status "$SERVICE" --no-pager | sed -n '1,12p'
echo "=========================="
echo ""
echo "✓ 部署完成"
echo ""
echo "备份文件: $BACKUP (下次部署会覆盖)"
echo "手动回滚: bash $0 rollback"
echo "查看日志: journalctl -u $SERVICE -n 30 -f"
