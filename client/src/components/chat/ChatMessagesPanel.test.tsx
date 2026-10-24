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

  it("opens chat settings from the header button", () => {
    const onOpenSettings = vi.fn();
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
        onOpenSettings={onOpenSettings}
      />,
    );

    fireEvent.click(screen.getByRole("button", { name: "Chat settings" }));
    expect(onOpenSettings).toHaveBeenCalled();
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

  describe("scrolling to the latest message", () => {
    const readMessage = {
      ...messagesWithUnread[0],
      unread: false,
    };
    const firstUnread = {
      ...messagesWithUnread[1],
      id: "first-unread",
      unread: true,
    };
    const otherChat = { ...chat, id: "22222222-2222-2222-2222-222222222222" };

    function panel(
      chatArg: typeof chat,
      messages: Array<typeof readMessage>,
      loading = false,
    ) {
      return (
        <ChatMessagesPanel
          chat={chatArg}
          messages={messages}
          currentUserId="current-user"
          loading={loading}
          error={null}
          sending={false}
          sendError={null}
          onSend={vi.fn()}
          onMarkRead={vi.fn()}
        />
      );
    }

    function stubScrollMetrics() {
      vi.spyOn(HTMLElement.prototype, "scrollIntoView").mockImplementation(() => {});
      vi.spyOn(HTMLElement.prototype, "scrollHeight", "get").mockReturnValue(900);
      const scrollTopSet = vi.fn();
      vi.spyOn(HTMLElement.prototype, "scrollTop", "set").mockImplementation(scrollTopSet);
      return scrollTopSet;
    }

    it("scrolls to the bottom once when a chat with no unread messages opens", () => {
      const scrollTopSet = stubScrollMetrics();

      const { rerender } = render(panel(chat, [], true));
      expect(scrollTopSet).not.toHaveBeenCalled();

      rerender(panel(chat, [readMessage]));
      expect(scrollTopSet).toHaveBeenCalledTimes(1);
      expect(scrollTopSet).toHaveBeenCalledWith(900);
    });

    it("scrolls to the bottom when the first message is the first unread one", () => {
      const scrollTopSet = stubScrollMetrics();

      render(panel(chat, [firstUnread]));

      expect(scrollTopSet).toHaveBeenCalledTimes(1);
      expect(scrollTopSet).toHaveBeenCalledWith(900);
    });

    it("does not scroll to the bottom when an unread divider is shown", () => {
      const scrollTopSet = stubScrollMetrics();

      render(panel(chat, messagesWithUnread));

      expect(scrollTopSet).not.toHaveBeenCalled();
    });

    it("does not scroll again when new messages arrive in the same chat", () => {
      const scrollTopSet = stubScrollMetrics();

      const { rerender } = render(panel(chat, [readMessage]));
      expect(scrollTopSet).toHaveBeenCalledTimes(1);

      rerender(panel(chat, [readMessage, { ...readMessage, id: "read-2" }]));
      rerender(panel(chat, [readMessage, { ...readMessage, id: "read-2" }], true));
      rerender(panel(chat, [readMessage, { ...readMessage, id: "read-2" }]));

      expect(scrollTopSet).toHaveBeenCalledTimes(1);
    });

    it("waits for the opened chat's messages instead of scrolling on the pre-fetch render", () => {
      const scrollTopSet = stubScrollMetrics();

      const { rerender } = render(panel(chat, [readMessage]));
      expect(scrollTopSet).toHaveBeenCalledTimes(1);

      // Chat switch: first render has loading=false and the previous chat's messages.
      rerender(panel(otherChat, [readMessage]));
      rerender(panel(otherChat, [], true));
      expect(scrollTopSet).toHaveBeenCalledTimes(1);

      rerender(panel(otherChat, [{ ...readMessage, chatId: otherChat.id }]));
      expect(scrollTopSet).toHaveBeenCalledTimes(2);
    });

    it("keeps the message list mounted while the open chat reloads, so the scroll position is not reset", () => {
      stubScrollMetrics();

      const { rerender } = render(panel(chat, [readMessage]));
      const list = screen.getByRole("list");

      rerender(panel(chat, [readMessage], true));

      expect(screen.getByRole("list")).toBe(list);
      expect(screen.queryByText("Loading messages...")).not.toBeInTheDocument();
    });

    it("shows the loading state instead of the previous chat's messages while a new chat loads", () => {
      stubScrollMetrics();

      const { rerender } = render(panel(chat, [readMessage]));
      rerender(panel(otherChat, [readMessage], true));

      expect(screen.getByText("Loading messages...")).toBeInTheDocument();
      expect(screen.queryByRole("list")).not.toBeInTheDocument();
    });

    it("scrolls to the bottom again when switching to another chat", () => {
      const scrollTopSet = stubScrollMetrics();

      const { rerender } = render(panel(chat, [readMessage]));
      expect(scrollTopSet).toHaveBeenCalledTimes(1);

      rerender(panel(otherChat, [{ ...readMessage, chatId: otherChat.id }]));

      expect(scrollTopSet).toHaveBeenCalledTimes(2);
    });
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

  it("renders a long unbroken sender name unmodified, relying on CSS to truncate it (regression for #63)", () => {
    const longName = "7".repeat(200);
    render(
      <ChatMessagesPanel
        chat={chat}
        messages={[
          {
            id: "long-name-1",
            chatId: chat.id,
            userId: "99999999-9999-9999-9999-999999999999/other-user",
            userName: longName,
            data: "r23r23r3",
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

    const authorEl = document.querySelector(".chats-page__message-author");
    expect(authorEl).not.toBeNull();
    expect(authorEl!.textContent).toBe(longName);
  });

  it("truncates a long unbroken sender name with an ellipsis instead of letting it overflow the bubble (regression for #63)", () => {
    // Issue #63: the screenshot showed the long unbroken string overflowing
    // next to the message bubble was the sender's own long display name
    // (rendered via .chats-page__message-author), not the message body. This
    // guards the CSS truncation rule, mirroring .site-head__user-name.
    const cssPath = path.resolve(
      path.dirname(fileURLToPath(import.meta.url)),
      "../../styles/index.css",
    );
    const css = readFileSync(cssPath, "utf8");

    const authorMatch = css.match(/\.chats-page__message-author\s*\{([^}]*)\}/);
    expect(authorMatch).not.toBeNull();
    expect(authorMatch![1]).toMatch(/overflow:\s*hidden/);
    expect(authorMatch![1]).toMatch(/white-space:\s*nowrap/);
    expect(authorMatch![1]).toMatch(/text-overflow:\s*ellipsis/);
  });
});
