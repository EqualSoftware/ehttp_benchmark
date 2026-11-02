// Shared helpers for the k6 scripts. Every size comes from an environment variable.

export function num(name, fallback) {
  const raw = __ENV[name];
  if (raw === undefined || raw === '') return fallback;
  const value = Number(raw);
  if (Number.isNaN(value)) throw new Error(`${name} must be a number, got "${raw}"`);
  return value;
}

export function str(name, fallback) {
  const raw = __ENV[name];
  return raw === undefined || raw === '' ? fallback : raw;
}

// Parses "30s", "5m", "1h" into seconds. A bare number is seconds.
export function seconds(text) {
  const m = /^(\d+(?:\.\d+)?)(ms|s|m|h)?$/.exec(String(text));
  if (!m) throw new Error(`bad duration "${text}"`);
  const unit = { ms: 0.001, s: 1, m: 60, h: 3600 }[m[2] || 's'];
  return Number(m[1]) * unit;
}
