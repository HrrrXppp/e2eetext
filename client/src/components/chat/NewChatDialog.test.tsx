import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { NewChatDialog } from "@/components/chat/NewChatDialog";
import { encodeBase64 } from "@/lib/bytes";
import { generateKemKeyPair } from "@/lib/crypto";

const createChat = vi.fn();

vi.mock("@/lib/chats", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@/lib/chats")>();
  return {
    ...actual,
    createChat: (...args: unknown[]) => createChat(...args),
  };
});

const memberKemPublicKey = encodeBase64(generateKemKeyPair().publicKey);
const currentUserKemPublicKey = encodeBase64(generateKemKeyPair().publicKey);

vi.mock("@/components/chat/NewChatMemberSearch", () => ({
  NewChatMemberSearch: ({
    onAddMember,
    onBack,
  }: {
    onAddMember: (member: { id: string; name?: string; kemPublicKey: string }) => void;
    onBack: () => void;
  }) => (
    <div>
      <button
        type="button"
        onClick={() => onAddMember({ id: "member-1", name: "Bob", kemPublicKey: memberKemPublicKey })}
      >
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
      <NewChatDialog
        currentUserId="user-1"
        currentUserKemPublicKey={currentUserKemPublicKey}
        onClose={vi.fn()}
        onCreated={vi.fn()}
      />,
    );

    fireEvent.click(screen.getByRole("button", { name: "Create chat" }));

    expect(await screen.findByRole("alert")).toHaveTextContent("Chat name is required.");
    expect(createChat).not.toHaveBeenCalled();
  });

  it("creates chat with current user and added members, wrapping the chat key for each", async () => {
    createChat.mockReset();
    createChat.mockResolvedValue({
      id: "node/chat-1",
      name: "Project",
      adminUserId: "user-1",
      kemPublicKey: "chat-kem-public-key",
      wrappedChatPrivateKey: "wrapped",
      kemCiphertext: "ct",
      createdAt: "2026-06-11T12:00:00.000Z",
      updatedAt: "2026-06-11T12:00:00.000Z",
    });
    const onCreated = vi.fn();
    const onClose = vi.fn();

    render(
      <NewChatDialog
        currentUserId="user-1"
        currentUserKemPublicKey={currentUserKemPublicKey}
        onClose={onClose}
        onCreated={onCreated}
      />,
    );

    fireEvent.change(screen.getByLabelText(/Chat name/i), { target: { value: "Project" } });
    fireEvent.click(screen.getByRole("button", { name: "Search users" }));
    fireEvent.click(screen.getByRole("button", { name: "Add mock member" }));
    fireEvent.click(screen.getByRole("button", { name: "Create chat" }));

    await waitFor(() => {
      expect(createChat).toHaveBeenCalledTimes(1);
    });

    const [input] = createChat.mock.calls[0];
    expect(input.name).toBe("Project");
    expect(typeof input.kemPublicKey).toBe("string");
    expect(input.kemPublicKey.length).toBeGreaterThan(0);
    expect(input.members).toHaveLength(2);
    expect(input.members.map((member: { userId: string }) => member.userId)).toEqual([
      "user-1",
      "member-1",
    ]);
    for (const member of input.members) {
      expect(typeof member.wrappedChatPrivateKey).toBe("string");
      expect(member.wrappedChatPrivateKey.length).toBeGreaterThan(0);
      expect(typeof member.kemCiphertext).toBe("string");
      expect(member.kemCiphertext.length).toBeGreaterThan(0);
    }

    expect(onCreated).toHaveBeenCalled();
    expect(onClose).toHaveBeenCalled();
  });

  it("returns from search view on escape", () => {
    render(
      <NewChatDialog
        currentUserId="user-1"
        currentUserKemPublicKey={currentUserKemPublicKey}
        onClose={vi.fn()}
        onCreated={vi.fn()}
      />,
    );

    fireEvent.click(screen.getByRole("button", { name: "Search users" }));
    expect(screen.getByRole("button", { name: "Add mock member" })).toBeInTheDocument();

    fireEvent.keyDown(document, { key: "Escape" });
    expect(screen.getByLabelText(/Chat name/i)).toBeInTheDocument();
  });
});
