import { readFileSync } from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";
import { fireEvent, render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { IdentityBackupPrompt } from "@/components/auth/IdentityBackupPrompt";

const exportStoredIdentityBackup = vi.fn();
const importIdentityBackup = vi.fn();
const fetchIdentityPublicKey = vi.fn();
const saveStoredIdentity = vi.fn();
const uploadIdentityKey = vi.fn();

vi.mock("@/lib/e2ee/storage", () => ({
  exportStoredIdentityBackup: (...args: unknown[]) => exportStoredIdentityBackup(...args),
  fetchIdentityPublicKey: (...args: unknown[]) => fetchIdentityPublicKey(...args),
  loadStoredIdentity: vi.fn().mockResolvedValue(null),
  saveStoredIdentity: (...args: unknown[]) => saveStoredIdentity(...args),
  uploadIdentityKey: (...args: unknown[]) => uploadIdentityKey(...args),
}));

vi.mock("@/lib/e2ee/crypto", () => ({
  importIdentityBackup: (...args: unknown[]) => importIdentityBackup(...args),
}));

describe("IdentityBackupPrompt", () => {
  beforeEach(() => {
    exportStoredIdentityBackup.mockReset();
    importIdentityBackup.mockReset();
    fetchIdentityPublicKey.mockReset();
    saveStoredIdentity.mockReset();
    uploadIdentityKey.mockReset();
  });

  it("explains the new key and offers back up, restore, and skip", () => {
    render(<IdentityBackupPrompt userId="user-1" onDismiss={vi.fn()} />);

    expect(screen.getByText("Save a backup of your private key")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Back up now" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Restore from a backup instead" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Skip for now" })).toBeInTheDocument();
  });

  it("calls onDismiss when skipped", () => {
    const onDismiss = vi.fn();
    render(<IdentityBackupPrompt userId="user-1" onDismiss={onDismiss} />);

    fireEvent.click(screen.getByRole("button", { name: "Skip for now" }));

    expect(onDismiss).toHaveBeenCalled();
  });

  it("switches to the backup dialog on Back up now", () => {
    render(<IdentityBackupPrompt userId="user-1" onDismiss={vi.fn()} />);

    fireEvent.click(screen.getByRole("button", { name: "Back up now" }));

    expect(screen.getByText("Back up your private key")).toBeInTheDocument();
  });

  it("switches to the restore dialog on Restore from a backup instead", () => {
    render(<IdentityBackupPrompt userId="user-1" onDismiss={vi.fn()} />);

    fireEvent.click(screen.getByRole("button", { name: "Restore from a backup instead" }));

    expect(screen.getByText("Restore your private key")).toBeInTheDocument();
  });

  it("spaces Restore and Skip instead of sitting them flush together", () => {
    const cssPath = path.resolve(
      path.dirname(fileURLToPath(import.meta.url)),
      "../../styles/index.css",
    );
    const css = readFileSync(cssPath, "utf8");
    const match = css.match(/\.identity-backup-prompt__secondary\s*\{([^}]*)\}/);
    expect(match).not.toBeNull();
    expect(match![1]).toMatch(/display:\s*flex/);
    expect(match![1]).toMatch(/gap:\s*0\.75rem\s+1\.25rem/);

    render(<IdentityBackupPrompt userId="user-1" onDismiss={vi.fn()} />);
    const restore = screen.getByRole("button", { name: "Restore from a backup instead" });
    const skip = screen.getByRole("button", { name: "Skip for now" });
    expect(restore.parentElement).toBe(skip.parentElement);
    expect(restore.parentElement).toHaveClass("identity-backup-prompt__secondary");
  });
});
