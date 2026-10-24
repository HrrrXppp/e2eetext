import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { ChatSettingsDialog } from "@/components/chat/ChatSettingsDialog";
import type { Chat, ChatMember } from "@/lib/chats";

const getChatMembers = vi.fn();
const removeMember = vi.fn();

vi.mock("@/lib/chats", () => ({
  getChatMembers: (...args: unknown[]) => getChatMembers(...args),
  removeMember: (...args: unknown[]) => removeMember(...args),
}));

const chat: Chat = {
  id: "node/chat-1",
  name: "General",
  disappearAfterMinutes: 86400,
  createdAt: "2026-06-11T12:00:00.000Z",
  updatedAt: "2026-06-11T12:00:00.000Z",
  isAdmin: true,
};

function member(id: string, name: string, isAdmin = false): ChatMember {
  return {
    id,
    name,
    isAdmin,
    oidcProviderId: "node/provider",
    subject: `sub-${id}`,
    createdAt: "2026-06-11T12:00:00.000Z",
    updatedAt: "2026-06-11T12:00:00.000Z",
  };
}

const alice = member("node/alice", "Alice", true);
const bob = member("node/bob", "Bob");

describe("ChatSettingsDialog", () => {
  beforeEach(() => {
    getChatMembers.mockReset();
    removeMember.mockReset();
    getChatMembers.mockResolvedValue([alice, bob]);
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("lists members and offers Remove only for other members when admin", async () => {
    render(<ChatSettingsDialog chat={chat} currentUserId="node/alice" onClose={vi.fn()} />);

    expect(await screen.findByText("Bob")).toBeInTheDocument();
    expect(getChatMembers).toHaveBeenCalledWith("node/chat-1");
    expect(screen.getByRole("button", { name: "Remove Bob" })).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Remove Alice" })).not.toBeInTheDocument();
  });

  it("hides Remove buttons for non-admins", async () => {
    render(
      <ChatSettingsDialog
        chat={{ ...chat, isAdmin: false }}
        currentUserId="node/bob"
        onClose={vi.fn()}
      />,
    );

    expect(await screen.findByText("Alice")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /^Remove/ })).not.toBeInTheDocument();
  });

  it("removes a member after confirmation", async () => {
    vi.spyOn(window, "confirm").mockReturnValue(true);
    removeMember.mockResolvedValue(undefined);
    render(<ChatSettingsDialog chat={chat} currentUserId="node/alice" onClose={vi.fn()} />);

    fireEvent.click(await screen.findByRole("button", { name: "Remove Bob" }));

    await waitFor(() => {
      expect(removeMember).toHaveBeenCalledWith("node/chat-1", "node/bob");
    });
    await waitFor(() => {
      expect(screen.queryByText("Bob")).not.toBeInTheDocument();
    });
  });

  it("does nothing when the confirmation is declined", async () => {
    vi.spyOn(window, "confirm").mockReturnValue(false);
    render(<ChatSettingsDialog chat={chat} currentUserId="node/alice" onClose={vi.fn()} />);

    fireEvent.click(await screen.findByRole("button", { name: "Remove Bob" }));

    expect(removeMember).not.toHaveBeenCalled();
    expect(screen.getByText("Bob")).toBeInTheDocument();
  });

  it("shows an error when removal fails and keeps the member", async () => {
    vi.spyOn(window, "confirm").mockReturnValue(true);
    removeMember.mockRejectedValue(new Error("failed"));
    render(<ChatSettingsDialog chat={chat} currentUserId="node/alice" onClose={vi.fn()} />);

    fireEvent.click(await screen.findByRole("button", { name: "Remove Bob" }));

    expect(await screen.findByRole("alert")).toHaveTextContent("Could not remove member. Try again.");
    expect(screen.getByText("Bob")).toBeInTheDocument();
  });

  it("shows an error when members cannot be loaded", async () => {
    getChatMembers.mockRejectedValue(new Error("failed"));
    render(<ChatSettingsDialog chat={chat} currentUserId="node/alice" onClose={vi.fn()} />);

    expect(await screen.findByRole("alert")).toHaveTextContent("Could not load chat members");
  });

  it("closes on escape", async () => {
    const onClose = vi.fn();
    render(<ChatSettingsDialog chat={chat} currentUserId="node/alice" onClose={onClose} />);
    await screen.findByText("Bob");

    fireEvent.keyDown(document, { key: "Escape" });
    expect(onClose).toHaveBeenCalled();
  });
});
