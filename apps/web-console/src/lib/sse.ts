"use client";

import { createParser } from "eventsource-parser";
import { useEffect, useRef, useState } from "react";

// SSE hook for streaming plan/verifier/promotion progress.
//
// We deliberately use fetch + eventsource-parser over the EventSource browser
// API because we need to send the tenant bearer token in a header. EventSource
// only supports cookies.
export function useSse<T>(
  url: string | null,
  opts?: { onEvent?: (evt: { event: string; data: T }) => void; token?: string },
): { events: { event: string; data: T }[]; connected: boolean; error?: Error } {
  const [events, setEvents] = useState<{ event: string; data: T }[]>([]);
  const [connected, setConnected] = useState(false);
  const [error, setError] = useState<Error | undefined>(undefined);
  const aborted = useRef(false);

  useEffect(() => {
    if (!url) return;
    aborted.current = false;
    const ctrl = new AbortController();
    setConnected(false);
    setError(undefined);

    (async () => {
      try {
        const headers: Record<string, string> = { Accept: "text/event-stream" };
        if (opts?.token) headers.Authorization = `Bearer ${opts.token}`;
        const res = await fetch(url, { headers, signal: ctrl.signal });
        if (!res.ok || !res.body) throw new Error(`SSE ${url}: ${res.status}`);
        setConnected(true);
        const parser = createParser({
          onEvent: (e) => {
            try {
              const data = JSON.parse(e.data) as T;
              const evt = { event: e.event ?? "message", data };
              setEvents((prev) => [...prev, evt]);
              opts?.onEvent?.(evt);
            } catch {
              // Skip malformed frames; the parser already isolated us at the boundary.
            }
          },
        });
        const reader = res.body.getReader();
        const decoder = new TextDecoder();
        while (!aborted.current) {
          const { done, value } = await reader.read();
          if (done) break;
          parser.feed(decoder.decode(value, { stream: true }));
        }
        setConnected(false);
      } catch (e) {
        if (!aborted.current) {
          setError(e as Error);
          setConnected(false);
        }
      }
    })();

    return () => {
      aborted.current = true;
      ctrl.abort();
    };
  }, [url, opts?.token]);

  return { events, connected, error };
}
