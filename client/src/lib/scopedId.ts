export function nodePrefixFromScopedId(scopedOrLocalId: string): string | null {
  const slash = scopedOrLocalId.indexOf("/");
  if (slash <= 0) {
    return null;
  }
  return scopedOrLocalId.slice(0, slash);
}

export function scopeResourceId(localOrScopedId: string, nodePrefix: string): string {
  if (localOrScopedId.includes("/")) {
    return localOrScopedId;
  }
  return `${nodePrefix}/${localOrScopedId}`;
}

export function scopeResourceIds(ids: string[]): string[] {
  const nodePrefix = ids.map(nodePrefixFromScopedId).find((prefix): prefix is string => prefix !== null);
  if (!nodePrefix) {
    return ids;
  }
  return ids.map((id) => scopeResourceId(id, nodePrefix));
}
