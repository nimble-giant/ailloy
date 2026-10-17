import { describe, expect, it } from "vitest";
import {
  behaviorFiles,
  changedFiles,
  commandSurfaceFiles,
  hasDocsUpdate,
  hasFeaturesUpdate,
  isBehaviorCode,
  isCommandSurface,
} from "../src/predicates.js";

describe("isBehaviorCode", () => {
  it("Given non-test Go under internal/commands, When classified, Then it is behavior code", () => {
    expect(isBehaviorCode("internal/commands/cast.go")).toBe(true);
  });

  it("Given non-test Go under pkg, When classified, Then it is behavior code", () => {
    expect(isBehaviorCode("pkg/mold/mold.go")).toBe(true);
  });

  it("Given a Go test file, When classified, Then it is not behavior code", () => {
    expect(isBehaviorCode("internal/commands/cast_test.go")).toBe(false);
  });

  it("Given a doc or cmd path, When classified, Then it is not behavior code", () => {
    expect(isBehaviorCode("cmd/ailloy/main.go")).toBe(false);
    expect(isBehaviorCode("docs/cast.md")).toBe(false);
  });
});

describe("behaviorFiles + hasFeaturesUpdate", () => {
  it("Given a mixed change set, When filtered, Then only behavior files remain", () => {
    const files = ["internal/commands/cast.go", "internal/commands/cast_test.go", "README.md", "pkg/foundry/x.go"];
    expect(behaviorFiles(files)).toEqual(["internal/commands/cast.go", "pkg/foundry/x.go"]);
  });

  it("Given features.md is in the set, When checked, Then the contract is satisfied", () => {
    expect(hasFeaturesUpdate(["internal/commands/cast.go", "features.md"])).toBe(true);
    expect(hasFeaturesUpdate(["internal/commands/cast.go"])).toBe(false);
  });
});

describe("command surface + docs", () => {
  it("Given command/cmd Go files, When filtered, Then they are the command surface", () => {
    const files = ["internal/commands/forge.go", "cmd/ailloy/main.go", "pkg/mold/mold.go"];
    expect(commandSurfaceFiles(files)).toEqual(["internal/commands/forge.go", "cmd/ailloy/main.go"]);
  });

  it("Given any docs/README/AGENTS path, When checked, Then a docs update is present", () => {
    expect(hasDocsUpdate(["docs/forge.md"])).toBe(true);
    expect(hasDocsUpdate(["README.md"])).toBe(true);
    expect(hasDocsUpdate(["AGENTS.md"])).toBe(true);
    expect(hasDocsUpdate(["internal/commands/forge.go"])).toBe(false);
  });

  it("Given a docs-only file, When classified as command surface, Then it is excluded", () => {
    expect(isCommandSurface("docs/forge.md")).toBe(false);
  });
});

describe("changedFiles", () => {
  it("Given no change context, When read, Then it returns null (whole-target audit)", () => {
    expect(changedFiles(null)).toBeNull();
    expect(changedFiles(undefined)).toBeNull();
  });

  it("Given a change context, When read, Then it returns a copy of the changed files", () => {
    const change = { scanMode: "changed", changedFiles: ["a.go"], worktree: false } as const;
    expect(changedFiles(change)).toEqual(["a.go"]);
  });
});
