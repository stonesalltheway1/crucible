"use client";

import { useState } from "react";
import { Input } from "@/components/ui/input";
import { Search } from "lucide-react";

export function MemoryFilters() {
  const [q, setQ] = useState("");
  return (
    <div className="flex items-center gap-2">
      <div className="relative w-72">
        <Search className="absolute left-2 top-1/2 h-3.5 w-3.5 -translate-y-1/2 text-muted-foreground" />
        <Input
          className="pl-7"
          placeholder="search by rule, category, repo…"
          value={q}
          onChange={(e) => setQ(e.target.value)}
        />
      </div>
    </div>
  );
}
