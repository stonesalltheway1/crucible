import Link from "next/link";
import { Button } from "@/components/ui/button";
import { Skull } from "lucide-react";

export default function NotFound() {
  return (
    <div className="grid h-[60vh] place-items-center">
      <div className="flex flex-col items-center gap-3 text-center">
        <Skull className="h-8 w-8 text-muted-foreground" />
        <div>
          <h2 className="text-lg font-semibold">404 · no attestation for this path</h2>
          <p className="text-xs text-muted-foreground">The route doesn't resolve. Go home.</p>
        </div>
        <Button asChild>
          <Link href="/">Overview</Link>
        </Button>
      </div>
    </div>
  );
}
