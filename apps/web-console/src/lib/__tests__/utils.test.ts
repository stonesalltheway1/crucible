import { describe, expect, it } from "vitest";
import { clampPercent, formatDuration, formatRelative, formatUsd, shortHash } from "@/lib/utils";

describe("utils", () => {
  describe("shortHash", () => {
    it("returns the input when it's short enough", () => {
      expect(shortHash("abc")).toBe("abc");
    });
    it("truncates with an ellipsis in the middle", () => {
      const h = "abcdef1234567890";
      expect(shortHash(h, 4, 4)).toBe("abcd…7890");
    });
  });

  describe("formatUsd", () => {
    it("formats with 2 fraction digits by default", () => {
      expect(formatUsd(1.234)).toBe("$1.23");
    });
  });

  describe("formatDuration", () => {
    it("formats sub-second values in milliseconds", () => {
      expect(formatDuration(0.42)).toBe("420ms");
    });
    it("formats minute-scale values", () => {
      expect(formatDuration(125)).toBe("2m 5s");
    });
    it("formats hour-scale values", () => {
      expect(formatDuration(3600 * 2 + 60 * 30)).toBe("2h 30m");
    });
  });

  describe("formatRelative", () => {
    it("produces 'just now' for very recent times", () => {
      const now = new Date("2026-05-15T00:00:00Z");
      const iso = new Date(now.getTime() - 1000).toISOString();
      expect(formatRelative(iso, now)).toBe("just now");
    });
    it("formats minute deltas", () => {
      const now = new Date("2026-05-15T00:00:00Z");
      const iso = new Date(now.getTime() - 120_000).toISOString();
      expect(formatRelative(iso, now)).toBe("2m ago");
    });
  });

  describe("clampPercent", () => {
    it("clamps to [0,100]", () => {
      expect(clampPercent(150)).toBe(100);
      expect(clampPercent(-10)).toBe(0);
      expect(clampPercent(33.3)).toBe(33.3);
    });
  });
});
