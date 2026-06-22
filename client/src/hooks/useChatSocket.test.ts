import { act, renderHook, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { useChatSocket } from "@/hooks/useChatSocket";
import { TOKEN_REFRESHED_EVENT } from "@/lib/auth";
import { makeToken } from "@/test/makeToken";

class MockWebSocket {
  static instances: MockWebSocket[] = [];
  static readonly CONNECTING = 0;
  static readonly OPEN = 1;
  static readonly CLOSING = 2;
  static readonly CLOSED = 3;
  url: string;
  readyState = MockWebSocket.OPEN;
  onopen: (() => void) | null = null;
  onclose: ((event: { code: number }) => void) | null = null;
  onmessage: ((event: { data: string }) => void) | null = null;
  onerror: (() => void) | null = null;
  send = vi.fn();

  constructor(url: string) {
    this.url = url;
    MockWebSocket.instances.push(this);
    queueMicrotask(() => this.onopen?.());
  }

  close(code = 1000) {
    this.readyState = MockWebSocket.CLOSED;
    this.onclose?.({ code });
  }

  emitMessage(data: string) {
    this.onmessage?.({ data });
  }
}

function storeValidIdToken() {
  localStorage.setItem(
    "messenger_id_token",
    makeToken({
      sub: "google-subject-1",
      iss: "https://accounts.google.com",
      exp: Math.floor(Date.now() / 1000) + 3600,
    }),
  );
}

describe("useChatSocket", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
    MockWebSocket.instances = [];
    localStorage.clear();
  });

  it("stays idle when disabled", () => {
    vi.stubGlobal("WebSocket", MockWebSocket as unknown as typeof WebSocket);
    const { result } = renderHook(() => useChatSocket(false));
    expect(result.current).toBe("idle");
    expect(MockWebSocket.instances).toHaveLength(0);
  });

  it("opens connection and dispatches chat events", async () => {
    storeValidIdToken();
    vi.stubGlobal("WebSocket", MockWebSocket as unknown as typeof WebSocket);

    const onChatUnread = vi.fn();
    const onChatAdded = vi.fn();
    const { result } = renderHook(() => useChatSocket(true, onChatUnread, onChatAdded));

    await waitFor(() => {
      expect(result.current).toBe("open");
    });

    const socket = MockWebSocket.instances[0];
    socket.emitMessage(
      JSON.stringify({
        type: "chat.unread",
        chats: [{ chatId: "node/chat-1", unreadMessageCount: 2, updatedAt: "2026-06-11T12:00:00.000Z" }],
      }),
    );
    socket.emitMessage(JSON.stringify({ type: "chat.added", chatId: "node/chat-2" }));

    expect(onChatUnread).toHaveBeenCalled();
    expect(onChatAdded).toHaveBeenCalled();
  });

  it("does not open websocket with only an access token", async () => {
    vi.useFakeTimers();
    localStorage.setItem("messenger_access_token", "ya29.access-token");
    vi.stubGlobal("WebSocket", MockWebSocket as unknown as typeof WebSocket);

    renderHook(() => useChatSocket(true));

    await act(async () => {
      await Promise.resolve();
    });

    expect(MockWebSocket.instances).toHaveLength(0);

    act(() => {
      vi.advanceTimersByTime(1_000);
    });
    expect(MockWebSocket.instances).toHaveLength(0);

    vi.useRealTimers();
  });

  it("reconnects after token refresh event", async () => {
    storeValidIdToken();
    vi.stubGlobal("WebSocket", MockWebSocket as unknown as typeof WebSocket);

    renderHook(() => useChatSocket(true));

    await waitFor(() => {
      expect(MockWebSocket.instances).toHaveLength(1);
    });

    window.dispatchEvent(new Event(TOKEN_REFRESHED_EVENT));

    await waitFor(() => {
      expect(MockWebSocket.instances.length).toBeGreaterThan(1);
    });
  });

  it("sends heartbeat immediately and every 30 seconds", async () => {
    vi.useFakeTimers();
    storeValidIdToken();
    vi.stubGlobal("WebSocket", MockWebSocket as unknown as typeof WebSocket);

    renderHook(() => useChatSocket(true));

    await act(async () => {
      await Promise.resolve();
    });

    const socket = MockWebSocket.instances[0];
    expect(socket.send).toHaveBeenCalledWith(JSON.stringify({ type: "heartbeat" }));

    act(() => {
      vi.advanceTimersByTime(30_000);
    });
    expect(socket.send).toHaveBeenCalledTimes(2);

    act(() => {
      vi.advanceTimersByTime(30_000);
    });
    expect(socket.send).toHaveBeenCalledTimes(3);

    vi.useRealTimers();
  });

  it("keeps heartbeat on the new socket when a stale socket closes", async () => {
    vi.useFakeTimers();
    storeValidIdToken();
    vi.stubGlobal("WebSocket", MockWebSocket as unknown as typeof WebSocket);

    renderHook(() => useChatSocket(true));

    await act(async () => {
      await Promise.resolve();
    });

    const first = MockWebSocket.instances[0];
    const staleOnClose = first.onclose;
    first.onclose = null;

    window.dispatchEvent(new Event(TOKEN_REFRESHED_EVENT));

    await act(async () => {
      await Promise.resolve();
    });

    const second = MockWebSocket.instances[1];
    expect(second.send).toHaveBeenCalledWith(JSON.stringify({ type: "heartbeat" }));

    staleOnClose?.({ code: 1000 });

    act(() => {
      vi.advanceTimersByTime(30_000);
    });
    expect(second.send).toHaveBeenCalledTimes(2);

    vi.useRealTimers();
  });
});
