import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { NewChatDialog } from "@/components/chat/NewChatDialog";

const createChat = vi.fn();

vi.mock("@/lib/chats", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@/lib/chats")>();
  return {
    ...actual,
    createChat: (...args: unknown[]) => createChat(...args),
  };
});

vi.mock("@/components/chat/NewChatMemberSearch", () => ({
  NewChatMemberSearch: ({
    onAddMember,
    onBack,
  }: {
    onAddMember: (member: { id: string; name?: string }) => void;
    onBack: () => void;
  }) => (
    <div>
      <button type="button" onClick={() => onAddMember({ id: "member-1", name: "Bob" })}>
        Add mock member
      </button>
      <button type="button" onClick={onBack}>
        Back from search
      </button>
    </div>
  ),
}));

describe("NewChatDialog", () => {
  it("requires chat name before create", async () => {
    createChat.mockReset();
    render(
      <NewChatDialog currentUserId="user-1" onClose={vi.fn()} onCreated={vi.fn()} />,
    );

    fireEvent.click(screen.getByRole("button", { name: "Create chat" }));

    expect(await screen.findByRole("alert")).toHaveTextContent("Chat name is required.");
    expect(createChat).not.toHaveBeenCalled();
  });

  it("creates chat with current user and added members", async () => {
    createChat.mockResolvedValue({
      id: "node/chat-1",
      name: "Project",
      createdAt: "2026-06-11T12:00:00.000Z",
      updatedAt: "2026-06-11T12:00:00.000Z",
    });
    const onCreated = vi.fn();
    const onClose = vi.fn();

    render(
      <NewChatDialog currentUserId="user-1" onClose={onClose} onCreated={onCreated} />,
    );

    fireEvent.change(screen.getByLabelText(/Chat name/i), { target: { value: "Project" } });
    fireEvent.click(screen.getByRole("button", { name: "Search users" }));
    fireEvent.click(screen.getByRole("button", { name: "Add mock member" }));
    fireEvent.click(screen.getByRole("button", { name: "Create chat" }));

    await waitFor(() => {
      expect(createChat).toHaveBeenCalledWith({
        name: "Project",
        usersUids: ["user-1", "member-1"],
      });
    });
    expect(onCreated).toHaveBeenCalled();
    expect(onClose).toHaveBeenCalled();
  });

  it("returns from search view on escape", () => {
    render(
      <NewChatDialog currentUserId="user-1" onClose={vi.fn()} onCreated={vi.fn()} />,
    );

    fireEvent.click(screen.getByRole("button", { name: "Search users" }));
    expect(screen.getByRole("button", { name: "Add mock member" })).toBeInTheDocument();

    fireEvent.keyDown(document, { key: "Escape" });
    expect(screen.getByLabelText(/Chat name/i)).toBeInTheDocument();
  });
});
