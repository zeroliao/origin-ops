const pageLabels = {
  overview: "总览",
  applications: "应用",
  services: "服务",
  deployments: "发布记录",
  events: "通知",
};

const metricDefinitions = {
  cpu: { label: "CPU 使用率", color: "#1e68cf", unit: "%" },
  memory: { label: "内存使用率", color: "#117a70", unit: "%" },
  network: { label: "网络出站", color: "#6b5fc7", unit: " MB/s" },
};

let applications = [];
let selectedAppID = null;
const collapsedApplicationGroups = new Set();
let activeRange = null;
let selectedTrendMetric = "cpu";
let activeChartBounds = null;
let toastTimeout;
let currentUsername = "";

const $ = (selector) => document.querySelector(selector);
const $$ = (selector) => [...document.querySelectorAll(selector)];

function escapeHTML(value) {
  return String(value).replace(
    /[&<>'"]/g,
    (character) =>
      ({
        "&": "&amp;",
        "<": "&lt;",
        ">": "&gt;",
        "'": "&#39;",
        '"': "&quot;",
      })[character],
  );
}

function showToast(message, isError = false) {
  const toast = $("#toast");
  toast.textContent = message;
  toast.classList.toggle("is-error", isError);
  toast.classList.add("is-visible");
  window.clearTimeout(toastTimeout);
  toastTimeout = window.setTimeout(
    () => toast.classList.remove("is-visible"),
    3600,
  );
}

async function requestJSON(path, options = {}) {
  const response = await fetch(path, {
    ...options,
    credentials: "same-origin",
    headers: { Accept: "application/json", ...(options.headers || {}) },
  });
  if (response.status === 204) return null;
  let payload = null;
  try {
    payload = await response.json();
  } catch {
    throw new Error("后端返回了无效响应");
  }
  if (!response.ok) {
    const error = new Error(payload.error || "请求失败");
    error.status = response.status;
    if (response.status === 401 && path !== "/api/v1/session") {
      showLogin("登录已失效，请重新登录。");
    }
    throw error;
  }
  return payload;
}

function showLogin(message = "") {
  $("#app-shell").hidden = true;
  $("#auth-gate").hidden = false;
  $("#auth-status").hidden = true;
  $("#login-form").hidden = false;
  $("#login-error").textContent = message;
  $("#account-menu").hidden = true;
  $("#profile-button").setAttribute("aria-expanded", "false");
  $("#login-username").focus();
}

function showDashboard(username) {
  currentUsername = username || "operator";
  $("#auth-gate").hidden = true;
  $("#app-shell").hidden = false;
  $("#account-name").textContent = currentUsername;
  $("#profile-button").textContent = currentUsername.slice(0, 1).toUpperCase();
  $("#profile-button").ariaLabel = `${currentUsername} 账户菜单`;
}

function pad(value) {
  return String(value).padStart(2, "0");
}

function formatDateInput(date) {
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}T${pad(date.getHours())}:${pad(date.getMinutes())}`;
}

function formatDateTime(value) {
  const date = value instanceof Date ? value : new Date(value);
  if (Number.isNaN(date.getTime())) return "未知时间";
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())} ${pad(date.getHours())}:${pad(date.getMinutes())}`;
}

function formatAxisDate(value) {
  const date = value instanceof Date ? value : new Date(value);
  return `${pad(date.getMonth() + 1)}-${pad(date.getDate())} ${pad(date.getHours())}:${pad(date.getMinutes())}`;
}

function formatBytes(bytes) {
  if (!Number.isFinite(bytes)) return "--";
  return `${(bytes / 1024 ** 3).toFixed(2)} GB`;
}

function formatMetricValue(metricKey, value) {
  if (!Number.isFinite(value)) return "--";
  return `${value.toFixed(metricKey === "network" ? 2 : 1)}${metricDefinitions[metricKey].unit}`;
}

function formatMetricChange(metricKey, value) {
  if (!Number.isFinite(value)) return "--";
  return `${value >= 0 ? "+" : ""}${formatMetricValue(metricKey, value)}`;
}

function percentage(used, total) {
  return total > 0 ? (used / total) * 100 : 0;
}

function setDataBadge(label, isError = false) {
  const badge = $("#data-source-badge");
  badge.lastChild.textContent = label;
  badge.classList.toggle("is-error", isError);
}

function samplerMessage(lastError) {
  if (!lastError) return "等待首个采样";
  if (lastError.includes("supported only on Linux"))
    return "当前运行环境不支持 Linux 指标采集";
  return lastError.length > 72
    ? "采集失败，详见服务日志"
    : `采集失败：${lastError}`;
}

function setOverviewUnavailable(message, hostname = "后端离线") {
  $("#host-name").textContent = hostname;
  ["#cpu-value", "#memory-value", "#disk-value", "#network-value"].forEach(
    (selector) => {
      $(selector).textContent = "--";
    },
  );
  $("#cpu-change").textContent = message;
  $("#cpu-change").className = "change";
  $("#memory-total").textContent = "/ --";
  $("#disk-total").textContent = "/ --";
  $("#memory-progress").value = 0;
  $("#disk-progress").value = 0;
  $("#cpu-caption").textContent = message;
  $("#memory-caption").textContent = message;
  $("#disk-caption").textContent = message;
  $("#network-change").textContent = message;
  $("#network-change").className = "change";
  $("#network-caption").textContent = message;
  $("#server-time").textContent = "数据不可用";
  $("#server-time").removeAttribute("datetime");
  $("#current-metrics-time").textContent = "数据不可用";
  $("#current-metrics-time").removeAttribute("datetime");
  $("#overall-status-title").textContent = "主机指标不可用";
  $("#overall-status-note").textContent = message;
  $("#status-banner").className = "status-banner is-error";
}

function renderOverview(payload) {
  const host = payload.host || {};
  const sampler = payload.sampler || {};
  $("#host-name").textContent = host.hostname || "未知主机";
  if (!sampler.hasLatest) {
    const message = samplerMessage(sampler.lastError);
    setOverviewUnavailable(message, host.hostname || "未知主机");
    return;
  }

  const snapshot = sampler.latest;
  const memoryPercent = percentage(
    snapshot.memoryUsedBytes,
    snapshot.memoryTotalBytes,
  );
  const diskPercent = percentage(
    snapshot.diskUsedBytes,
    snapshot.diskTotalBytes,
  );
  const latestTime = new Date(snapshot.time);
  $("#server-time").textContent = formatDateTime(latestTime);
  $("#server-time").dateTime = snapshot.time;
  $("#current-metrics-time").textContent =
    `更新于 ${formatDateTime(latestTime)}`;
  $("#current-metrics-time").dateTime = snapshot.time;
  $("#cpu-value").textContent = formatMetricValue("cpu", snapshot.cpuPercent);
  $("#cpu-change").textContent = "实时采样";
  $("#cpu-change").className = "change is-good";
  $("#cpu-caption").textContent =
    `${host.cpuCount || "--"} vCPU · Load ${Number(snapshot.load1 || 0).toFixed(2)}`;
  $("#memory-value").textContent = formatBytes(snapshot.memoryUsedBytes);
  $("#memory-total").textContent =
    `/ ${formatBytes(snapshot.memoryTotalBytes)}`;
  $("#memory-progress").value = memoryPercent;
  $("#memory-progress").setAttribute(
    "aria-label",
    `内存使用率 ${memoryPercent.toFixed(1)}%`,
  );
  $("#memory-caption").textContent = `使用率 ${memoryPercent.toFixed(1)}%`;
  $("#disk-value").textContent = formatBytes(snapshot.diskUsedBytes);
  $("#disk-total").textContent = `/ ${formatBytes(snapshot.diskTotalBytes)}`;
  $("#disk-progress").value = diskPercent;
  $("#disk-progress").setAttribute(
    "aria-label",
    `磁盘使用率 ${diskPercent.toFixed(1)}%`,
  );
  $("#disk-caption").textContent = `使用率 ${diskPercent.toFixed(1)}%`;
  $("#network-value").textContent = formatMetricValue(
    "network",
    snapshot.txBytesPerSecond / 1024 ** 2,
  );
  $("#network-change").textContent =
    `入站 ${(snapshot.rxBytesPerSecond / 1024 ** 2).toFixed(2)} MB/s`;
  $("#network-change").className = "change is-good";
  $("#network-caption").textContent = "最近一次采样速率";
  $("#overall-status-title").textContent = "主机正在采样";
  $("#overall-status-note").textContent =
    `${host.os || "未知系统"}/${host.arch || "未知架构"} · 最近采样 ${formatDateTime(latestTime)}`;
  $("#status-banner").className = "status-banner is-healthy";
}

function metricValues(bucket, key) {
  if (key === "cpu") return bucket.avg.cpuPercent;
  if (key === "memory") return bucket.avg.memoryPercent;
  return bucket.avg.txBytesPerSecond / 1024 ** 2;
}

function metricPeakValues(bucket, key) {
  if (key === "cpu") return bucket.max.cpuPercent;
  if (key === "memory") return bucket.max.memoryPercent;
  return bucket.max.txBytesPerSecond / 1024 ** 2;
}

function buildRange(payload) {
  const buckets = (payload.buckets || []).filter(
    (bucket) => bucket.samples > 0,
  );
  const metrics = Object.fromEntries(
    Object.keys(metricDefinitions).map((key) => [
      key,
      {
        label: metricDefinitions[key].label,
        color: metricDefinitions[key].color,
        values: buckets.map((bucket) => metricValues(bucket, key)),
        peaks: buckets.map((bucket) => metricPeakValues(bucket, key)),
        times: buckets.map((bucket) => new Date(bucket.start)),
      },
    ]),
  );
  const missing = (payload.buckets || []).filter(
    (bucket) => bucket.missing,
  ).length;
  return {
    start: new Date(payload.start),
    end: new Date(payload.end),
    buckets,
    metrics,
    missing,
    samplingInterval: payload.samplingInterval,
    label: `${formatDateTime(payload.start)} 至 ${formatDateTime(payload.end)}`,
  };
}

function getChartBounds(values) {
  const minimum = Math.min(...values);
  const maximum = Math.max(...values);
  const padding = Math.max((maximum - minimum) * 0.12, 1);
  return {
    minimum: Math.max(0, minimum - padding),
    maximum: maximum + padding,
  };
}

function toChartPoints(values, bounds) {
  const left = 24;
  const right = 696;
  const top = 24;
  const bottom = 156;
  const span = Math.max(bounds.maximum - bounds.minimum, 1);
  return values.map((value, index) => {
    const x =
      values.length === 1
        ? (left + right) / 2
        : left + ((right - left) * index) / (values.length - 1);
    const y = top + ((bounds.maximum - value) / span) * (bottom - top);
    return [Number(x.toFixed(1)), Number(y.toFixed(1))];
  });
}

function hideTrendTooltip() {
  $("#chart-tooltip").hidden = true;
  $("#trend-hover-guide").classList.remove("is-visible");
  $("#trend-hover-point").classList.remove("is-visible");
}

function renderEmptyTrend(message) {
  const baseline = "24,156 696,156";
  $("#trend-line").setAttribute("points", baseline);
  $("#trend-peak-line").setAttribute("points", baseline);
  $("#trend-area").setAttribute("points", `${baseline} 696,156 24,156`);
  $("#trend-point").setAttribute("cx", "696");
  $("#trend-point").setAttribute("cy", "156");
  $("#trend-chart").classList.add("is-empty");
  $("#trend-range-name").textContent = "暂无历史指标";
  $("#trend-sample-note").textContent = message;
  $("#trend-current-value").textContent = "--";
  $("#trend-axis").innerHTML = "<span>--</span><span>--</span><span>--</span>";
  ["one", "two", "three", "four"].forEach((part) => {
    $(`#summary-${part}-value`).textContent = "--";
    $(`#summary-${part}-note`).textContent = message;
  });
  hideTrendTooltip();
}

function riskFor(metricKey, peak) {
  const limits = metricKey === "network" ? [5, 8] : [65, 85];
  if (peak >= limits[1])
    return {
      label: "高风险",
      className: "risk-high",
      note: `峰值达到 ${formatMetricValue(metricKey, peak)}`,
    };
  if (peak >= limits[0])
    return {
      label: "中风险",
      className: "risk-medium",
      note: `峰值达到 ${formatMetricValue(metricKey, peak)}`,
    };
  return {
    label: "低风险",
    className: "risk-low",
    note: `峰值低于 ${formatMetricValue(metricKey, limits[0])}`,
  };
}

function renderTrend() {
  if (!activeRange) {
    renderEmptyTrend("等待指标查询");
    return;
  }
  const metric = activeRange.metrics[selectedTrendMetric];
  if (!metric.values.length) {
    renderEmptyTrend("该时间范围没有有效采样");
    return;
  }
  $("#trend-chart").classList.remove("is-empty");
  activeChartBounds = getChartBounds([...metric.values, ...metric.peaks]);
  const averagePoints = toChartPoints(metric.values, activeChartBounds);
  const peakPoints = toChartPoints(metric.peaks, activeChartBounds);
  const pointText = averagePoints.map(([x, y]) => `${x},${y}`).join(" ");
  $("#trend-line").setAttribute("points", pointText);
  $("#trend-peak-line").setAttribute(
    "points",
    peakPoints.map(([x, y]) => `${x},${y}`).join(" "),
  );
  $("#trend-area").setAttribute("points", `${pointText} 696,156 24,156`);
  const lastPoint = averagePoints.at(-1);
  $("#trend-point").setAttribute("cx", lastPoint[0]);
  $("#trend-point").setAttribute("cy", lastPoint[1]);
  $(".trend-chart").style.color = metric.color;
  $("#trend-chart-desc").textContent =
    `${activeRange.label}内的${metric.label}平均值与峰值趋势`;
  $("#trend-chart").setAttribute(
    "aria-label",
    `${metric.label}趋势图。实线为平均值，虚线为峰值；使用方向键查看每个时间桶。`,
  );
  $("#trend-range-name").textContent = activeRange.label;
  $("#trend-sample-note").textContent =
    `${metric.values.length} 个有效时间桶 · 平均值实线 · 峰值虚线${activeRange.missing ? ` · ${activeRange.missing} 个缺失` : ""}`;
  $("#trend-metric-label").textContent = metric.label;
  $("#trend-current-value").textContent =
    `末值 ${formatMetricValue(selectedTrendMetric, metric.values.at(-1))}`;
  $("#trend-axis").innerHTML = [
    metric.times[0],
    metric.times[Math.floor(metric.times.length / 2)],
    metric.times.at(-1),
  ]
    .map((time) => `<span>${escapeHTML(formatAxisDate(time))}</span>`)
    .join("");
  $("#trend-y-axis-left").hidden = true;
  $("#trend-y-axis-right").hidden = true;
  $("#chart-stage").classList.remove("has-dual-axis");
  const average =
    metric.values.reduce((total, value) => total + value, 0) /
    metric.values.length;
  const change = metric.values.at(-1) - metric.values[0];
  const peak = Math.max(...metric.peaks);
  const peakIndex = metric.peaks.indexOf(peak);
  const risk = riskFor(selectedTrendMetric, peak);
  $("#summary-one-label").textContent = "采样平均值";
  $("#summary-one-value").textContent = formatMetricValue(
    selectedTrendMetric,
    average,
  );
  $("#summary-one-note").textContent = metric.label;
  $("#summary-two-label").textContent = "区间变化";
  $("#summary-two-value").textContent = formatMetricChange(
    selectedTrendMetric,
    change,
  );
  $("#summary-two-note").textContent = "首个至最后有效时间桶";
  $("#summary-three-label").textContent = "采样峰值";
  $("#summary-three-value").textContent = formatMetricValue(
    selectedTrendMetric,
    peak,
  );
  $("#summary-three-note").textContent =
    `${formatDateTime(metric.times[peakIndex])} 出现`;
  $("#summary-four-label").textContent = "峰值风险";
  const riskValue = $("#summary-four-value");
  riskValue.textContent = risk.label;
  riskValue.className = risk.className;
  $("#summary-four-note").textContent = risk.note;
  $$("[data-trend-metric]").forEach((button) =>
    button.classList.toggle(
      "is-selected",
      button.dataset.trendMetric === selectedTrendMetric,
    ),
  );
  hideTrendTooltip();
}

function showTrendTooltip(index) {
  if (!activeRange) return;
  const metric = activeRange.metrics[selectedTrendMetric];
  if (!metric.values.length) return;
  const safeIndex = Math.max(0, Math.min(index, metric.values.length - 1));
  const [x, y] = toChartPoints(metric.values, activeChartBounds)[safeIndex];
  $("#trend-hover-guide").setAttribute("x1", x);
  $("#trend-hover-guide").setAttribute("x2", x);
  $("#trend-hover-point").setAttribute("cx", x);
  $("#trend-hover-point").setAttribute("cy", y);
  $("#trend-hover-guide").classList.add("is-visible");
  $("#trend-hover-point").classList.add("is-visible");
  const tooltip = $("#chart-tooltip");
  tooltip.style.setProperty("--tooltip-x", `${(x / 720) * 100}%`);
  tooltip.style.setProperty("--tooltip-y", `${(y / 180) * 100}%`);
  tooltip.classList.toggle("is-below", y < 72);
  tooltip.innerHTML = `<strong>${escapeHTML(formatDateTime(metric.times[safeIndex]))}</strong><span>${escapeHTML(metric.label)} 平均 ${escapeHTML(formatMetricValue(selectedTrendMetric, metric.values[safeIndex]))} · 峰值 ${escapeHTML(formatMetricValue(selectedTrendMetric, metric.peaks[safeIndex]))}</span>`;
  tooltip.hidden = false;
}

function bindTrendChartInteractions() {
  const chart = $("#trend-chart");
  let keyboardIndex = 0;
  const pointerIndex = (event) => {
    const metric = activeRange?.metrics[selectedTrendMetric];
    if (!metric?.values.length) return 0;
    const bounds = chart.getBoundingClientRect();
    const ratio = Math.max(
      0,
      Math.min(1, (event.clientX - bounds.left) / bounds.width),
    );
    return Math.round(ratio * (metric.values.length - 1));
  };
  chart.addEventListener("pointermove", (event) =>
    showTrendTooltip(pointerIndex(event)),
  );
  chart.addEventListener("pointerdown", (event) =>
    showTrendTooltip(pointerIndex(event)),
  );
  chart.addEventListener("pointerleave", hideTrendTooltip);
  chart.addEventListener("focus", () => showTrendTooltip(keyboardIndex));
  chart.addEventListener("blur", hideTrendTooltip);
  chart.addEventListener("keydown", (event) => {
    const length =
      activeRange?.metrics[selectedTrendMetric]?.values.length || 0;
    if (!length) return;
    if (event.key === "Home") keyboardIndex = 0;
    else if (event.key === "End") keyboardIndex = length - 1;
    else if (event.key === "ArrowLeft")
      keyboardIndex = Math.max(0, keyboardIndex - 1);
    else if (event.key === "ArrowRight")
      keyboardIndex = Math.min(length - 1, keyboardIndex + 1);
    else return;
    event.preventDefault();
    showTrendTooltip(keyboardIndex);
  });
}

function renderRangeData() {
  if (!activeRange) return;
  $("#range-data-title-period").textContent = activeRange.label;
  $("#range-data-description").textContent =
    "展示后端返回的有效时间桶平均值；趋势图虚线表示同桶峰值。";
  $("#range-data-count").textContent =
    `${activeRange.buckets.length} 个有效时间桶`;
  $("#range-data-retention").textContent = activeRange.missing
    ? `${activeRange.missing} 个时间桶存在缺失`
    : "没有检测到缺失时间桶";
  $("#range-data-list").innerHTML = activeRange.buckets
    .map(
      (bucket) =>
        `<tr><td>${escapeHTML(formatDateTime(bucket.start))}</td><td>${escapeHTML(formatMetricValue("cpu", metricValues(bucket, "cpu")))}</td><td>${escapeHTML(formatMetricValue("memory", metricValues(bucket, "memory")))}</td><td>${escapeHTML(formatMetricValue("memory", bucket.avg.diskPercent))}</td><td>${escapeHTML(formatMetricValue("network", metricValues(bucket, "network")))}</td></tr>`,
    )
    .join("");
}

function openRangeData() {
  renderRangeData();
  $("#range-data-dialog").showModal();
}

function applicationHealth(application) {
  if (
    application.health.status === "healthy" &&
    application.services.every((service) => service.status === "running")
  )
    return ["健康", "is-healthy"];
  if (application.health.status === "unhealthy")
    return ["健康检查失败", "is-attention"];
  if (application.health.status === "unavailable")
    return ["健康检查不可用", "is-attention"];
  if (application.services.some((service) => service.status === "failed"))
    return ["服务失败", "is-attention"];
  if (!application.health.configured) return ["未配置健康检查", "is-attention"];
  return ["状态未知", "is-attention"];
}

function latestRelease(application) {
  return application.releases?.[0] || null;
}

function avatarClass(index) {
  return ["app-blue", "app-orange", "app-green"][index % 3];
}

function renderDerivedServiceState() {
  const services = applications.flatMap((application) =>
    application.services.map((service) => ({ service, application })),
  );
  $("#nav-app-count").textContent = String(applications.length);
  $("#nav-service-count").textContent = String(services.length);
  $("#applications-count").textContent = `已纳管 ${applications.length} 个应用`;
  $("#service-list").hidden = false;
  $("#event-list").hidden = false;
  $("#service-list").innerHTML = services.length
    ? services
        .map(({ service, application }) => {
          const healthy = service.status === "running";
          const detail =
            service.status === "unknown"
              ? "状态不可读取"
              : `${service.activeState || service.status}${service.subState ? ` · ${service.subState}` : ""}`;
          const description = service.description || "未提供功能介绍";
          return `<li><span class="service-health ${healthy ? "is-healthy" : "is-attention"}"></span><div><strong>${escapeHTML(service.name)}</strong><span>${escapeHTML(description)}</span><small>${escapeHTML(application.name)} · ${escapeHTML(detail)}</small></div><code>${escapeHTML(service.status || "unknown")}</code></li>`;
        })
        .join("")
    : '<li><span class="service-health is-attention"></span><div><strong>暂无登记服务</strong><span>应用配置为空或未设置 systemd unit。</span></div><code>--</code></li>';
  $("#event-list").innerHTML =
    '<li><span class="event-icon info">i</span><div><strong>当前没有可读取的运行事件</strong><span>事件检索不属于版本 001 的只读接口。</span></div></li>';
}

function renderInventoryUnavailable(message) {
  $("#nav-app-count").textContent = "--";
  $("#nav-service-count").textContent = "--";
  $("#applications-count").textContent = "应用清单不可用";
  $("#application-list").innerHTML =
    `<div class="application-table table-header" aria-hidden="true"><span>应用</span><span>状态</span><span>当前版本</span><span>最近发布</span><span>访问</span><span></span></div><div class="application-table"><span class="app-name"><span><strong>无法读取应用清单</strong><small>${escapeHTML(message)}</small></span></span><span>--</span><code>--</code><time>--</time><span>--</span><span></span></div>`;
  $("#service-list").hidden = false;
  $("#service-list").innerHTML =
    `<li><span class="service-health is-attention"></span><div><strong>服务状态不可用</strong><span>${escapeHTML(message)}</span></div><code>--</code></li>`;
  $("#event-list").hidden = false;
  $("#event-list").innerHTML =
    '<li><span class="event-icon info">i</span><div><strong>运行事件尚未接入</strong><span>版本 001 仅提供主机、应用和发布记录读取。</span></div></li>';
}

function groupApplications() {
  const groups = new Map();
  applications.forEach((application, index) => {
    const name = application.group?.trim() || "未分组";
    if (!groups.has(name)) groups.set(name, []);
    groups.get(name).push({ application, index });
  });
  return [...groups.entries()];
}

function renderApplicationRow(application, index) {
  const [status, className] = applicationHealth(application);
  const release = latestRelease(application);
  const releaseTime = release ? formatDateTime(release.finishedAt) : "暂无记录";
  const version = release?.version || "--";
  const selected = application.id === selectedAppID;
  const description = application.description || "未提供功能介绍";
  const link = application.publicUrl
    ? `<a class="quick-link" href="${escapeHTML(application.publicUrl)}" target="_blank" rel="noreferrer" aria-label="打开 ${escapeHTML(application.name)}" title="打开 ${escapeHTML(application.name)}"><span>打开</span></a>`
    : '<span class="quick-link is-disabled" aria-label="无公网入口">--</span>';
  return `<div class="application-table app-row${selected ? " is-selected" : ""}" data-app="${escapeHTML(application.id)}"><button class="app-select" type="button" data-app="${escapeHTML(application.id)}" aria-label="查看 ${escapeHTML(application.name)} 应用详情"><span class="app-name"><span class="app-avatar ${avatarClass(index)}">${escapeHTML(application.name.slice(0, 1).toUpperCase())}</span><span><strong>${escapeHTML(application.name)}</strong><small title="${escapeHTML(description)}">${escapeHTML(description)}</small></span></span></button><span class="status-pill ${className}"><i></i>${escapeHTML(status)}</span><code>${escapeHTML(version)}</code><time>${escapeHTML(releaseTime)}</time>${link}<span class="row-action" aria-hidden="true">›</span></div>`;
}

function renderApplications() {
  const container = $("#application-list");
  const header =
    '<div class="application-table table-header" aria-hidden="true"><span>应用</span><span>状态</span><span>当前版本</span><span>最近发布</span><span>访问</span><span></span></div>';
  if (!applications.length) {
    container.innerHTML = `${header}<div class="application-table"><span class="app-name"><span><strong>暂无已纳管应用</strong><small>在配置文件 applications 中登记后会显示在这里。</small></span></span><span>--</span><code>--</code><time>--</time><span>--</span><span></span></div>`;
    renderApplicationDetail(null);
    renderDerivedServiceState();
    return;
  }
  const groups = groupApplications()
    .map(([name, entries], groupIndex) => {
      const collapsed = collapsedApplicationGroups.has(name);
      const attentionCount = entries.filter(
        ({ application }) =>
          applicationHealth(application)[1] === "is-attention",
      ).length;
      const groupID = `application-group-${groupIndex}`;
      const rows = entries
        .map(({ application, index }) =>
          renderApplicationRow(application, index),
        )
        .join("");
      return `<section class="application-group${collapsed ? " is-collapsed" : ""}"><h3><button class="application-group-toggle" type="button" data-group="${escapeHTML(name)}" aria-expanded="${String(!collapsed)}" aria-controls="${groupID}"><span class="application-group-name"><span class="group-chevron" aria-hidden="true"></span><span>${escapeHTML(name)}</span></span><span class="application-group-stats"><span>${entries.length} 个应用</span><span class="${attentionCount ? "has-attention" : ""}">需关注 ${attentionCount}</span></span></button></h3><div id="${groupID}"${collapsed ? " hidden" : ""}>${rows}</div></section>`;
    })
    .join("");
  container.innerHTML = `${header}${groups}`;
  renderDerivedServiceState();
}

function releaseStatusClass(status) {
  return status === "success" ? "is-healthy" : "is-attention";
}

function releaseStatusLabel(status) {
  return status === "success" ? "成功" : status || "未知";
}

function renderApplicationDetail(application) {
  const avatar = $("#detail-avatar");
  if (!application) {
    avatar.textContent = "--";
    $("#detail-name").textContent = "暂无应用";
    $("#detail-description").textContent = "没有可显示的应用介绍。";
    $("#detail-url").textContent = "等待配置";
    $("#detail-url").removeAttribute("href");
    $("#detail-check").textContent = "未配置健康检查";
    $("#detail-resource").textContent = "没有可读取的服务信息";
    $("#detail-commit").textContent = "--";
    $("#detail-branch").textContent = "";
    $("#deployments-title").textContent = "发布记录";
    $("#deployments .eyebrow").textContent = "只读";
    $("#deployment-list").innerHTML =
      '<tr><td colspan="6">暂无应用或发布记录。</td></tr>';
    return;
  }
  const index = applications.findIndex((item) => item.id === application.id);
  avatar.textContent = application.name.slice(0, 1).toUpperCase();
  avatar.className = `app-avatar ${avatarClass(index)}`;
  $("#detail-name").textContent = application.name;
  $("#detail-description").textContent =
    application.description || "未提供功能介绍。";
  const detailURL = $("#detail-url");
  if (application.publicUrl) {
    detailURL.hidden = false;
    detailURL.href = application.publicUrl;
    detailURL.childNodes[0].nodeValue = `${new URL(application.publicUrl).host} `;
  } else {
    detailURL.hidden = true;
    detailURL.removeAttribute("href");
  }
  const healthTime = application.health.checkedAt
    ? ` · ${formatDateTime(application.health.checkedAt)}`
    : "";
  $("#detail-check").textContent =
    `${applicationHealth(application)[0]}${healthTime}`;
  $("#detail-resource").textContent =
    `${application.services.length} 个登记服务 · ${application.services.filter((service) => service.status === "running").length} 个运行中`;
  const release = latestRelease(application);
  $("#detail-commit").textContent = release?.commit || "暂无记录";
  $("#detail-branch").textContent = " 只读发布记录";
  $("#deployments-title").textContent = `${application.name} 发布记录`;
  $("#deployments .eyebrow").textContent = application.name;
  const releases = application.releases || [];
  $("#deployment-list").innerHTML = releases.length
    ? releases
        .map(
          (release, index) =>
            `<tr${index === 0 ? ' class="current-release"' : ""}><td><div class="release-cell"><code>${escapeHTML(release.version)}</code>${index === 0 ? "<span>最新记录</span>" : ""}</div><small>${escapeHTML(release.commit)}</small></td><td><span class="status-pill ${releaseStatusClass(release.status)}"><i></i>${escapeHTML(releaseStatusLabel(release.status))}</span></td><td><time>${escapeHTML(formatDateTime(release.finishedAt))}</time></td><td><span class="person"><span class="person-avatar">${escapeHTML((release.actor || "-").slice(0, 1).toUpperCase())}</span>${escapeHTML(release.actor || "未记录")}</span></td><td class="numeric">${escapeHTML(formatDuration(release.startedAt, release.finishedAt))}</td><td><button class="quiet-button" type="button" disabled title="当前版本只支持读取发布记录">只读</button></td></tr>`,
        )
        .join("")
    : '<tr><td colspan="6">暂无发布记录。</td></tr>';
}

function formatDuration(start, end) {
  const duration = Math.max(0, new Date(end) - new Date(start));
  if (!Number.isFinite(duration)) return "--";
  return `${Math.floor(duration / 60000)}m ${Math.floor((duration % 60000) / 1000)}s`;
}

async function selectApplication(id) {
  const application = applications.find((item) => item.id === id);
  if (!application) return;
  selectedAppID = id;
  renderApplications();
  renderApplicationDetail(application);
  if (application.releases) return;
  try {
    const payload = await requestJSON(
      `/api/v1/applications/${encodeURIComponent(id)}/releases`,
    );
    application.releases = payload.releases || [];
  } catch {
    application.releases = [];
    showToast("发布记录暂时不可读取。", true);
  }
  renderApplications();
  renderApplicationDetail(application);
}

async function loadApplications() {
  const payload = await requestJSON("/api/v1/applications");
  applications = payload.applications || [];
  applications.forEach((application) => {
    application.releases = null;
  });
  selectedAppID = applications[0]?.id || null;
  renderApplications();
  if (selectedAppID) await selectApplication(selectedAppID);
}

async function loadRange(start, end) {
  const feedback = $("#range-feedback");
  feedback.classList.remove("is-error");
  feedback.textContent = "正在加载历史指标…";
  $("#time-range-form button[type='submit']").disabled = true;
  try {
    const query = new URLSearchParams({
      start: start.toISOString(),
      end: end.toISOString(),
    });
    activeRange = buildRange(await requestJSON(`/api/v1/metrics?${query}`));
    renderTrend();
    if ($("#range-data-dialog").open) renderRangeData();
    feedback.textContent = activeRange.buckets.length
      ? "已加载后端指标数据。"
      : "该时间范围暂无指标数据。";
  } catch (error) {
    activeRange = null;
    renderTrend();
    feedback.classList.add("is-error");
    feedback.textContent = `无法加载指标：${error.message}`;
  } finally {
    $("#time-range-form button[type='submit']").disabled = false;
  }
}

function bindAuthentication() {
  $("#login-form").addEventListener("submit", async (event) => {
    event.preventDefault();
    const form = event.currentTarget;
    const button = form.querySelector("button[type='submit']");
    const username = $("#login-username").value.trim();
    const password = $("#login-password").value;
    $("#login-error").textContent = "";
    button.disabled = true;
    button.textContent = "登录中";
    try {
      const session = await requestJSON("/api/v1/session", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ username, password }),
      });
      $("#login-password").value = "";
      showDashboard(session.username);
      await loadDashboard();
    } catch (error) {
      $("#login-password").value = "";
      $("#login-error").textContent =
        error.status === 429
          ? "登录尝试过多，请稍后再试。"
          : error.status === 401
            ? "账号或密码不正确。"
            : "认证服务暂时不可用。";
      $("#login-password").focus();
    } finally {
      button.disabled = false;
      button.textContent = "登录";
    }
  });

  $("#profile-button").addEventListener("click", () => {
    const menu = $("#account-menu");
    menu.hidden = !menu.hidden;
    $("#profile-button").setAttribute("aria-expanded", String(!menu.hidden));
  });
  $("#logout-button").addEventListener("click", async (event) => {
    event.currentTarget.disabled = true;
    try {
      await requestJSON("/api/v1/session", { method: "DELETE" });
    } finally {
      currentUsername = "";
      showLogin();
      event.currentTarget.disabled = false;
    }
  });
  document.addEventListener("click", (event) => {
    if (event.target.closest(".account-control")) return;
    $("#account-menu").hidden = true;
    $("#profile-button").setAttribute("aria-expanded", "false");
  });
}

function bindInteractions() {
  $$(".nav-link[data-target]").forEach((button) =>
    button.addEventListener("click", () => {
      const target = document.getElementById(button.dataset.target);
      if (!target) return;
      $$(".nav-link[data-target]").forEach((item) => {
        item.classList.toggle("is-active", item === button);
        item.removeAttribute("aria-current");
      });
      button.setAttribute("aria-current", "page");
      $("#page-label").textContent = pageLabels[button.dataset.target];
      target.scrollIntoView({
        behavior: window.matchMedia("(prefers-reduced-motion: reduce)").matches
          ? "auto"
          : "smooth",
        block: "start",
      });
    }),
  );
  $$("[data-trend-metric]").forEach((button) =>
    button.addEventListener("click", () => {
      selectedTrendMetric = button.dataset.trendMetric;
      renderTrend();
    }),
  );
  $("#application-list").addEventListener("click", (event) => {
    const groupButton = event.target.closest(".application-group-toggle");
    if (groupButton) {
      const groupName = groupButton.dataset.group;
      const groupBody = document.getElementById(
        groupButton.getAttribute("aria-controls"),
      );
      const expanded = groupButton.getAttribute("aria-expanded") === "true";
      groupButton.setAttribute("aria-expanded", String(!expanded));
      groupButton
        .closest(".application-group")
        .classList.toggle("is-collapsed", expanded);
      groupBody.hidden = expanded;
      if (expanded) collapsedApplicationGroups.add(groupName);
      else collapsedApplicationGroups.delete(groupName);
      return;
    }
    const button = event.target.closest(".app-select");
    if (button) selectApplication(button.dataset.app);
  });
  $("#open-range-data").addEventListener("click", openRangeData);
  $("#time-range-form").addEventListener("submit", (event) => {
    event.preventDefault();
    const startInput = $("#range-start");
    const endInput = $("#range-end");
    const start = new Date(startInput.value);
    const end = new Date(endInput.value);
    const invalid =
      !startInput.value ||
      !endInput.value ||
      Number.isNaN(start.getTime()) ||
      Number.isNaN(end.getTime()) ||
      end <= start;
    const message = invalid ? "请填写结束时间晚于开始时间的分钟级范围。" : "";
    startInput.setCustomValidity(message);
    endInput.setCustomValidity(message);
    startInput.setAttribute("aria-invalid", String(Boolean(message)));
    endInput.setAttribute("aria-invalid", String(Boolean(message)));
    if (message) {
      $("#range-feedback").classList.add("is-error");
      $("#range-feedback").textContent = message;
      return;
    }
    loadRange(start, end);
  });
  $("#open-deployments").addEventListener("click", () =>
    document
      .getElementById("deployments")
      .scrollIntoView({ behavior: "smooth", block: "start" }),
  );
  $("#deploy-button").addEventListener("click", () =>
    showToast("当前版本仅支持读取发布记录。", true),
  );
  $("#detail-menu").addEventListener("click", () =>
    showToast("当前版本没有可执行的应用操作。", true),
  );
  $$("[data-demo-action]").forEach((button) => {
    button.disabled = true;
    button.setAttribute("aria-disabled", "true");
    button.title = "当前版本尚未提供此功能";
  });
  $("#theme-toggle").addEventListener("click", () =>
    document.body.classList.toggle("dark"),
  );
  $("#refresh-button").addEventListener("click", async (event) => {
    event.currentTarget.classList.add("is-refreshing");
    try {
      renderOverview(await requestJSON("/api/v1/overview"));
      await loadApplications();
      setDataBadge("后端已连接");
      showToast("状态已从后端刷新。");
    } catch (error) {
      setOverviewUnavailable(`后端不可用：${error.message}`);
      renderInventoryUnavailable(`后端不可用：${error.message}`);
      setDataBadge("后端不可用", true);
      showToast(`刷新失败：${error.message}`, true);
    } finally {
      event.currentTarget.classList.remove("is-refreshing");
    }
  });
  bindTrendChartInteractions();
}

async function loadDashboard() {
  try {
    const [overview] = await Promise.all([
      requestJSON("/api/v1/overview"),
      loadApplications(),
      loadRange(
        new Date($("#range-start").value),
        new Date($("#range-end").value),
      ),
    ]);
    renderOverview(overview);
    setDataBadge("后端已连接");
  } catch (error) {
    if (error.status === 401) return;
    setOverviewUnavailable(`后端不可用：${error.message}`);
    renderInventoryUnavailable(`后端不可用：${error.message}`);
    setDataBadge("后端不可用", true);
    showToast(`无法连接后端：${error.message}`, true);
  }
}

async function initialize() {
  const end = new Date();
  end.setSeconds(0, 0);
  const start = new Date(end.getTime() - 24 * 60 * 60 * 1000);
  $("#range-start").value = formatDateInput(start);
  $("#range-end").value = formatDateInput(end);
  renderEmptyTrend("正在连接后端");
  bindInteractions();
  bindAuthentication();
  try {
    const session = await requestJSON("/api/v1/session");
    if (!session.authenticated) {
      showLogin();
      return;
    }
    showDashboard(session.username);
    await loadDashboard();
  } catch (error) {
    showLogin("无法验证登录状态，请稍后重试。");
  }
}

initialize();
