# 001 Low-Memory Operations Console Design

状态：已确认，Phase A 本地实现完成
版本：001
基线 commit：`1744489482045b17aa2361c5541aad95456cfefe`

## 1. 目标

将当前静态原型工程化为部署在目标服务器上的真实运维控制台，同时保持较低资源占用。

版本 001 聚焦只读能力：

- 展示服务器当前 CPU、内存、磁盘、网络和系统负载。
- 按开始、结束时间查询历史指标，精确到分钟。
- 对每个时间桶同时返回平均值和峰值，用峰值判断风险。
- 展示应用、systemd 服务、健康检查、快捷 URL 和只读发布记录。
- 保留现有中文界面和桌面、移动端交互。

## 2. 非目标

版本 001 不执行以下生产变更：

- 不执行应用部署、重启、回滚或任意 shell 命令。
- 不访问 Docker socket，也不要求运行用户加入 `docker` group。
- 不修改 Caddy、cloudflared、systemd 或现有应用配置。
- 不实现多服务器管理、复杂告警编排或集中日志检索。

界面中的部署和回滚按钮继续保持禁用或演示状态，并明确标识没有执行生产操作。

## 3. 已确认环境

通过 `sub2api-cf` SSH alias 完成只读盘点：

- Ubuntu 20.04.6 LTS，Linux 5.4，x86_64。
- 2 vCPU，约 3.6 GB 内存，2 GB swap。
- 根磁盘约 59 GB，当前可用约 34 GB。
- systemd、Caddy、cloudflared、Docker 和 containerd 正在运行。
- 当前 SSH 用户无 Docker socket 访问权限。
- `sub2api` SSH alias 在盘点时连接超时。

这些事实说明常驻进程应尽量少、不能依赖 Docker 管理权限，并且需要限制查询内存和历史数据增长。

## 4. 总体架构

采用一个 Go 二进制提供静态页面、API、指标采集和只读状态检查：

```text
Browser
  |
  v
Caddy / cloudflared access layer
  |
  v
origin-ops (127.0.0.1:9080)
  |-- embedded index.html / styles.css / app.js
  |-- read-only HTTP API
  |-- /proc and filesystem collector
  |-- systemd status adapter
  |-- HTTP health-check adapter
  `-- compact metric history store
```

选择 Go 标准库的原因：

- 可编译为单个静态资源内嵌二进制。
- 不需要 Node.js、Python runtime 或独立数据库进程。
- 并发 HTTP 和周期采集实现简单，资源上限容易测量。
- Linux 上可以直接读取 `/proc` 和文件系统统计。

版本 001 不引入第三方 Go dependency。若后续需要 SQLite、Docker API 或 systemd D-Bus，再根据收益和资源影响单独评审。

## 5. 进程与目录

建议的生产布局：

```text
/usr/local/bin/origin-ops
/etc/origin-ops/config.json
/var/lib/origin-ops/metrics/
/var/lib/origin-ops/releases/
```

- 服务仅监听 `127.0.0.1:9080`，不直接暴露公网端口。
- 使用独立低权限用户 `origin-ops` 运行。
- 配置文件只保存非敏感应用元数据，不保存 token、private key 或登录凭据。
- 数据目录只允许 `origin-ops` 写入。
- 对外访问由现有 Caddy/cloudflared 层负责；正式暴露前必须确认认证方案。

## 6. 指标采集

### 6.1 采样

默认每 60 秒采样一次：

- CPU：读取两次 `/proc/stat` 的累计时间差，计算总使用率。
- 内存：读取 `/proc/meminfo`，记录 total、available 和 used。
- 磁盘：读取根文件系统容量、可用量和使用量。
- 网络：读取 `/proc/net/dev`，聚合非 loopback 网卡的 RX/TX 字节差。
- Load：读取 `/proc/loadavg` 的 1、5、15 分钟值。

采集失败时保留上一份成功快照，同时记录失败时间和原因；不能用 `0` 冒充真实采样值。

### 6.2 历史存储

使用按 UTC 日期分段的固定长度二进制文件，不加载完整历史到内存：

```text
/var/lib/origin-ops/metrics/2026-08-15.bin
```

每条记录包含 timestamp、CPU、内存、磁盘、RX/TX 速率和 load。目标记录大小不超过 80 bytes：

- 每分钟采样一年约 525,600 条。
- 按 80 bytes 估算，原始指标约 40 MB/年。
- 默认保留 365 天，配置可缩短但不能静默删除未过期数据。

启动时只打开当天分段；查询时按日期流式读取需要的分段。发现尾部不完整记录时忽略该记录并写日志，不覆盖已有有效记录。

### 6.3 区间聚合

`start` 和 `end` 使用 RFC3339，服务端统一校验：

- `start < end`。
- 精度到分钟。
- 默认最大查询范围 31 天；更大范围后续通过降采样档案支持。
- 返回点数默认不超过 240，避免浏览器和后端内存随区间无限增长。

服务端根据区间自动计算时间桶，每个桶返回：

- `avg`：桶内有效采样的平均值。
- `max`：桶内有效采样的峰值。
- `samples`：有效采样数量。
- `missing`：该桶是否存在缺失采样。

前端趋势图展示平均值曲线和峰值曲线；悬浮信息展示时间桶、平均值、峰值和采样数量。

## 7. 应用与服务模型

应用通过 `/etc/origin-ops/config.json` 显式登记，不自动扫描所有系统进程：

```json
{
  "applications": [
    {
      "id": "fetchgithub",
      "name": "FetchGitHub",
      "publicUrl": "https://example.com",
      "services": ["fetchgithub-web.service", "fetchgithub-worker.service"],
      "healthUrl": "http://127.0.0.1:8080/health",
      "releaseRecord": "/var/lib/origin-ops/releases/fetchgithub.jsonl"
    }
  ]
}
```

约束：

- `publicUrl` 只允许 `http` 或 `https`。
- `healthUrl` 默认只允许 loopback 或明确允许的目标。
- systemd unit 名必须来自配置，不接受 API 请求传入任意 unit。
- 版本 001 只执行固定参数的 `systemctl show` 查询，不执行 start、stop、restart。
- Docker 容器状态不作为 001 的发布门禁，直到运行用户权限方案明确。

## 8. 发布记录

版本 001 支持读取规范化 JSON Lines 发布记录：

```json
{
  "version": "v2.4.1",
  "commit": "a8c32f1",
  "status": "success",
  "startedAt": "2026-08-15T09:40:00+08:00",
  "finishedAt": "2026-08-15T09:42:00+08:00",
  "actor": "manual",
  "rollbackTarget": "v2.4.0"
}
```

- 控制台只读取记录，不修改项目仓库。
- 文件写入协议和真正的部署执行器放入后续版本设计。
- 不把日志正文、环境变量或认证信息写进发布记录。

## 9. HTTP API

所有响应使用 JSON，时间使用 RFC3339：

| Method | Path                                 | Purpose                    |
| ------ | ------------------------------------ | -------------------------- |
| `GET`  | `/api/v1/health`                     | 进程健康状态               |
| `GET`  | `/api/v1/overview`                   | 主机信息和最新指标快照     |
| `GET`  | `/api/v1/metrics`                    | 区间平均值、峰值和采样明细 |
| `GET`  | `/api/v1/applications`               | 应用、服务和健康状态       |
| `GET`  | `/api/v1/applications/{id}/releases` | 只读发布记录               |

版本 001 不提供部署或回滚 `POST` API。前端调用不存在的变更接口必须失败关闭，不能回退为本地成功提示。

## 10. 安全边界

- 进程不以 root 运行。
- 服务只监听 loopback。
- 不读取 Docker socket、SSH private key、项目 secrets 或完整环境变量。
- HTTP server 设置 header、read、write 和 idle timeout。
- API 限制 query 长度、时间范围、返回点数和并发查询数。
- 健康检查禁止跟随到非允许目标，避免 SSRF。
- 日志不记录 cookies、authorization header 或完整带 query 的敏感 URL。
- 在确定 Cloudflare Access、Caddy authentication 或仅内网访问之前，不允许公开部署。

未来的部署与回滚能力应由独立受限执行器承担。监控进程只能提交结构化任务，不能获得通用 shell 权限。

## 11. 资源预算

在目标 Ubuntu 服务器上验收：

| Resource           | Target                                   |
| ------------------ | ---------------------------------------- |
| Idle RSS           | 不高于 35 MB                             |
| Query RSS peak     | 不高于 60 MB                             |
| Idle CPU           | 5 分钟平均低于单核 1%                    |
| Sampling interval  | 默认 60 秒                               |
| Chart points       | 每次查询不超过 240 个时间桶              |
| Metric disk growth | 不高于约 45 MB/年                        |
| Log growth         | 默认交给 journald 限额，不写无界日志文件 |

若 Go runtime 在目标环境无法达到预算，优先减少缓存、并发和响应大小，再评估实现语言；不通过关闭历史持久化来伪造低占用。

## 12. 实现阶段

### Phase A: Backend foundation

- [x] Go module、静态资源内嵌、配置加载和 HTTP server。
- [x] Linux 指标采集器和非 Linux 明确失败实现。
- [x] 固定长度历史存储、区间聚合和单元测试。
- [ ] 目标 Linux 主机上的采样值与资源预算验收。

### Phase B: Read-only inventory

- [x] systemd 状态适配器。
- [x] HTTP 健康检查适配器。
- [x] 应用和只读发布记录 API。
- [ ] 目标服务器实际应用配置与只读查询验收。

### Phase C: Frontend integration

- 将演示指标替换为 API 数据。
- 平均值、峰值双曲线和悬浮信息。
- 加载、空数据、缺失采样、后端离线和权限不足状态。
- 发布与回滚按钮保持禁用并说明当前不可执行。

### Phase D: Deployment preparation

- 生成 systemd unit 和示例配置。
- 本地与目标 Linux 构建验证。
- 资源测量、安全检查和部署前清单。

生产安装和 Caddy/cloudflared 变更必须再次获得用户明确授权。

## 13. 验收标准

- `go test ./...`、`go vet ./...`、`node --check app.js` 通过。
- Linux 采样数据与 `/proc`、`free`、`df` 的可观察值处于合理误差范围。
- 任意合法分钟区间返回平均值和峰值，超过限制的请求得到明确错误。
- 采样文件重启后可继续追加，尾部损坏不会破坏之前数据。
- 无历史数据和部分缺失数据时，界面不显示伪造值。
- 应用快捷 URL、systemd 状态、健康检查和发布记录来自配置/API。
- 桌面和移动端无横向溢出、文字重叠或图表轴越界，浏览器控制台无错误。
- 目标服务器实测满足资源预算，或记录偏差和调整结论。

## 14. 待确认项

以下内容不阻塞本地 Phase A，但在生产部署前必须明确：

- 控制台访问方式：Cloudflare Access、Caddy authentication 或仅内网访问。
- 实际纳管应用、systemd unit、健康检查 URL 和公开 URL 清单。
- `sub2api` SSH 超时原因及是否与 `sub2api-cf` 指向同一台服务器。
- Docker 服务是否需要纳管，以及是否接受只读 socket proxy，而不是直接开放 Docker group 权限。
- 各项目现有发布记录来源和未来部署执行器的权限模型。
