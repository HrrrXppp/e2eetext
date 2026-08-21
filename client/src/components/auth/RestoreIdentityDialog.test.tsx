import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { RestoreIdentityDialog } from "@/components/auth/RestoreIdentityDialog";

const importIdentityBackup = vi.fn();
const fetchIdentityPublicKey = vi.fn();
const loadStoredIdentity = vi.fn();
const saveStoredIdentity = vi.fn();
const uploadIdentityKey = vi.fn();

vi.mock("@/lib/e2ee/crypto", () => ({
  importIdentityBackup: (...args: unknown[]) => importIdentityBackup(...args),
}));

vi.mock("@/lib/e2ee/storage", () => ({
  fetchIdentityPublicKey: (...args: unknown[]) => fetchIdentityPublicKey(...args),
  loadStoredIdentity: (...args: unknown[]) => loadStoredIdentity(...args),
  saveStoredIdentity: (...args: unknown[]) => saveStoredIdentity(...args),
  uploadIdentityKey: (...args: unknown[]) => uploadIdentityKey(...args),
}));

function selectFile(contents = "{}") {
  const file = new File([contents], "backup.json", { type: "application/json" });
  fireEvent.change(screen.getByLabelText("Backup file"), { target: { files: [file] } });
}

describe("RestoreIdentityDialog", () => {
  beforeEach(() => {
    importIdentityBackup.mockReset();
    fetchIdentityPublicKey.mockReset();
    loadStoredIdentity.mockReset();
    saveStoredIdentity.mockReset();
    uploadIdentityKey.mockReset();
    loadStoredIdentity.mockResolvedValue(null);
    saveStoredIdentity.mockResolvedValue(undefined);
    uploadIdentityKey.mockResolvedValue(undefined);
  });

  it("requires a file before submitting", async () => {
    render(<RestoreIdentityDialog userId="user-1" onClose={vi.fn()} />);

    fireEvent.change(screen.getByLabelText("Backup passphrase"), { target: { value: "abc" } });
    fireEvent.click(screen.getByRole("button", { name: "Restore" }));

    expect(await screen.findByRole("alert")).toHaveTextContent("Choose a backup file.");
    expect(importIdentityBackup).not.toHaveBeenCalled();
  });

  it("requires a passphrase before submitting", async () => {
    render(<RestoreIdentityDialog userId="user-1" onClose={vi.fn()} />);

    selectFile();
    fireEvent.click(screen.getByRole("button", { name: "Restore" }));

    expect(await screen.findByRole("alert")).toHaveTextContent("Enter the backup passphrase.");
    expect(importIdentityBackup).not.toHaveBeenCalled();
  });

  it("restores directly when there is no existing server key to collide with", async () => {
    importIdentityBackup.mockResolvedValue({ publicKey: "pk-1", secretKey: "sk-1" });
    fetchIdentityPublicKey.mockRejectedValue(new Error("identity key not found"));

    render(<RestoreIdentityDialog userId="user-1" onClose={vi.fn()} />);

    selectFile();
    fireEvent.change(screen.getByLabelText("Backup passphrase"), { target: { value: "abc" } });
    fireEvent.click(screen.getByRole("button", { name: "Restore" }));

    await waitFor(() => {
      expect(saveStoredIdentity).toHaveBeenCalledWith("user-1", { publicKey: "pk-1", secretKey: "sk-1" });
    });
    expect(uploadIdentityKey).toHaveBeenCalledWith("user-1", { publicKey: "pk-1", secretKey: "sk-1" });
    expect(await screen.findByText(/Backup restored/)).toBeInTheDocument();
  });

  it("restores directly when the imported key matches the current server key", async () => {
    importIdentityBackup.mockResolvedValue({ publicKey: "pk-1", secretKey: "sk-1" });
    fetchIdentityPublicKey.mockResolvedValue({ v: 1, alg: "hybrid-kem-mlkem768-x25519", publicKey: "pk-1" });

    render(<RestoreIdentityDialog userId="user-1" onClose={vi.fn()} />);

    selectFile();
    fireEvent.change(screen.getByLabelText("Backup passphrase"), { target: { value: "abc" } });
    fireEvent.click(screen.getByRole("button", { name: "Restore" }));

    await waitFor(() => {
      expect(saveStoredIdentity).toHaveBeenCalled();
    });
    expect(screen.queryByText(/already has a different key/)).not.toBeInTheDocument();
  });

  it("requires explicit confirmation before overwriting a different local key even with no server key on file", async () => {
    importIdentityBackup.mockResolvedValue({ publicKey: "pk-imported", secretKey: "sk-imported" });
    fetchIdentityPublicKey.mockRejectedValue(new Error("identity key not found"));
    loadStoredIdentity.mockResolvedValue({ publicKey: "pk-local", secretKey: "sk-local" });

    render(<RestoreIdentityDialog userId="user-1" onClose={vi.fn()} />);

    selectFile();
    fireEvent.change(screen.getByLabelText("Backup passphrase"), { target: { value: "abc" } });
    fireEvent.click(screen.getByRole("button", { name: "Restore" }));

    expect(await screen.findByText(/already has a different key/)).toBeInTheDocument();
    expect(saveStoredIdentity).not.toHaveBeenCalled();
  });

  it("requires explicit confirmation before overwriting a different server key", async () => {
    importIdentityBackup.mockResolvedValue({ publicKey: "pk-imported", secretKey: "sk-imported" });
    fetchIdentityPublicKey.mockResolvedValue({
      v: 1,
      alg: "hybrid-kem-mlkem768-x25519",
      publicKey: "pk-current",
    });

    render(<RestoreIdentityDialog userId="user-1" onClose={vi.fn()} />);

    selectFile();
    fireEvent.change(screen.getByLabelText("Backup passphrase"), { target: { value: "abc" } });
    fireEvent.click(screen.getByRole("button", { name: "Restore" }));

    expect(await screen.findByText(/already has a different key/)).toBeInTheDocument();
    expect(saveStoredIdentity).not.toHaveBeenCalled();

    fireEvent.click(screen.getByRole("button", { name: "Overwrite and restore" }));

    await waitFor(() => {
      expect(saveStoredIdentity).toHaveBeenCalledWith("user-1", {
        publicKey: "pk-imported",
        secretKey: "sk-imported",
      });
    });
    expect(await screen.findByText(/Backup restored/)).toBeInTheDocument();
  });

  it("lets the user cancel out of the overwrite confirmation", async () => {
    importIdentityBackup.mockResolvedValue({ publicKey: "pk-imported", secretKey: "sk-imported" });
    fetchIdentityPublicKey.mockResolvedValue({
      v: 1,
      alg: "hybrid-kem-mlkem768-x25519",
      publicKey: "pk-current",
    });

    render(<RestoreIdentityDialog userId="user-1" onClose={vi.fn()} />);

    selectFile();
    fireEvent.change(screen.getByLabelText("Backup passphrase"), { target: { value: "abc" } });
    fireEvent.click(screen.getByRole("button", { name: "Restore" }));

    expect(await screen.findByRole("button", { name: "Cancel" })).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Cancel" }));

    expect(saveStoredIdentity).not.toHaveBeenCalled();
    expect(screen.getByLabelText("Backup file")).toBeInTheDocument();
  });

  it("surfaces the generic error message on a wrong passphrase", async () => {
    importIdentityBackup.mockRejectedValue(new Error("wrong passphrase or corrupted backup file"));

    render(<RestoreIdentityDialog userId="user-1" onClose={vi.fn()} />);

    selectFile();
    fireEvent.change(screen.getByLabelText("Backup passphrase"), { target: { value: "wrong" } });
    fireEvent.click(screen.getByRole("button", { name: "Restore" }));

    expect(await screen.findByRole("alert")).toHaveTextContent("wrong passphrase or corrupted backup file");
    expect(saveStoredIdentity).not.toHaveBeenCalled();
  });

  it("closes on escape", () => {
    const onClose = vi.fn();
    render(<RestoreIdentityDialog userId="user-1" onClose={onClose} />);

    fireEvent.keyDown(document, { key: "Escape" });
    expect(onClose).toHaveBeenCalled();
  });
});
