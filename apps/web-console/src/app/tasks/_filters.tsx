"use client";

import { Search } from "lucide-react";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { useState } from "react";

const STATUS_CHIPS = [
  { value: "", label: "All" },
  { value: "plan_pending_approval", label: "Plan pending" },
  { value: "executing", label: "Executing" },
  { value: "verifying", label: "Verifying" },
  { value: "promotion_pending", label: "Promote pending" },
  { value: "completed", label: "Completed" },
  { value: "failed", label: "Failed" },
];

export function TaskFilters() {
  const [q, setQ] = useState("");
  const [status, setStatus] = useState("");

  return (
    <div className="mb-4 flex flex-wrap items-center gap-2">
      <div className="relative">
        <Search className="absolute left-2 top-1/2 h-3.5 w-3.5 -translate-y-1/2 text-muted-foreground" />
        <Input
          className="w-72 pl-7"
          placeholder="search description, repo, task id…"
          value={q}
          onChange={(e) => setQ(e.target.value)}
        />
      </div>
      <div className="flex items-center gap-1">
        {STATUS_CHIPS.map((c) => (
          <Button
            key={c.value || "all"}
            variant={status === c.value ? "ink" : "outline"}
            size="sm"
            onClick={() => setStatus(c.value)}
          >
            {c.label}
          </Button>
        ))}
      </div>
    </div>
  );
}
