export const formatNumber = (value: number, digits = 2) => {
  if (!Number.isFinite(value)) {
    return "--";
  }
  return value.toLocaleString("en-US", {
    minimumFractionDigits: digits,
    maximumFractionDigits: digits
  });
};

export const formatCompact = (value: number) => {
  if (!Number.isFinite(value)) {
    return "--";
  }
  return new Intl.NumberFormat("en-US", {
    notation: "compact",
    maximumFractionDigits: 2
  }).format(value);
};

export const formatPercent = (value: number, digits = 2) => {
  if (!Number.isFinite(value)) {
    return "--";
  }
  return `${(value * 100).toFixed(digits)}%`;
};

export const formatTime = (ts: number) => {
  if (!Number.isFinite(ts)) {
    return "--";
  }
  const date = new Date(ts);
  return date.toLocaleTimeString("zh-CN", {
    hour: "2-digit",
    minute: "2-digit"
  });
};
