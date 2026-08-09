import { render } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { useMarkMessageReadOnView } from "@/hooks/useMarkMessageReadOnView";
import type { Message } from "@/lib/messages";

const unreadMessage: Message = {
  id: "unread-1",
  chatId: "11111111-1111-1111-1111-111111111111",
  userId: "other-user",
  data: "unread message",
  createdAt: "2026-06-11T12:01:00.000Z",
  updatedAt: "2026-06-11T12:01:00.000Z",
  unread: true,
};

function TestHarness({ onMarkRead }: { onMarkRead: (messageId: string) => void }) {
  const scrollRootRef = { current: document.createElement("div") };
  const setMessageRef = useMarkMessageReadOnView(
    unreadMessage.chatId,
    [unreadMessage],
    "current-user",
    scrollRootRef,
    onMarkRead,
  );

  return (
    <article
      ref={(element) => setMessageRef(unreadMessage.id, element)}
      data-message-id={unreadMessage.id}
    >
      unread
    </article>
  );
}

describe("useMarkMessageReadOnView", () => {
  afterEach(() => {
    vi.useRealTimers();
    vi.unstubAllGlobals();
  });

  it("marks a message read after it stays visible for five seconds", () => {
    vi.useFakeTimers();

    const observe = vi.fn();
    const disconnect = vi.fn();
    let observerCallback: IntersectionObserverCallback = () => {};

    class TrackingIntersectionObserver {
      constructor(callback: IntersectionObserverCallback) {
        observerCallback = callback;
      }

      observe = observe;

      disconnect = disconnect;
    }

    vi.stubGlobal("IntersectionObserver", TrackingIntersectionObserver);

    const onMarkRead = vi.fn();
    render(<TestHarness onMarkRead={onMarkRead} />);

    observerCallback(
      [
        {
          isIntersecting: true,
          target: document.querySelector("[data-message-id='unread-1']") as Element,
        } as IntersectionObserverEntry,
      ],
      {} as IntersectionObserver,
    );

    expect(onMarkRead).not.toHaveBeenCalled();

    vi.advanceTimersByTime(5000);

    expect(onMarkRead).toHaveBeenCalledWith("unread-1");
  });
});
