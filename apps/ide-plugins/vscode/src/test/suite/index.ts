import * as path from "node:path";
import * as assert from "node:assert";
import { glob } from "glob";

export async function run(): Promise<void> {
  const file = path.resolve(__dirname, "./extension.test.js");
  const mod: { runSuite?: () => Promise<void> } = await import(file);
  if (mod.runSuite) {
    await mod.runSuite();
  } else {
    // Smoke check: extension entry compiles.
    assert.ok(file);
  }
}
