import { readFileSync } from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";
import { fireEvent, render, screen } from "@testing-library/react";
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

  it("does not show a back button when onBack is not provided", () => {
    render(
      <ChatMessagesPanel
        chat={chat}
        messages={[]}
        currentUserId="current-user"
        loading={false}
        error={null}
        sending={false}
        sendError={null}
        onSend={vi.fn()}
        onMarkRead={vi.fn()}
      />,
    );

    expect(
      screen.queryByRole("button", { name: "Back to chats" }),
    ).not.toBeInTheDocument();
  });

  it("calls onBack when the back button is clicked", () => {
    const onBack = vi.fn();

    render(
      <ChatMessagesPanel
        chat={chat}
        messages={[]}
        currentUserId="current-user"
        loading={false}
        error={null}
        sending={false}
        sendError={null}
        onSend={vi.fn()}
        onMarkRead={vi.fn()}
        onBack={onBack}
      />,
    );

    fireEvent.click(screen.getByRole("button", { name: "Back to chats" }));

    expect(onBack).toHaveBeenCalledTimes(1);
  });

  it("groups the back button with the title, separate from the meta caption (regression for alignment bug)", () => {
    render(
      <ChatMessagesPanel
        chat={chat}
        messages={[]}
        currentUserId="current-user"
        loading={false}
        error={null}
        sending={false}
        sendError={null}
        onSend={vi.fn()}
        onMarkRead={vi.fn()}
        onBack={vi.fn()}
      />,
    );

    const backButton = screen.getByRole("button", { name: "Back to chats" });
    const title = screen.getByRole("heading", { name: chat.name });
    const heading = backButton.parentElement;

    // The back button and title must share the same single-line wrapper so
    // they can be vertically centered against each other, not against the
    // two-line title+caption block.
    expect(heading).toHaveClass("chats-page__panel-heading");
    expect(title.parentElement).toBe(heading);

    const meta = document.querySelector(".chats-page__panel-meta");
    expect(meta?.parentElement).not.toBe(heading);
    expect(meta?.parentElement).toBe(heading?.parentElement);
  });

  it("renders a long unbroken message unmodified, relying on CSS to wrap it (regression for #63)", () => {
    const longToken = "7".repeat(200);
    render(
      <ChatMessagesPanel
        chat={chat}
        messages={[
          {
            id: "long-1",
            chatId: chat.id,
            userId: "99999999-9999-9999-9999-999999999999/other-user",
            userName: "Evgenii Khrunov",
            data: longToken,
            createdAt: "2026-06-11T12:00:00.000Z",
            updatedAt: "2026-06-11T12:00:00.000Z",
            unread: false,
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

    const textEl = document.querySelector(".chats-page__message-text");
    expect(textEl).not.toBeNull();
    expect(textEl!.textContent).toBe(longToken);
  });

  it("wraps long unbroken message text instead of letting it overflow the bubble (regression for #63)", () => {
    // Issue #63: a message consisting of one long unbroken run of characters
    // (no spaces) overflowed past the right edge of the message bubble
    // instead of wrapping. This guards the CSS chain that forces the text to
    // break and caps the bubble container's width regardless of content.
    const cssPath = path.resolve(
      path.dirname(fileURLToPath(import.meta.url)),
      "../../styles/index.css",
    );
    const css = readFileSync(cssPath, "utf8");

    const textMatch = css.match(/\.chats-page__message-text\s*\{([^}]*)\}/);
    expect(textMatch).not.toBeNull();
    expect(textMatch![1]).toMatch(/overflow-wrap:\s*anywhere/);
    expect(textMatch![1]).toMatch(/word-break:\s*break-word/);

    const bubbleMatch = css.match(/\.chats-page__message\s*\{([^}]*)\}/);
    expect(bubbleMatch).not.toBeNull();
    expect(bubbleMatch![1]).toMatch(/min-width:\s*0/);
    expect(bubbleMatch![1]).toMatch(/max-width:\s*75%/);
  });
});
