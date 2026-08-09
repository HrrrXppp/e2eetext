import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { ChatSidebar } from "@/components/chat/ChatSidebar";
import type { Chat } from "@/lib/chats";

const chats: Chat[] = [
  {
    id: "node/chat-1",
    name: "General",
    disappearAfterMinutes: 86400,
    createdAt: "2026-06-11T12:00:00.000Z",
    updatedAt: "2026-06-11T12:00:00.000Z",
    unreadMessageCount: 3,
  },
  {
    id: "node/chat-2",
    name: "",
    disappearAfterMinutes: 86400,
    createdAt: "2026-06-10T12:00:00.000Z",
    updatedAt: "2026-06-10T12:00:00.000Z",
    unreadMessageCount: 0,
  },
];

describe("ChatSidebar", () => {
  it("shows loading state", () => {
    render(
      <ChatSidebar
        chats={[]}
        selectedChatId={null}
        loading
        error={null}
        onSelect={vi.fn()}
        onNewChat={vi.fn()}
      />,
    );
    expect(screen.getByText("Loading chats...")).toBeInTheDocument();
  });

  it("shows error state", () => {
    render(
      <ChatSidebar
        chats={[]}
        selectedChatId={null}
        loading={false}
        error="Failed to load chats"
        onSelect={vi.fn()}
        onNewChat={vi.fn()}
      />,
    );
    expect(screen.getByRole("alert")).toHaveTextContent("Failed to load chats");
  });

  it("renders chats with unread badge and selection", () => {
    const onSelect = vi.fn();
    render(
      <ChatSidebar
        chats={chats}
        selectedChatId="node/chat-1"
        loading={false}
        error={null}
        onSelect={onSelect}
        onNewChat={vi.fn()}
      />,
    );

    expect(screen.getByText("General")).toBeInTheDocument();
    expect(screen.getByLabelText("3 unread messages")).toHaveTextContent("3");
    expect(screen.getByText("Unnamed chat")).toBeInTheDocument();
    expect(screen.getByRole("option", { name: /General/i })).toHaveAttribute("aria-selected", "true");

    fireEvent.click(screen.getByRole("option", { name: /Unnamed chat/i }));
    expect(onSelect).toHaveBeenCalledWith("node/chat-2");
  });

  it("calls onNewChat", () => {
    const onNewChat = vi.fn();
    render(
      <ChatSidebar
        chats={[]}
        selectedChatId={null}
        loading={false}
        error={null}
        onSelect={vi.fn()}
        onNewChat={onNewChat}
      />,
    );

    fireEvent.click(screen.getByRole("button", { name: "New chat" }));
    expect(onNewChat).toHaveBeenCalled();
  });
});
