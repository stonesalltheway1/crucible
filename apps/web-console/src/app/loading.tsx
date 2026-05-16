export default function Loading() {
  return (
    <div className="space-y-4 animate-pulse">
      <div className="h-7 w-1/3 bg-ink-200 dark:bg-ink-800" />
      <div className="h-4 w-1/2 bg-ink-200 dark:bg-ink-800" />
      <div className="grid grid-cols-4 gap-3">
        {Array.from({ length: 4 }).map((_, i) => (
          <div key={i} className="h-20 border border-ink-200 bg-ink-50 dark:border-ink-800 dark:bg-ink-900" />
        ))}
      </div>
    </div>
  );
}
