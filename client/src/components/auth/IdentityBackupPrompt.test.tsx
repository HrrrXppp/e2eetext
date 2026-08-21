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
});
