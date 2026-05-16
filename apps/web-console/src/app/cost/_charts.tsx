"use client";

import { AreaChart, BarChart, LineChart } from "@tremor/react";

type Row = { key: string; label: string; cost_usd: number; tasks: number; cache_hit_rate: number };

const INK_PALETTE = ["slate", "stone", "neutral", "zinc"] as const;

export function CostCharts({ kind, data }: { kind: "day" | "bar"; data: Row[] }) {
  if (kind === "day") {
    return (
      <div className="space-y-6">
        <AreaChart
          className="h-56"
          data={data.map((d) => ({ date: d.label, "USD spend": Number(d.cost_usd.toFixed(2)) }))}
          index="date"
          categories={["USD spend"]}
          colors={["slate"]}
          valueFormatter={(n) => `$${n.toFixed(2)}`}
          showLegend={false}
          showGridLines
        />
        <div className="grid grid-cols-2 gap-4">
          <BarChart
            className="h-44"
            data={data.map((d) => ({ date: d.label, Tasks: d.tasks }))}
            index="date"
            categories={["Tasks"]}
            colors={["stone"]}
            showLegend={false}
          />
          <LineChart
            className="h-44"
            data={data.map((d) => ({ date: d.label, "Cache hit %": Math.round(d.cache_hit_rate * 100) }))}
            index="date"
            categories={["Cache hit %"]}
            colors={["zinc"]}
            valueFormatter={(n) => `${n}%`}
            showLegend={false}
          />
        </div>
      </div>
    );
  }
  return (
    <BarChart
      className="h-64"
      data={data.map((d) => ({ name: d.label, "USD spend": Number(d.cost_usd.toFixed(2)) }))}
      index="name"
      categories={["USD spend"]}
      colors={["slate"]}
      valueFormatter={(n) => `$${n.toFixed(2)}`}
      layout="vertical"
      showLegend={false}
    />
  );
}
