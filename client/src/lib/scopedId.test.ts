import { describe, expect, it } from "vitest";
import { nodePrefixFromScopedId, scopeResourceId, scopeResourceIds } from "@/lib/scopedId";

const nodeId = "99999999-9999-9999-9999-999999999999";
const localUserId = "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa";
const scopedUserId = `${nodeId}/${localUserId}`;

describe("scopeResourceIds", () => {
  it("prefixes bare UUIDs using node id from a scoped id in the list", () => {
    expect(
      scopeResourceIds([
        scopedUserId,
        "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb",
      ]),
    ).toEqual([
      scopedUserId,
      `${nodeId}/bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb`,
    ]);
  });

  it("leaves already scoped ids unchanged", () => {
    expect(scopeResourceIds([scopedUserId])).toEqual([scopedUserId]);
  });

  it("returns ids unchanged when no node prefix is available", () => {
    const bare = [localUserId, "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"];
    expect(scopeResourceIds(bare)).toEqual(bare);
  });
});

describe("scopeResourceId", () => {
  it("scopes a local id", () => {
    expect(scopeResourceId(localUserId, nodeId)).toBe(scopedUserId);
  });
});

describe("nodePrefixFromScopedId", () => {
  it("extracts the node prefix", () => {
    expect(nodePrefixFromScopedId(scopedUserId)).toBe(nodeId);
  });

  it("returns null for bare ids", () => {
    expect(nodePrefixFromScopedId(localUserId)).toBeNull();
  });
});
