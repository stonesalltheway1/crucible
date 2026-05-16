import { PageHeader, MetricStat } from "@/components/page-header";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Tabs, TabsList, TabsTrigger, TabsContent } from "@/components/ui/tabs";
import { CostCharts } from "./_charts";
import { formatUsd } from "@/lib/utils";

function genSeries(groupBy: "day" | "repo" | "dev") {
  if (groupBy === "day") {
    const days: { key: string; label: string; cost_usd: number; tasks: number; cache_hit_rate: number }[] = [];
    for (let i = 13; i >= 0; i--) {
      const d = new Date(Date.now() - i * 86_400_000);
      days.push({
        key: d.toISOString().slice(0, 10),
        label: d.toLocaleDateString("en-US", { month: "short", day: "numeric" }),
        cost_usd: 8 + Math.sin(i / 2) * 4 + Math.random() * 3,
        tasks: 6 + Math.round(Math.sin(i / 2) * 3 + Math.random() * 4),
        cache_hit_rate: 0.7 + Math.cos(i / 3) * 0.08,
      });
    }
    return days;
  }
  if (groupBy === "repo") {
    return [
      { key: "payments", label: "acme/payments", cost_usd: 41.2, tasks: 32, cache_hit_rate: 0.78 },
      { key: "billing", label: "acme/billing", cost_usd: 22.7, tasks: 18, cache_hit_rate: 0.74 },
      { key: "platform", label: "acme/platform-api", cost_usd: 14.4, tasks: 11, cache_hit_rate: 0.83 },
      { key: "dashboard", label: "acme/dashboard", cost_usd: 9.8, tasks: 9, cache_hit_rate: 0.69 },
    ];
  }
  return [
    { key: "sarah", label: "sarah@acme.dev", cost_usd: 28.4, tasks: 24, cache_hit_rate: 0.81 },
    { key: "marcus", label: "marcus@acme.dev", cost_usd: 17.2, tasks: 14, cache_hit_rate: 0.78 },
    { key: "priya", label: "priya@acme.dev", cost_usd: 11.9, tasks: 9, cache_hit_rate: 0.73 },
    { key: "yusef", label: "yusef@acme.dev", cost_usd: 8.2, tasks: 7, cache_hit_rate: 0.7 },
  ];
}

export default async function CostPage() {
  const byDay = genSeries("day");
  const byRepo = genSeries("repo");
  const byDev = genSeries("dev");
  const total14d = byDay.reduce((a, b) => a + b.cost_usd, 0);
  const tasks14d = byDay.reduce((a, b) => a + b.tasks, 0);
  const medianTask = tasks14d > 0 ? total14d / tasks14d : 0;
  const avgCache = byDay.reduce((a, b) => a + b.cache_hit_rate, 0) / byDay.length;

  return (
    <>
      <PageHeader
        title="Cost"
        description="Per-task, per-repo, per-developer rollups. Crucible's median target is $1.69/task — below that and the unit economics work."
      />

      <div className="mb-4 grid grid-cols-4 gap-3">
        <MetricStat label="Spend 14d" value={formatUsd(total14d)} hint={`${tasks14d} tasks`} />
        <MetricStat label="Median / task" value={formatUsd(medianTask)} hint="target ≤ $1.69" tone={medianTask <= 1.69 ? "ok" : "warn"} />
        <MetricStat label="Cache hit avg" value={`${Math.round(avgCache * 100)}%`} hint="target ≥ 70%" tone={avgCache >= 0.7 ? "ok" : "warn"} />
        <MetricStat label="Tier 4 % of spend" value="6%" hint="hermetic-rebuild verification" tone="ok" />
      </div>

      <Tabs defaultValue="day">
        <TabsList>
          <TabsTrigger value="day">By day</TabsTrigger>
          <TabsTrigger value="repo">By repository</TabsTrigger>
          <TabsTrigger value="dev">By developer</TabsTrigger>
        </TabsList>
        <TabsContent value="day" className="pt-4">
          <Card>
            <CardHeader>
              <CardTitle>14-day trend</CardTitle>
              <CardDescription>USD spend + task count + cache hit rate, per day.</CardDescription>
            </CardHeader>
            <CardContent>
              <CostCharts kind="day" data={byDay} />
            </CardContent>
          </Card>
        </TabsContent>
        <TabsContent value="repo" className="pt-4">
          <Card>
            <CardHeader>
              <CardTitle>By repository</CardTitle>
            </CardHeader>
            <CardContent>
              <CostCharts kind="bar" data={byRepo} />
            </CardContent>
          </Card>
        </TabsContent>
        <TabsContent value="dev" className="pt-4">
          <Card>
            <CardHeader>
              <CardTitle>By developer</CardTitle>
            </CardHeader>
            <CardContent>
              <CostCharts kind="bar" data={byDev} />
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>
    </>
  );
}
