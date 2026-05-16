"use client";

import { Copy, Check } from "lucide-react";
import { useState } from "react";
import { shortHash } from "@/lib/utils";

export function HashPill({
  value,
  href,
  className,
}: {
  value: string;
  href?: string;
  className?: string;
}) {
  const [copied, setCopied] = useState(false);
  const onCopy = async () => {
    try {
      await navigator.clipboard.writeText(value);
      setCopied(true);
      setTimeout(() => setCopied(false), 1100);
    } catch {
      // Clipboard API unavailable in some sandboxes — fail silently.
    }
  };
  const label = shortHash(value);
  const content = (
    <span className={`inline-flex items-center gap-1 font-mono text-xs ${className ?? ""}`}>
      <span className="border border-ink-200 bg-ink-50 px-1 py-0.5 dark:border-ink-800 dark:bg-ink-900">
        {label}
      </span>
      <button
        type="button"
        onClick={onCopy}
        className="text-muted-foreground hover:text-foreground"
        aria-label="copy hash"
      >
        {copied ? <Check className="h-3 w-3" /> : <Copy className="h-3 w-3" />}
      </button>
    </span>
  );
  if (href) {
    return (
      <a href={href} className="underline-offset-2 hover:underline">
        {content}
      </a>
    );
  }
  return content;
}
