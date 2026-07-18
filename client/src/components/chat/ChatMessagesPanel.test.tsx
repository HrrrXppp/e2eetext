import { render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { ChatMessagesPanel } from "@/components/chat/ChatMessagesPanel";

const chat = {
  id: "11111111-1111-1111-1111-111111111111",
  name: "General",
  disappearAfterMinutes: 86400,
  createdAt: "2026-06-11T12:00:00.000Z",
  updatedAt: "2026-06-11T12:00:00.000Z",
};

const messagesWithUnread = [
  {
    id: "read-1",
    chatId: chat.id,
    userId: "99999999-9999-9999-9999-999999999999/other-user",
    userName: "Alice",
    data: "read message",
    createdAt: "2026-06-11T12:00:00.000Z",
    updatedAt: "2026-06-11T12:00:00.000Z",
    unread: false,
  },
  {
    id: "unread-1",
    chatId: chat.id,
    userId: "99999999-9999-9999-9999-999999999999/11111111-1111-1111-1111-111111111111",
    data: "unread message",
    createdAt: "2026-06-11T12:01:00.000Z",
    updatedAt: "2026-06-11T12:01:00.000Z",
    unread: true,
  },
];

describe("ChatMessagesPanel", () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("shows a divider between read and unread messages", () => {
    render(
      <ChatMessagesPanel
        chat={chat}
        messages={messagesWithUnread}
        currentUserId="current-user"
        loading={false}
        error={null}
        sending={false}
        sendError={null}
        onSend={vi.fn()}
        onMarkRead={vi.fn()}
      />,
    );

    expect(screen.getByRole("separator", { name: "Unread messages" })).toBeInTheDocument();
    expect(screen.getByText("Alice")).toBeInTheDocument();
    expect(screen.getByText("read message")).toBeInTheDocument();
    expect(screen.getByText("unread message")).toBeInTheDocument();
    expect(
      screen.getByText("99999999...111111111111"),
    ).toBeInTheDocument();
  });

  it("marks each message with its absolute disappear date and time", () => {
    render(
      <ChatMessagesPanel
        chat={chat}
        messages={messagesWithUnread}
        currentUserId="current-user"
        loading={false}
        error={null}
        sending={false}
        sendError={null}
        onSend={vi.fn()}
        onMarkRead={vi.fn()}
      />,
    );

    const stamps = document.querySelectorAll(".chats-page__message-time time");
    expect(stamps.length).toBe(4);
    expect(stamps[1]).toHaveAttribute("dateTime", "2026-08-10T12:00:00.000Z");
    expect(stamps[3]).toHaveAttribute("dateTime", "2026-08-10T12:01:00.000Z");
    expect(screen.getAllByText(/disappears/i).length).toBeGreaterThanOrEqual(2);
  });

  it("scrolls to the unread divider once when a chat opens", () => {
    const scrollIntoView = vi
      .spyOn(HTMLElement.prototype, "scrollIntoView")
      .mockImplementation(() => {});

    const { rerender } = render(
      <ChatMessagesPanel
        chat={chat}
        messages={[]}
        currentUserId="current-user"
        loading
        error={null}
        sending={false}
        sendError={null}
        onSend={vi.fn()}
        onMarkRead={vi.fn()}
      />,
    );

    expect(scrollIntoView).not.toHaveBeenCalled();

    rerender(
      <ChatMessagesPanel
        chat={chat}
        messages={messagesWithUnread}
        currentUserId="current-user"
        loading={false}
        error={null}
        sending={false}
        sendError={null}
        onSend={vi.fn()}
        onMarkRead={vi.fn()}
      />,
    );

    expect(scrollIntoView).toHaveBeenCalledTimes(1);
    expect(scrollIntoView).toHaveBeenCalledWith({ block: "start" });

    rerender(
      <ChatMessagesPanel
        chat={chat}
        messages={messagesWithUnread}
        currentUserId="current-user"
        loading
        error={null}
        sending={false}
        sendError={null}
        onSend={vi.fn()}
        onMarkRead={vi.fn()}
      />,
    );

    rerender(
      <ChatMessagesPanel
        chat={chat}
        messages={[
          ...messagesWithUnread,
          {
            id: "unread-2",
            chatId: chat.id,
            userId: "99999999-9999-9999-9999-999999999999/other-user",
            data: "another unread",
            createdAt: "2026-06-11T12:02:00.000Z",
            updatedAt: "2026-06-11T12:02:00.000Z",
            unread: true,
          },
        ]}
        currentUserId="current-user"
        loading={false}
        error={null}
        sending={false}
        sendError={null}
        onSend={vi.fn()}
        onMarkRead={vi.fn()}
      />,
    );

    expect(scrollIntoView).toHaveBeenCalledTimes(1);
  });
});
