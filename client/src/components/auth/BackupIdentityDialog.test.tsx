import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { BackupIdentityDialog } from "@/components/auth/BackupIdentityDialog";

const exportStoredIdentityBackup = vi.fn();

vi.mock("@/lib/e2ee/storage", () => ({
  exportStoredIdentityBackup: (...args: unknown[]) => exportStoredIdentityBackup(...args),
}));

describe("BackupIdentityDialog", () => {
  let clickSpy: ReturnType<typeof vi.spyOn>;

  beforeEach(() => {
    exportStoredIdentityBackup.mockReset();
    URL.createObjectURL = vi.fn(() => "blob:mock-url");
    URL.revokeObjectURL = vi.fn();
    clickSpy = vi.spyOn(HTMLAnchorElement.prototype, "click").mockImplementation(() => {});
  });

  afterEach(() => {
    clickSpy.mockRestore();
    vi.restoreAllMocks();
  });

  it("requires a passphrase before submitting", async () => {
    render(<BackupIdentityDialog userId="user-1" onClose={vi.fn()} />);

    fireEvent.click(screen.getByRole("button", { name: "Download backup" }));

    expect(await screen.findByRole("alert")).toHaveTextContent(
      "Enter a passphrase to protect the backup file.",
    );
    expect(exportStoredIdentityBackup).not.toHaveBeenCalled();
  });

  it("requires the confirmation passphrase to match", async () => {
    render(<BackupIdentityDialog userId="user-1" onClose={vi.fn()} />);

    fireEvent.change(screen.getByLabelText("Backup passphrase"), { target: { value: "abc" } });
    fireEvent.change(screen.getByLabelText("Confirm passphrase"), { target: { value: "xyz" } });
    fireEvent.click(screen.getByRole("button", { name: "Download backup" }));

    expect(await screen.findByRole("alert")).toHaveTextContent("Passphrases do not match.");
    expect(exportStoredIdentityBackup).not.toHaveBeenCalled();
  });

  it("downloads a backup file once the passphrase is confirmed", async () => {
    exportStoredIdentityBackup.mockResolvedValue(JSON.stringify({ alg: "scrypt-aes256gcm" }));

    render(<BackupIdentityDialog userId="user-1" onClose={vi.fn()} />);

    fireEvent.change(screen.getByLabelText("Backup passphrase"), { target: { value: "abc" } });
    fireEvent.change(screen.getByLabelText("Confirm passphrase"), { target: { value: "abc" } });
    fireEvent.click(screen.getByRole("button", { name: "Download backup" }));

    await waitFor(() => {
      expect(exportStoredIdentityBackup).toHaveBeenCalledWith("user-1", "abc");
    });
    expect(clickSpy).toHaveBeenCalled();
    expect(await screen.findByText(/Backup downloaded/)).toBeInTheDocument();
  });

  it("accepts a short passphrase with no minimum length enforced", async () => {
    exportStoredIdentityBackup.mockResolvedValue(JSON.stringify({ alg: "scrypt-aes256gcm" }));

    render(<BackupIdentityDialog userId="user-1" onClose={vi.fn()} />);

    fireEvent.change(screen.getByLabelText("Backup passphrase"), { target: { value: "x" } });
    fireEvent.change(screen.getByLabelText("Confirm passphrase"), { target: { value: "x" } });
    fireEvent.click(screen.getByRole("button", { name: "Download backup" }));

    await waitFor(() => {
      expect(exportStoredIdentityBackup).toHaveBeenCalledWith("user-1", "x");
    });
  });

  it("closes on escape", () => {
    const onClose = vi.fn();
    render(<BackupIdentityDialog userId="user-1" onClose={onClose} />);

    fireEvent.keyDown(document, { key: "Escape" });
    expect(onClose).toHaveBeenCalled();
  });
});
