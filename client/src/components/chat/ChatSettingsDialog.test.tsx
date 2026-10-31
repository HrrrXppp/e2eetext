import { fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { ChatSettingsDialog } from "@/components/chat/ChatSettingsDialog";
import type { Chat, ChatMember } from "@/lib/chats";

const getChatMembers = vi.fn();
const removeMember = vi.fn();
const addMembers = vi.fn();
const prepareAddMembers = vi.fn();
const searchUsers = vi.fn();

vi.mock("@/lib/chats", () => ({
  getChatMembers: (...args: unknown[]) => getChatMembers(...args),
  removeMember: (...args: unknown[]) => removeMember(...args),
  addMembers: (...args: unknown[]) => addMembers(...args),
}));

vi.mock("@/lib/e2ee/session", () => ({
  prepareAddMembers: (...args: unknown[]) => prepareAddMembers(...args),
}));

vi.mock("@/lib/users", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/lib/users")>()),
  searchUsers: (...args: unknown[]) => searchUsers(...args),
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
    addMembers.mockReset();
    prepareAddMembers.mockReset();
    searchUsers.mockReset();
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

  it("hides Add members for non-admins", async () => {
    render(
      <ChatSettingsDialog chat={{ ...chat, isAdmin: false }} currentUserId="node/bob" onClose={vi.fn()} />,
    );
    await screen.findByText("Alice");
    expect(screen.queryByRole("button", { name: "Add members" })).not.toBeInTheDocument();
  });

  async function selectUser(name: string) {
    fireEvent.click(await screen.findByRole("button", { name: "Add members" }));
    fireEvent.change(screen.getByLabelText(/Search/i), { target: { value: name } });
    fireEvent.click(screen.getByRole("button", { name: "Search" }));
    const row = (await screen.findByText(name)).closest("li") as HTMLElement;
    fireEvent.click(within(row).getByRole("button", { name: "Add" }));
  }

  it("adds several selected users with a single request and refreshes the member list", async () => {
    const carol = member("node/carol", "Carol");
    const dave = member("node/dave", "Dave");
    searchUsers.mockImplementation(async (query: string) =>
      [alice, bob, carol, dave].filter((user) => user.name === query),
    );
    prepareAddMembers.mockResolvedValue({ keyId: "k2", wraps: [{ userId: "node/carol", wrap: {} }] });
    addMembers.mockResolvedValue(undefined);
    render(<ChatSettingsDialog chat={chat} currentUserId="node/alice" onClose={vi.fn()} />);
    await screen.findByText("Bob");

    await selectUser("Carol");
    await selectUser("Dave");
    getChatMembers.mockResolvedValue([alice, bob, carol, dave]);
    fireEvent.click(screen.getByRole("button", { name: "Add 2 members" }));

    await waitFor(() => {
      expect(addMembers).toHaveBeenCalledTimes(1);
    });
    expect(prepareAddMembers).toHaveBeenCalledWith("node/alice", "node/chat-1", ["node/carol", "node/dave"]);
    expect(addMembers).toHaveBeenCalledWith(
      "node/chat-1",
      ["node/carol", "node/dave"],
      "k2",
      [{ userId: "node/carol", wrap: {} }],
    );
    expect(await screen.findByText("Dave")).toBeInTheDocument();
    expect(screen.getByRole("list", { name: "Chat members" })).toHaveTextContent("Carol");
    expect(screen.queryByRole("button", { name: "Add 2 members" })).not.toBeInTheDocument();
  });

  it("excludes current members from the search and shows an error when adding fails", async () => {
    const carol = member("node/carol", "Carol");
    searchUsers.mockResolvedValue([bob, carol]);
    prepareAddMembers.mockRejectedValue(new Error("identity key not found for node/carol"));
    render(<ChatSettingsDialog chat={chat} currentUserId="node/alice" onClose={vi.fn()} />);
    await screen.findByText("Bob");

    fireEvent.click(screen.getByRole("button", { name: "Add members" }));
    fireEvent.change(screen.getByLabelText(/Search/i), { target: { value: "Carol" } });
    fireEvent.click(screen.getByRole("button", { name: "Search" }));
    const results = await screen.findByRole("list", { name: "Search results" });
    expect(within(results).queryByText("Bob")).not.toBeInTheDocument();
    fireEvent.click(within(results).getByRole("button", { name: "Add" }));
    fireEvent.click(screen.getByRole("button", { name: "Add 1 member" }));

    expect(await screen.findByRole("alert")).toHaveTextContent("identity key not found for node/carol");
    expect(addMembers).not.toHaveBeenCalled();
  });
});
