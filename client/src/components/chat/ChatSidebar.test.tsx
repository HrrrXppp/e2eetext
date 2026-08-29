import { readFileSync } from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";
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

  it("packs .chats-page__list items at the top via flex layout (regression for #50)", () => {
    // Issue #50: .chats-page__list used display: grid with no grid-template-rows,
    // so Grid's default align-content (normal, behaves as stretch) spread
    // leftover vertical space evenly across implicit row tracks, pushing chat
    // items apart instead of packing them at the top. This guards against
    // that regression by asserting the rule stays flex + column, matching the
    // sibling .chats-page__message-list rule.
    const cssPath = path.resolve(
      path.dirname(fileURLToPath(import.meta.url)),
      "../../styles/index.css",
    );
    const css = readFileSync(cssPath, "utf8");
    const match = css.match(/\.chats-page__list\s*\{([^}]*)\}/);
    expect(match).not.toBeNull();
    const rule = match![1];
    expect(rule).toMatch(/display:\s*flex/);
    expect(rule).toMatch(/flex-direction:\s*column/);
    expect(rule).not.toMatch(/display:\s*grid/);
  });

  it("stacks the chat layout on narrow viewports instead of always using two columns (regression for #58)", () => {
    // Issue #58: .chats-page__layout was unconditionally display: flex (row
    // direction), so the sidebar (min-width 14rem) and message panel were
    // always squeezed side by side, even on phone-width viewports. This
    // guards against that regression by asserting the base rule stacks the
    // panes in a column and only switches to a row layout inside a
    // min-width media query (desktop and up).
    const cssPath = path.resolve(
      path.dirname(fileURLToPath(import.meta.url)),
      "../../styles/index.css",
    );
    const css = readFileSync(cssPath, "utf8");

    const baseMatch = css.match(/\.chats-page__layout\s*\{([^}]*)\}/);
    expect(baseMatch).not.toBeNull();
    expect(baseMatch![1]).toMatch(/flex-direction:\s*column/);

    const mediaBlockMatch = css.match(
      /@media \(min-width: 640px\) \{\s*\.chats-page__layout \{([^}]*)\}/,
    );
    expect(mediaBlockMatch).not.toBeNull();
    expect(mediaBlockMatch![1]).toMatch(/flex-direction:\s*row/);
  });
});
