"use client";
import { useState } from "react";

export function CertificateRow({ pem }: { pem: string }) {
  const [open, setOpen] = useState(false);
  return (
    <div className="border border-ink-200 dark:border-ink-800">
      <button
        type="button"
        onClick={() => setOpen((v) => !v)}
        className="flex w-full items-center justify-between bg-ink-50 px-3 py-2 text-left text-xs hover:bg-ink-100 dark:bg-ink-900 dark:hover:bg-ink-800"
      >
        <span>{open ? "▾" : "▸"} certificate</span>
        <span className="font-mono text-[10px] text-muted-foreground">{pem.length} bytes</span>
      </button>
      {open && (
        <pre className="hash-block max-h-64 overflow-auto whitespace-pre-wrap border-t border-ink-200 text-[10px] dark:border-ink-800">
{pem}
        </pre>
      )}
    </div>
  );
}
