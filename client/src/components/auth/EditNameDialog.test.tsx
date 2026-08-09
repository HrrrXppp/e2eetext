import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { EditNameDialog } from "@/components/auth/EditNameDialog";

const updateUserName = vi.fn();

vi.mock("@/lib/users", () => ({
  updateUserName: (...args: unknown[]) => updateUserName(...args),
}));

describe("EditNameDialog", () => {
  it("saves trimmed name", async () => {
    updateUserName.mockResolvedValue({
      id: "user-1",
      name: "Alice",
      oidcProviderId: "provider-1",
      subject: "sub-1",
      createdAt: "2026-06-11T12:00:00.000Z",
      updatedAt: "2026-06-11T12:00:00.000Z",
    });
    const onSaved = vi.fn();
    const onClose = vi.fn();

    render(
      <EditNameDialog
        userId="user-1"
        currentName=" Old "
        onClose={onClose}
        onSaved={onSaved}
      />,
    );

    fireEvent.change(screen.getByLabelText(/Display name/i), { target: { value: " Alice " } });
    fireEvent.click(screen.getByRole("button", { name: "Save name" }));

    await waitFor(() => {
      expect(updateUserName).toHaveBeenCalledWith("user-1", "Alice");
    });
    expect(onSaved).toHaveBeenCalledWith("Alice");
  });

  it("shows error when save fails", async () => {
    updateUserName.mockRejectedValue(new Error("failed"));
    render(
      <EditNameDialog userId="user-1" onClose={vi.fn()} onSaved={vi.fn()} />,
    );

    fireEvent.change(screen.getByLabelText(/Display name/i), { target: { value: "Alice" } });
    fireEvent.click(screen.getByRole("button", { name: "Save name" }));

    expect(await screen.findByRole("alert")).toHaveTextContent("Could not update name. Try again.");
  });

  it("closes on escape", () => {
    const onClose = vi.fn();
    render(
      <EditNameDialog userId="user-1" onClose={onClose} onSaved={vi.fn()} />,
    );

    fireEvent.keyDown(document, { key: "Escape" });
    expect(onClose).toHaveBeenCalled();
  });
});
