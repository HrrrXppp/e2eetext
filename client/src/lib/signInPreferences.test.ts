import { afterEach, describe, expect, it } from "vitest";
import {
  consumeSkipProfileOnCreate,
  setSkipProfileOnCreate,
} from "@/lib/signInPreferences";

afterEach(() => {
  localStorage.clear();
});

describe("signInPreferences", () => {
  it("stores and consumes skip-profile flag once", () => {
    setSkipProfileOnCreate(true);
    expect(localStorage.getItem("messenger_skip_profile_on_create")).toBe("1");
    expect(consumeSkipProfileOnCreate()).toBe(true);
    expect(consumeSkipProfileOnCreate()).toBe(false);
    expect(localStorage.getItem("messenger_skip_profile_on_create")).toBeNull();
  });

  it("clears skip-profile flag when disabled", () => {
    setSkipProfileOnCreate(true);
    setSkipProfileOnCreate(false);
    expect(localStorage.getItem("messenger_skip_profile_on_create")).toBeNull();
    expect(consumeSkipProfileOnCreate()).toBe(false);
  });
});
