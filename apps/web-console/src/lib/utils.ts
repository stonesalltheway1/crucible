import { clsx, type ClassValue } from "clsx";
import { twMerge } from "tailwind-merge";

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs));
}

export function shortHash(h: string, head = 8, tail = 6): string {
  if (!h) return "";
  if (h.length <= head + tail + 1) return h;
  return `${h.slice(0, head)}…${h.slice(-tail)}`;
}

export function formatUsd(n: number, fractionDigits = 2): string {
  return `$${n.toLocaleString("en-US", {
    minimumFractionDigits: fractionDigits,
    maximumFractionDigits: fractionDigits,
  })}`;
}

export function formatDuration(seconds: number): string {
  if (!Number.isFinite(seconds) || seconds < 0) return "—";
  if (seconds < 1) return `${Math.round(seconds * 1000)}ms`;
  if (seconds < 60) return `${seconds.toFixed(1)}s`;
  const m = Math.floor(seconds / 60);
  const s = Math.round(seconds - m * 60);
  if (m < 60) return `${m}m ${s}s`;
  const h = Math.floor(m / 60);
  return `${h}h ${m - h * 60}m`;
}

export function formatRelative(iso: string, now: Date = new Date()): string {
  const t = new Date(iso).getTime();
  if (!Number.isFinite(t)) return iso;
  const diff = (now.getTime() - t) / 1000;
  const abs = Math.abs(diff);
  const sign = diff < 0 ? "in " : "";
  const suffix = diff < 0 ? "" : " ago";
  if (abs < 5) return "just now";
  if (abs < 60) return `${sign}${Math.round(abs)}s${suffix}`;
  if (abs < 3600) return `${sign}${Math.round(abs / 60)}m${suffix}`;
  if (abs < 86_400) return `${sign}${Math.round(abs / 3600)}h${suffix}`;
  return `${sign}${Math.round(abs / 86_400)}d${suffix}`;
}

export function clampPercent(n: number): number {
  if (!Number.isFinite(n)) return 0;
  return Math.max(0, Math.min(100, n));
}
