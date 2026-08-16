# Origin Ops 部署准备

版本 001 只提供主机指标、应用状态和发布记录读取。本文提供安装模板和验收步骤，不代表已经在生产服务器执行部署。

## 文件布局

```text
/usr/local/bin/origin-ops
/etc/origin-ops/config.json
/var/lib/origin-ops/auth/users.json
/var/lib/origin-ops/metrics/
/var/lib/origin-ops/releases/
/etc/systemd/system/origin-ops.service
```

运行用户应为独立的 `origin-ops`，数据目录由该用户写入；配置文件只保存应用元数据，不保存 token、cookie、私钥或其他凭据。

## 安装步骤

在目标 Linux 主机上执行前，先确认变更窗口、备份和访问认证方案。以下命令假定从仓库根目录操作：

```bash
go test ./...
go vet ./...
GOOS=linux GOARCH=amd64 go build -o origin-ops .

sudo install -d -o origin-ops -g origin-ops -m 0700 /var/lib/origin-ops/auth
sudo install -d -o origin-ops -g origin-ops -m 0750 \
  /var/lib/origin-ops/metrics /var/lib/origin-ops/releases
sudo install -d -m 0755 /etc/origin-ops
sudo install -o root -g root -m 0755 origin-ops /usr/local/bin/origin-ops
sudo install -o root -g origin-ops -m 0640 config.json /etc/origin-ops/config.json
sudo install -o root -g root -m 0644 deploy/origin-ops.service \
  /etc/systemd/system/origin-ops.service

sudo systemctl daemon-reload
sudo systemctl enable --now origin-ops.service
curl --fail --silent http://127.0.0.1:9080/api/v1/health
sudo systemctl --no-pager --full status origin-ops.service
```

`config.json` 必须使用 loopback `listenAddress`，应用的 `healthUrl` 只能指向允许的 loopback 地址；不要直接复制 `config.example.json` 中的路径到不存在的生产目录。

## 本地账号初始化

Origin Ops 不引入数据库。账号文件只保存 PBKDF2-SHA256 盐值和哈希，不保存明文密码。首次初始化必须由管理员在服务器交互输入密码，不要把密码写进命令历史、配置文件或提交记录：

```bash
read -r -s ORIGIN_OPS_PASSWORD
printf '%s\n' "$ORIGIN_OPS_PASSWORD" | \
  sudo -u origin-ops /usr/local/bin/origin-ops user add \
  --config /etc/origin-ops/config.json --username <username> --password-stdin
unset ORIGIN_OPS_PASSWORD
```

后续账号操作：

```bash
sudo -u origin-ops /usr/local/bin/origin-ops user list --config /etc/origin-ops/config.json
printf '%s\n' "$ORIGIN_OPS_PASSWORD" | sudo -u origin-ops \
  /usr/local/bin/origin-ops user set-password \
  --config /etc/origin-ops/config.json --username <username> --password-stdin
sudo -u origin-ops /usr/local/bin/origin-ops user disable \
  --config /etc/origin-ops/config.json --username <username>
```

改密、启用或禁用用户会立即使旧会话失效；服务重启也会清除内存会话。

## 发布记录接入

部署脚本只应通过已登记应用 ID 追加 JSONL 记录，不能传入任意文件路径：

```bash
/usr/local/bin/origin-ops release append \
  --config /etc/origin-ops/config.json --application <id> \
  --version <version> --commit <commit> --status success \
  --started-at <RFC3339> --finished-at <RFC3339> --actor <actor> \
  --rollback-target <version>
```

## 反向代理与认证

Origin Ops 只监听 `127.0.0.1:9080`，不应直接暴露公网。当前选择 Cloudflare Access 作为外层认证，控制台自身再使用本地账号登录；Cloudflare Access 应在控制台域名对外开放前由管理员在 Cloudflare 控制台创建并验证策略。版本 001 不自动修改现有代理配置。

## 安全检查

- `systemctl cat origin-ops.service` 中确认 `User=origin-ops`、`ProtectSystem=strict` 和 `NoNewPrivileges=true`。
- `ss -ltnp` 确认只监听 `127.0.0.1:9080`。
- 检查配置、发布记录和指标目录权限，不允许其他用户写入。
- 检查 `/var/lib/origin-ops/auth/users.json` 为 `0600`，认证目录为 `0700`。
- 确认服务账户不属于 `docker` 组，不读取 Docker socket、SSH 私钥或完整环境变量。
- 使用异常、超长和重定向健康检查 URL 验证 API 返回失败关闭，不发生外连跳转。
- 查看 `journalctl -u origin-ops.service`，确认日志不包含 authorization、cookie 或敏感 query。

## 资源验收

在目标主机产生首个采样后记录空闲资源；查询 31 天范围时再次记录峰值。结果应对照设计目标：Idle RSS 不高于 35 MB、查询 RSS 峰值不高于 60 MB、空闲 CPU 5 分钟平均低于单核 1%、历史增长约不高于 45 MB/年。

```bash
pid=$(systemctl show -p MainPID --value origin-ops.service)
ps -o pid,rss,pcpu,cmd -p "$pid"
/usr/bin/time -v curl --fail --silent \
  'http://127.0.0.1:9080/api/v1/metrics?start=2026-01-01T00:00:00Z&end=2026-01-31T00:00:00Z' \
  >/dev/null
du -sh /var/lib/origin-ops/metrics
```

若目标主机无法满足预算，应先降低并发、查询范围或缓存，再重新验收；不能通过关闭历史存储来掩盖偏差。

## 回滚

版本 001 尚未建立生产 release tag。发生异常时，停止服务并恢复上一份已验证二进制和配置，再执行 `daemon-reload`、重启和 `/api/v1/health` 检查。回滚前保留 `journalctl` 输出和指标目录，不删除历史数据。
