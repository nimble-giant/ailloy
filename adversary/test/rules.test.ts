import { mkdtemp } from "node:fs/promises";
import { tmpdir } from "node:os";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";
import { afterAll, beforeAll, describe, expect, it } from "vitest";
import type { ReviewResult } from "@adversarylabs/sdk";
import adversary from "../src/index.js";

const here = dirname(fileURLToPath(import.meta.url));
const fixtures = join(here, "fixtures");

// An empty directory isolates the diff-aware rules: they inspect the synthetic
// change list only, and the content-scanning rules find nothing on disk.
let emptyRepo: string;
beforeAll(async () => {
  emptyRepo = await mkdtemp(join(tmpdir(), "ailloy-guardian-"));
});
afterAll(() => {
  /* mkdtemp dirs are tiny and OS-cleaned; nothing to remove for the test. */
});

function runWithChange(changedFiles: string[], path: string): Promise<ReviewResult> {
  return adversary.run({
    input: { source: { path }, change: { scan_mode: "changed", changed_files: changedFiles } },
  });
}

const ruleIds = (result: ReviewResult): string[] => result.findings.map((f) => f.ruleId ?? "");

describe("features-contract rule", () => {
  it("Given behavior code changed and no features.md update, When reviewed, Then it flags the contract", async () => {
    const result = await runWithChange(["internal/commands/cast.go"], emptyRepo);
    expect(ruleIds(result)).toContain("ailloy.features-contract");
  });

  it("Given behavior code changed WITH a features.md update, When reviewed, Then it stays silent", async () => {
    const result = await runWithChange(["internal/commands/cast.go", "features.md"], emptyRepo);
    expect(ruleIds(result)).not.toContain("ailloy.features-contract");
  });

  it("Given only test files changed, When reviewed, Then it stays silent", async () => {
    const result = await runWithChange(["internal/commands/cast_test.go"], emptyRepo);
    expect(ruleIds(result)).not.toContain("ailloy.features-contract");
  });
});

describe("docs-drift rule", () => {
  it("Given command surface changed and no docs update, When reviewed, Then it flags drift", async () => {
    const result = await runWithChange(["cmd/ailloy/main.go"], emptyRepo);
    expect(ruleIds(result)).toContain("ailloy.docs-drift");
  });

  it("Given command surface changed WITH a docs update, When reviewed, Then it stays silent", async () => {
    const result = await runWithChange(["cmd/ailloy/main.go", "docs/cli.md"], emptyRepo);
    expect(ruleIds(result)).not.toContain("ailloy.docs-drift");
  });
});

describe("mold-correctness rule", () => {
  it("Given a mold missing a field and an ingot referencing a missing file, When scanned, Then both are flagged", async () => {
    const result = await adversary.run({
      input: { source: { path: join(fixtures, "mold-bad") }, change: null },
    });
    const moldFindings = result.findings.filter((f) => f.ruleId === "ailloy.mold-correctness");
    expect(moldFindings.length).toBeGreaterThanOrEqual(2);
    const titles = moldFindings.map((f) => f.title).join("\n");
    expect(titles).toContain("version");
    expect(titles).toContain("does not exist");
  });
});
