import { describe, expect, it } from "vitest";
import { appVersion } from "@/lib/version";

describe("appVersion", () => {
  it("is injected at build time", () => {
    expect(appVersion).toMatch(/^\d+\.\d+\.\d+/);
  });
});
