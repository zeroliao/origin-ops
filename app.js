const applications = {
  sub2api: {
    name: "Sub2API",
    version: "v2.4.1",
    avatar: "S",
    avatarClass: "app-blue",
    url: "sub2api.example.com",
    check: "/health · 200 OK · 08:59:51",
    resource: "CPU 6.4% · 内存 684 MB",
    commit: "a8c32f1",
    branch: "main",
  },
  "sub2api-cf": {
    name: "Sub2API-CF",
    version: "v1.8.0",
    avatar: "C",
    avatarClass: "app-orange",
    url: "sub2api-cf.example.com",
    check: "/status · 200 OK · 08:59:46",
    resource: "边缘请求 1,284/min · 错误率 0.02%",
    commit: "c37d10a",
    branch: "main",
  },
  gateway: {
    name: "Nginx Gateway",
    version: "1.26.1",
    avatar: "N",
    avatarClass: "app-green",
    url: "gateway.example.com",
    check: "/health · 200 OK · 08:59:43",
    resource: "CPU 0.8% · 内存 22 MB",
    commit: "1.26.1",
    branch: "systemd",
  },
};

const deploymentHistory = {
  sub2api: [
    {
      version: "v2.4.1",
      commit: "a8c32f1 · api + worker",
      status: "已验证",
      statusClass: "is-healthy",
      time: "今天 09:42",
      duration: "1m 16s",
      current: true,
    },
    {
      version: "v2.4.0",
      commit: "6e54ba9 · api + worker",
      status: "已验证",
      statusClass: "is-healthy",
      time: "8 月 13 日 21:27",
      duration: "1m 08s",
      rollbackCommit: "6e54ba9",
    },
    {
      version: "v2.3.9",
      commit: "f19d8d2 · api + worker",
      status: "已回滚",
      statusClass: "is-rolled-back",
      time: "8 月 10 日 14:05",
      duration: "1m 22s",
      rollbackCommit: "f19d8d2",
    },
  ],
  "sub2api-cf": [
    {
      version: "v1.8.0",
      commit: "c37d10a · worker bundle",
      status: "已验证",
      statusClass: "is-healthy",
      time: "昨天 22:10",
      duration: "34s",
      current: true,
    },
    {
      version: "v1.7.9",
      commit: "0d41cab · worker bundle",
      status: "已验证",
      statusClass: "is-healthy",
      time: "8 月 11 日 18:02",
      duration: "31s",
      rollbackCommit: "0d41cab",
    },
    {
      version: "v1.7.8",
      commit: "bb6428e · worker bundle",
      status: "已验证",
      statusClass: "is-healthy",
      time: "8 月 05 日 11:46",
      duration: "29s",
      rollbackCommit: "bb6428e",
    },
  ],
  gateway: [
    {
      version: "1.26.1",
      commit: "nginx:1.26.1 · systemd",
      status: "需关注",
      statusClass: "is-attention",
      time: "8 月 13 日 16:35",
      duration: "9s",
      current: true,
    },
    {
      version: "1.26.0",
      commit: "nginx:1.26.0 · systemd",
      status: "已验证",
      statusClass: "is-healthy",
      time: "7 月 26 日 13:20",
      duration: "10s",
      rollbackCommit: "nginx:1.26.0",
    },
    {
      version: "1.25.5",
      commit: "nginx:1.25.5 · systemd",
      status: "已验证",
      statusClass: "is-healthy",
      time: "7 月 04 日 08:46",
      duration: "11s",
      rollbackCommit: "nginx:1.25.5",
    },
  ],
};

const rangeData = {
  "1h": {
    label: "1 小时",
    count: "12 条采样记录",
    cadence: "每 5 分钟",
    axis: ["11:00", "11:30", "现在"],
    metrics: {
      cpu: {
        label: "CPU 使用率",
        current: "24.6%",
        color: "#1e68cf",
        values: [19, 22, 18, 27, 24, 31, 26, 23, 28, 21, 25, 24],
      },
      memory: {
        label: "内存使用",
        current: "2.86 GB",
        color: "#117a70",
        values: [
          2.72, 2.73, 2.74, 2.78, 2.8, 2.81, 2.8, 2.83, 2.85, 2.84, 2.86, 2.86,
        ],
      },
      network: {
        label: "网络出站",
        current: "4.1 MB/s",
        color: "#6b5fc7",
        values: [3.1, 3.4, 2.8, 4.6, 4.1, 5.2, 3.9, 4.7, 3.6, 4.3, 4.4, 4.1],
      },
    },
    summary: [
      ["平均 CPU", "23.7%", "较前 1 小时 +1.2%"],
      ["峰值", "31.0%", "11:25 出现"],
      ["内存变化", "+142 MB", "区间起止对比"],
      ["磁盘增长", "+18 MB", "持续监控中"],
    ],
    details: [
      ["11:00", "19.0%", "2.72 GB", "43.18 GB", "3.1 MB/s"],
      ["11:10", "18.0%", "2.74 GB", "43.19 GB", "2.8 MB/s"],
      ["11:20", "24.0%", "2.80 GB", "43.19 GB", "4.1 MB/s"],
      ["11:30", "31.0%", "2.81 GB", "43.20 GB", "5.2 MB/s"],
      ["11:40", "23.0%", "2.83 GB", "43.20 GB", "4.7 MB/s"],
      ["11:50", "25.0%", "2.86 GB", "43.20 GB", "4.4 MB/s"],
      ["现在", "24.6%", "2.86 GB", "43.20 GB", "4.1 MB/s"],
    ],
  },
  "24h": {
    label: "24 小时",
    count: "96 条采样记录",
    cadence: "每 15 分钟",
    axis: ["昨天 12:00", "00:00", "现在"],
    metrics: {
      cpu: {
        label: "CPU 使用率",
        current: "23.8%",
        color: "#1e68cf",
        values: [
          17, 23, 21, 29, 25, 19, 27, 31, 24, 38, 33, 27, 36, 42, 29, 24, 34,
          28, 46, 35, 31, 39, 26, 24,
        ],
      },
      memory: {
        label: "内存使用",
        current: "2.81 GB",
        color: "#117a70",
        values: [
          2.63, 2.64, 2.66, 2.67, 2.68, 2.7, 2.72, 2.71, 2.73, 2.75, 2.74, 2.77,
          2.78, 2.8, 2.79, 2.8, 2.82, 2.84, 2.83, 2.82, 2.8, 2.81, 2.8, 2.81,
        ],
      },
      network: {
        label: "网络出站",
        current: "3.8 MB/s",
        color: "#6b5fc7",
        values: [
          2.1, 2.8, 1.9, 3.2, 3.8, 2.5, 3.1, 4.4, 2.9, 5.1, 4.3, 3.7, 5.4, 4.7,
          3.6, 4.9, 3.1, 6.2, 4.8, 3.9, 5.3, 3.7, 4.2, 3.8,
        ],
      },
    },
    summary: [
      ["平均 CPU", "22.6%", "较上个周期 -3.4%"],
      ["峰值", "46.1%", "14:20 出现"],
      ["内存变化", "+128 MB", "区间起止对比"],
      ["磁盘增长", "+0.6 GB", "持续监控中"],
    ],
    details: [
      ["昨天 12:00", "17.2%", "2.63 GB", "42.60 GB", "2.1 MB/s"],
      ["昨天 16:00", "25.1%", "2.68 GB", "42.71 GB", "3.8 MB/s"],
      ["昨天 20:00", "19.4%", "2.72 GB", "42.84 GB", "2.5 MB/s"],
      ["00:00", "31.3%", "2.75 GB", "42.93 GB", "5.1 MB/s"],
      ["04:00", "24.2%", "2.79 GB", "43.01 GB", "3.6 MB/s"],
      ["08:00", "38.7%", "2.83 GB", "43.12 GB", "5.3 MB/s"],
      ["现在", "23.8%", "2.81 GB", "43.20 GB", "3.8 MB/s"],
    ],
  },
  "7d": {
    label: "7 天",
    count: "336 条采样记录",
    cadence: "每 30 分钟",
    axis: ["8 月 08 日", "8 月 11 日", "现在"],
    metrics: {
      cpu: {
        label: "CPU 使用率",
        current: "20.1%",
        color: "#1e68cf",
        values: [
          15, 19, 27, 22, 18, 31, 24, 21, 34, 28, 25, 20, 37, 29, 22, 18, 33,
          26, 19, 30, 23, 17, 28, 20,
        ],
      },
      memory: {
        label: "内存使用",
        current: "2.74 GB",
        color: "#117a70",
        values: [
          2.45, 2.47, 2.49, 2.52, 2.55, 2.57, 2.6, 2.62, 2.64, 2.66, 2.68, 2.69,
          2.7, 2.71, 2.72, 2.73, 2.73, 2.74, 2.74, 2.75, 2.74, 2.74, 2.73, 2.74,
        ],
      },
      network: {
        label: "网络出站",
        current: "3.4 MB/s",
        color: "#6b5fc7",
        values: [
          2.3, 2.7, 3.1, 2.5, 3.8, 3.2, 2.9, 4.1, 3.3, 2.8, 4.4, 3.7, 3.1, 4.6,
          3.8, 3.2, 4.3, 3.5, 2.9, 4.0, 3.1, 2.6, 3.7, 3.4,
        ],
      },
    },
    summary: [
      ["平均 CPU", "21.8%", "较上个 7 天 -1.8%"],
      ["峰值", "37.4%", "8 月 12 日出现"],
      ["内存变化", "+294 MB", "区间起止对比"],
      ["磁盘增长", "+3.8 GB", "持续监控中"],
    ],
    details: [
      ["8 月 08 日", "15.3%", "2.45 GB", "39.40 GB", "2.3 MB/s"],
      ["8 月 09 日", "27.0%", "2.52 GB", "39.92 GB", "3.1 MB/s"],
      ["8 月 10 日", "18.4%", "2.60 GB", "40.48 GB", "3.8 MB/s"],
      ["8 月 11 日", "34.1%", "2.66 GB", "41.15 GB", "4.4 MB/s"],
      ["8 月 12 日", "37.4%", "2.70 GB", "41.89 GB", "4.6 MB/s"],
      ["8 月 13 日", "22.8%", "2.73 GB", "42.61 GB", "3.7 MB/s"],
      ["现在", "20.1%", "2.74 GB", "43.20 GB", "3.4 MB/s"],
    ],
  },
};

const pageLabels = {
  overview: "总览",
  applications: "应用",
  services: "服务",
  deployments: "发布记录",
  events: "通知",
};
let selectedApp = "sub2api";
let activeRange;
let selectedTrendMetric = "cpu";
let activeChartBounds;
let toastTimeout;

const $ = (selector) => document.querySelector(selector);
const $$ = (selector) => [...document.querySelectorAll(selector)];

function showToast(message, isError = false) {
  const toast = $("#toast");
  toast.textContent = message;
  toast.classList.toggle("is-error", isError);
  toast.classList.add("is-visible");
  window.clearTimeout(toastTimeout);
  toastTimeout = window.setTimeout(
    () => toast.classList.remove("is-visible"),
    3200,
  );
}

function getChartBounds(metricKey, values) {
  let minimum = Math.min(...values);
  let maximum = Math.max(...values);
  if (metricKey === "memory") {
    minimum = Math.floor(minimum * 10) / 10;
    maximum = Math.ceil(maximum * 10) / 10;
    if (maximum - minimum < 0.3) {
      minimum -= 0.1;
      maximum += 0.1;
    }
  }
  return { minimum, maximum };
}

function toChartPoints(
  values,
  bounds = getChartBounds(selectedTrendMetric, values),
) {
  const width = 672;
  const startX = 24;
  const top = 24;
  const bottom = 156;
  const span = Math.max(bounds.maximum - bounds.minimum, 1);
  return values.map((value, index) => {
    const x = startX + (width * index) / (values.length - 1);
    const y = top + ((bounds.maximum - value) / span) * (bottom - top);
    return [Number(x.toFixed(1)), Number(y.toFixed(1))];
  });
}

function renderTrendYAxis(metricKey, values) {
  const leftAxis = $("#trend-y-axis-left");
  const rightAxis = $("#trend-y-axis-right");
  const stage = $("#chart-stage");
  const showMemoryAxes = metricKey === "memory";
  stage.classList.toggle("has-dual-axis", showMemoryAxes);
  leftAxis.hidden = !showMemoryAxes;
  rightAxis.hidden = !showMemoryAxes;
  if (!showMemoryAxes) return;

  const ticks = Array.from(
    { length: 4 },
    (_, index) =>
      activeChartBounds.maximum -
      ((activeChartBounds.maximum - activeChartBounds.minimum) * index) / 3,
  );
  leftAxis.innerHTML = ticks
    .map((value) => `<span>${value.toFixed(2)} GB</span>`)
    .join("");
  rightAxis.innerHTML = ticks
    .map((value) => `<span>${Math.round((value / 8) * 100)}%</span>`)
    .join("");
}

function hideTrendTooltip() {
  $("#chart-tooltip").hidden = true;
  $("#trend-hover-guide").classList.remove("is-visible");
  $("#trend-hover-point").classList.remove("is-visible");
}

function showTrendTooltip(index) {
  const range = activeRange;
  const metric = range.metrics[selectedTrendMetric];
  const values = metric.values;
  const safeIndex = Math.max(0, Math.min(index, values.length - 1));
  const points = toChartPoints(values, activeChartBounds);
  const [x, y] = points[safeIndex];
  const time = new Date(
    range.start.getTime() +
      (range.end.getTime() - range.start.getTime()) *
        (safeIndex / (values.length - 1)),
  );
  const value = values[safeIndex];
  const detail =
    selectedTrendMetric === "memory"
      ? `${formatMetricValue("memory", value)} (${Math.round((value / 8) * 100)}%)`
      : formatMetricValue(selectedTrendMetric, value);
  const tooltip = $("#chart-tooltip");

  $("#trend-hover-guide").setAttribute("x1", x);
  $("#trend-hover-guide").setAttribute("x2", x);
  $("#trend-hover-point").setAttribute("cx", x);
  $("#trend-hover-point").setAttribute("cy", y);
  $("#trend-hover-guide").classList.add("is-visible");
  $("#trend-hover-point").classList.add("is-visible");
  tooltip.style.setProperty("--tooltip-x", `${(x / 720) * 100}%`);
  tooltip.style.setProperty("--tooltip-y", `${(y / 180) * 100}%`);
  tooltip.classList.toggle("is-below", y < 72);
  tooltip.innerHTML = `<strong>${formatDateTime(time)}</strong><span>${metric.label} ${detail}</span>`;
  tooltip.hidden = false;
}

function getTrendIndexFromPointer(event) {
  const chart = $("#trend-chart");
  const values = activeRange.metrics[selectedTrendMetric].values;
  const bounds = chart.getBoundingClientRect();
  const chartX = ((event.clientX - bounds.left) / bounds.width) * 720;
  const ratio = Math.max(0, Math.min(1, (chartX - 24) / 672));
  return Math.round(ratio * (values.length - 1));
}

function bindTrendChartInteractions() {
  const chart = $("#trend-chart");
  let keyboardIndex = 0;
  chart.addEventListener("pointermove", (event) =>
    showTrendTooltip(getTrendIndexFromPointer(event)),
  );
  chart.addEventListener("pointerdown", (event) =>
    showTrendTooltip(getTrendIndexFromPointer(event)),
  );
  chart.addEventListener("pointerleave", hideTrendTooltip);
  chart.addEventListener("focus", () => showTrendTooltip(keyboardIndex));
  chart.addEventListener("blur", hideTrendTooltip);
  chart.addEventListener("keydown", (event) => {
    const values = activeRange.metrics[selectedTrendMetric].values;
    if (event.key === "Home") keyboardIndex = 0;
    else if (event.key === "End") keyboardIndex = values.length - 1;
    else if (event.key === "ArrowLeft")
      keyboardIndex = Math.max(0, keyboardIndex - 1);
    else if (event.key === "ArrowRight")
      keyboardIndex = Math.min(values.length - 1, keyboardIndex + 1);
    else return;
    event.preventDefault();
    showTrendTooltip(keyboardIndex);
  });
}

function pad(value) {
  return String(value).padStart(2, "0");
}

function formatDateInput(date) {
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}T${pad(date.getHours())}:${pad(date.getMinutes())}`;
}

function formatDateTime(date) {
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())} ${pad(date.getHours())}:${pad(date.getMinutes())}`;
}

function formatAxisDate(date) {
  return `${pad(date.getMonth() + 1)}-${pad(date.getDate())} ${pad(date.getHours())}:${pad(date.getMinutes())}`;
}

function formatMetricValue(metricKey, value) {
  if (metricKey === "cpu") return `${value.toFixed(1)}%`;
  if (metricKey === "memory") return `${value.toFixed(2)} GB`;
  return `${value.toFixed(1)} MB/s`;
}

function formatMetricChange(metricKey, value) {
  const sign = value >= 0 ? "+" : "";
  return `${sign}${formatMetricValue(metricKey, value)}`;
}

function resolveTemplateKey(durationMinutes) {
  if (durationMinutes <= 90) return "1h";
  if (durationMinutes <= 48 * 60) return "24h";
  return "7d";
}

function resolveCadence(durationMinutes) {
  if (durationMinutes <= 6 * 60) return 5;
  if (durationMinutes <= 48 * 60) return 15;
  return 30;
}

function interpolateValue(values, ratio) {
  const index = Math.round(ratio * (values.length - 1));
  return values[index];
}

function buildRange(start, end) {
  const durationMinutes = Math.round((end.getTime() - start.getTime()) / 60000);
  const template = rangeData[resolveTemplateKey(durationMinutes)];
  const cadenceMinutes = resolveCadence(durationMinutes);
  const count = Math.floor(durationMinutes / cadenceMinutes) + 1;
  const ratios = [0, 1 / 6, 2 / 6, 3 / 6, 4 / 6, 5 / 6, 1];
  const startDisk = Math.max(38, 43.2 - durationMinutes / 4200);
  const details = ratios.map((ratio) => {
    const sampleTime = new Date(
      start.getTime() + (end.getTime() - start.getTime()) * ratio,
    );
    const cpu = interpolateValue(template.metrics.cpu.values, ratio);
    const memory = interpolateValue(template.metrics.memory.values, ratio);
    const network = interpolateValue(template.metrics.network.values, ratio);
    const disk = startDisk + (43.2 - startDisk) * ratio;
    return [
      formatDateTime(sampleTime),
      formatMetricValue("cpu", cpu),
      formatMetricValue("memory", memory),
      `${disk.toFixed(2)} GB`,
      formatMetricValue("network", network),
    ];
  });

  return {
    start,
    end,
    label: `${formatDateTime(start)} 至 ${formatDateTime(end)}`,
    count: `${count} 条采样记录`,
    cadence: `每 ${cadenceMinutes} 分钟`,
    axis: [
      formatAxisDate(start),
      formatAxisDate(new Date((start.getTime() + end.getTime()) / 2)),
      formatAxisDate(end),
    ],
    metrics: template.metrics,
    details,
  };
}

function getPeakRisk(metricKey, peak) {
  const limits = {
    cpu: { medium: 65, high: 85, unit: "%" },
    memory: { medium: 5.2, high: 6.8, unit: " GB" },
    network: { medium: 5, high: 8, unit: " MB/s" },
  };
  const limit = limits[metricKey];
  if (peak >= limit.high)
    return {
      label: "高风险",
      className: "risk-high",
      note: `达到 ${limit.high}${limit.unit} 阈值`,
    };
  if (peak >= limit.medium)
    return {
      label: "中风险",
      className: "risk-medium",
      note: `达到 ${limit.medium}${limit.unit} 阈值`,
    };
  return {
    label: "低风险",
    className: "risk-low",
    note: `低于 ${limit.medium}${limit.unit} 阈值`,
  };
}

function updateCurrentSnapshotClock(now = new Date()) {
  const dateTime = formatDateTime(now);
  const inputDateTime = formatDateInput(now);
  $("#server-time").textContent = dateTime;
  $("#server-time").dateTime = inputDateTime;
  $("#current-metrics-time").textContent =
    `更新于 ${pad(now.getHours())}:${pad(now.getMinutes())}`;
  $("#current-metrics-time").dateTime = inputDateTime;
}

function renderTrend() {
  const range = activeRange;
  const metric = range.metrics[selectedTrendMetric];
  activeChartBounds = getChartBounds(selectedTrendMetric, metric.values);
  const points = toChartPoints(metric.values, activeChartBounds);
  const pointText = points.map(([x, y]) => `${x},${y}`).join(" ");
  const lastPoint = points.at(-1);
  $("#trend-line").setAttribute("points", pointText);
  $("#trend-area").setAttribute("points", `${pointText} 696,156 24,156`);
  $("#trend-point").setAttribute("cx", lastPoint[0]);
  $("#trend-point").setAttribute("cy", lastPoint[1]);
  $(".trend-chart").style.color = metric.color;
  $("#trend-chart-desc").textContent =
    `${range.label}内的${metric.label}变化趋势`;
  $("#trend-chart").setAttribute(
    "aria-label",
    `${metric.label}趋势图。使用方向键查看每个采样点的具体数据。`,
  );
  $("#trend-range-name").textContent = range.label;
  $("#trend-sample-note").textContent = `${range.count} · ${range.cadence}`;
  $("#trend-metric-label").textContent = metric.label;
  $("#trend-current-value").textContent =
    `区间末值 ${formatMetricValue(selectedTrendMetric, metric.values.at(-1))}`;
  $("#trend-axis").innerHTML = range.axis
    .map((label) => `<span>${label}</span>`)
    .join("");
  const values = metric.values;
  renderTrendYAxis(selectedTrendMetric, values);
  hideTrendTooltip();
  const average =
    values.reduce((total, value) => total + value, 0) / values.length;
  const change = values.at(-1) - values[0];
  const peak = Math.max(...values);
  const peakIndex = values.indexOf(peak);
  const peakTime = new Date(
    range.start.getTime() +
      (range.end.getTime() - range.start.getTime()) *
        (peakIndex / (values.length - 1)),
  );
  const risk = getPeakRisk(selectedTrendMetric, peak);
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
  $("#summary-two-note").textContent = "起点至结束";
  $("#summary-three-label").textContent = "采样峰值";
  $("#summary-three-value").textContent = formatMetricValue(
    selectedTrendMetric,
    peak,
  );
  $("#summary-three-note").textContent = `${formatDateTime(peakTime)} 出现`;
  const riskValue = $("#summary-four-value");
  $("#summary-four-label").textContent = "峰值风险";
  riskValue.textContent = risk.label;
  riskValue.className = risk.className;
  $("#summary-four-note").textContent = risk.note;
  $$("[data-trend-metric]").forEach((button) =>
    button.classList.toggle(
      "is-selected",
      button.dataset.trendMetric === selectedTrendMetric,
    ),
  );
}

function renderRangeData() {
  const range = activeRange;
  $("#range-data-title-period").textContent = range.label;
  $("#range-data-description").textContent =
    `展示该区间按 ${range.cadence.replace("每 ", "")}归集的代表性采样点。`;
  $("#range-data-count").textContent = range.count;
  $("#range-data-retention").textContent = "保留周期：30 天";
  $("#range-data-list").innerHTML = range.details
    .map(
      (row) => `<tr>${row.map((value) => `<td>${value}</td>`).join("")}</tr>`,
    )
    .join("");
}

function openRangeData() {
  renderRangeData();
  $("#range-data-dialog").showModal();
}

function renderDeploymentHistory(key) {
  const rows = deploymentHistory[key];
  $("#deployment-list").innerHTML = rows
    .map(
      (release) => `
    <tr class="${release.current ? "current-release" : ""}">
      <td><div class="release-cell"><code>${release.version}</code>${release.current ? "<span>当前运行</span>" : ""}</div><small>${release.commit}</small></td>
      <td><span class="status-pill ${release.statusClass}"><i></i>${release.status}</span></td>
      <td>${release.time}</td>
      <td><span class="person"><span class="person-avatar">Z</span>zero</span></td>
      <td class="numeric">${release.duration}</td>
      <td>${release.current ? '<button class="quiet-button" type="button" disabled>当前版本</button>' : `<button class="rollback-button" type="button" data-version="${release.version}" data-commit="${release.rollbackCommit}">回滚至此版本</button>`}</td>
    </tr>`,
    )
    .join("");
  $$(".rollback-button").forEach((button) =>
    button.addEventListener("click", () =>
      openRollback(button.dataset.version, button.dataset.commit),
    ),
  );
}

function selectApplication(key) {
  const app = applications[key];
  if (!app) return;
  selectedApp = key;
  $$(".app-row").forEach((row) =>
    row.classList.toggle("is-selected", row.dataset.app === key),
  );
  const avatar = $("#detail-avatar");
  avatar.textContent = app.avatar;
  avatar.className = `app-avatar ${app.avatarClass}`;
  $("#detail-name").textContent = app.name;
  const url = $("#detail-url");
  url.href = `https://${app.url}`;
  url.childNodes[0].nodeValue = `${app.url} `;
  $("#detail-check").textContent = app.check;
  $("#detail-resource").textContent = app.resource;
  $("#detail-commit").textContent = app.commit;
  $("#detail-branch").textContent = ` ${app.branch}`;
  $("#deployments-title").textContent = `${app.name} 部署历史`;
  $("#deployments .eyebrow").textContent = app.name;
  renderDeploymentHistory(key);
}

function openRollback(version, commit) {
  const dialog = $("#rollback-dialog");
  $("#rollback-version").textContent = version;
  $("#rollback-commit").textContent = `${version} · ${commit}`;
  $("#rollback-app").textContent = applications[selectedApp].name;
  $("#rollback-current").textContent = applications[selectedApp].version;
  $("#confirm-input").value = "";
  $("#confirm-rollback").disabled = true;
  dialog.showModal();
  $("#confirm-input").focus();
}

function finishRollback() {
  const version = $("#rollback-version").textContent;
  const button = $("#confirm-rollback");
  button.disabled = true;
  button.textContent = "正在验证健康检查...";
  window.setTimeout(() => {
    $("#rollback-dialog").close();
    button.textContent = "确认回滚";
    showToast(
      `演示完成：${applications[selectedApp].name} 已回滚至 ${version}，健康检查通过。`,
    );
  }, 900);
}

$$(".app-select").forEach((button) =>
  button.addEventListener("click", () => selectApplication(button.dataset.app)),
);

$$(".nav-link[data-target]").forEach((button) => {
  button.addEventListener("click", () => {
    const target = button.dataset.target;
    const section = document.getElementById(target);
    if (!section) return;
    $$(".nav-link[data-target]").forEach((item) => {
      item.classList.toggle("is-active", item === button);
      item.removeAttribute("aria-current");
    });
    button.setAttribute("aria-current", "page");
    $("#page-label").textContent = pageLabels[target];
    section.scrollIntoView({
      behavior: window.matchMedia("(prefers-reduced-motion: reduce)").matches
        ? "auto"
        : "smooth",
      block: "start",
    });
  });
});

$$("[data-trend-metric]").forEach((button) =>
  button.addEventListener("click", () => {
    selectedTrendMetric = button.dataset.trendMetric;
    renderTrend();
  }),
);
$$(".rollback-button").forEach((button) =>
  button.addEventListener("click", () =>
    openRollback(button.dataset.version, button.dataset.commit),
  ),
);

$("#open-range-data").addEventListener("click", openRangeData);
$("#time-range-form").addEventListener("submit", (event) => {
  event.preventDefault();
  const startInput = $("#range-start");
  const endInput = $("#range-end");
  const feedback = $("#range-feedback");
  const start = new Date(startInput.value);
  const end = new Date(endInput.value);
  const now = new Date();
  const maxDuration = 30 * 24 * 60 * 60 * 1000;
  let error = "";

  if (
    !startInput.value ||
    !endInput.value ||
    Number.isNaN(start.getTime()) ||
    Number.isNaN(end.getTime())
  )
    error = "请填写精确到分钟的开始和结束时间。";
  else if (end <= start) error = "结束时间必须晚于开始时间。";
  else if (end > now) error = "结束时间不能晚于服务器当前时间。";
  else if (end.getTime() - start.getTime() > maxDuration)
    error = "当前演示最多可查询 30 天范围。";

  startInput.setCustomValidity(error);
  endInput.setCustomValidity(error);
  startInput.setAttribute("aria-invalid", String(Boolean(error)));
  endInput.setAttribute("aria-invalid", String(Boolean(error)));
  feedback.classList.toggle("is-error", Boolean(error));
  feedback.textContent = error;
  if (error) return;

  activeRange = buildRange(start, end);
  renderTrend();
  if ($("#range-data-dialog").open) renderRangeData();
  feedback.textContent = `已加载 ${activeRange.label} 的指标分析。`;
});
$("#confirm-input").addEventListener("input", (event) => {
  $("#confirm-rollback").disabled = event.target.value !== "ROLLBACK";
});
$("#confirm-rollback").addEventListener("click", finishRollback);
$("#open-deployments").addEventListener("click", () => {
  document
    .getElementById("deployments")
    .scrollIntoView({ behavior: "smooth", block: "start" });
});
$("#deploy-button").addEventListener("click", () =>
  showToast(
    `演示模式：连接部署执行器后可为 ${applications[selectedApp].name} 创建新部署。`,
  ),
);
$("#detail-menu").addEventListener("click", () =>
  showToast(`${applications[selectedApp].name} 的高级操作将在后端接入后启用。`),
);
$$("[data-demo-action]").forEach((button) =>
  button.addEventListener("click", () => showToast(button.dataset.demoAction)),
);
$("#theme-toggle").addEventListener("click", () =>
  document.body.classList.toggle("dark"),
);
$("#refresh-button").addEventListener("click", (event) => {
  event.currentTarget.classList.add("is-refreshing");
  window.setTimeout(() => {
    updateCurrentSnapshotClock();
    event.currentTarget.classList.remove("is-refreshing");
    showToast("状态已更新：采集正常。");
  }, 500);
});

const initialEnd = new Date();
initialEnd.setSeconds(0, 0);
const initialStart = new Date(initialEnd.getTime() - 24 * 60 * 60 * 1000);
$("#range-start").value = formatDateInput(initialStart);
$("#range-end").value = formatDateInput(initialEnd);
activeRange = buildRange(initialStart, initialEnd);
updateCurrentSnapshotClock(initialEnd);
window.setInterval(updateCurrentSnapshotClock, 60000);
bindTrendChartInteractions();
renderTrend();
