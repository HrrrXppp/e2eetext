import { type RefObject, useCallback, useEffect, useRef } from "react";
import type { Message } from "@/lib/messages";

const READ_DELAY_MS = 5000;

export function useMarkMessageReadOnView(
  chatId: string | null,
  messages: Message[],
  currentUserId: string,
  scrollRootRef: RefObject<HTMLElement | null>,
  onMarkRead: (messageId: string) => void,
) {
  const messageElementsRef = useRef(new Map<string, HTMLElement>());
  const timersRef = useRef(new Map<string, number>());
  const markedRef = useRef(new Set<string>());
  const onMarkReadRef = useRef(onMarkRead);
  onMarkReadRef.current = onMarkRead;

  useEffect(() => {
    markedRef.current.clear();
    for (const timer of timersRef.current.values()) {
      clearTimeout(timer);
    }
    timersRef.current.clear();
  }, [chatId]);

  useEffect(() => {
    const root = scrollRootRef.current;
    if (!root || !chatId) {
      return;
    }

    const unreadMessages = messages.filter(
      (message) =>
        message.unread && message.userId !== currentUserId && !markedRef.current.has(message.id),
    );

    const observer = new IntersectionObserver(
      (entries) => {
        for (const entry of entries) {
          const messageId = entry.target.getAttribute("data-message-id");
          if (!messageId) {
            continue;
          }

          if (entry.isIntersecting) {
            if (timersRef.current.has(messageId) || markedRef.current.has(messageId)) {
              continue;
            }

            const timer = window.setTimeout(() => {
              timersRef.current.delete(messageId);
              markedRef.current.add(messageId);
              onMarkReadRef.current(messageId);
            }, READ_DELAY_MS);
            timersRef.current.set(messageId, timer);
            continue;
          }

          const timer = timersRef.current.get(messageId);
          if (timer) {
            clearTimeout(timer);
            timersRef.current.delete(messageId);
          }
        }
      },
      { root, threshold: 0.5 },
    );

    for (const message of unreadMessages) {
      const element = messageElementsRef.current.get(message.id);
      if (element) {
        observer.observe(element);
      }
    }

    return () => {
      observer.disconnect();
      for (const timer of timersRef.current.values()) {
        clearTimeout(timer);
      }
      timersRef.current.clear();
    };
  }, [chatId, currentUserId, messages, scrollRootRef]);

  return useCallback((messageId: string, element: HTMLElement | null) => {
    if (element) {
      messageElementsRef.current.set(messageId, element);
      return;
    }
    messageElementsRef.current.delete(messageId);
  }, []);
}
