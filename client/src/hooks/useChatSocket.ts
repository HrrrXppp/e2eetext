import { useEffect, useRef, useState } from "react";
import { getWebSocketIdToken, TOKEN_REFRESHED_EVENT } from "@/lib/auth";
import {
  buildChatWebSocketURL,
  buildHeartbeatMessage,
  HEARTBEAT_INTERVAL_MS,
  isHeartbeatAckEvent,
  parseChatAddedEvent,
  parseChatUnreadEvent,
  type ChatAddedEvent,
  type ChatUnreadEvent,
} from "@/lib/ws";

export type ChatSocketState = "idle" | "connecting" | "open" | "closed";

const INITIAL_RETRY_MS = 1_000;
const MAX_RETRY_MS = 30_000;
const heartbeatMessage = buildHeartbeatMessage();

export function useChatSocket(
  enabled: boolean,
  onChatUnread?: (event: ChatUnreadEvent) => void,
  onChatAdded?: (event: ChatAddedEvent) => void,
): ChatSocketState {
  const [state, setState] = useState<ChatSocketState>("idle");
  const onChatUnreadRef = useRef(onChatUnread);
  const onChatAddedRef = useRef(onChatAdded);

  useEffect(() => {
    onChatUnreadRef.current = onChatUnread;
  }, [onChatUnread]);

  useEffect(() => {
    onChatAddedRef.current = onChatAdded;
  }, [onChatAdded]);

  useEffect(() => {
    if (!enabled) {
      setState("idle");
      return;
    }

    let active = true;
    let socket: WebSocket | null = null;
    let retryTimer: number | undefined;
    let heartbeatTimer: number | undefined;
    let heartbeatSocket: WebSocket | null = null;
    let retryAttempt = 0;

    function clearRetryTimer() {
      if (retryTimer !== undefined) {
        window.clearTimeout(retryTimer);
        retryTimer = undefined;
      }
    }

    function clearHeartbeatTimer(forSocket?: WebSocket | null) {
      if (forSocket !== undefined && forSocket !== null && heartbeatSocket !== forSocket) {
        return;
      }

      if (heartbeatTimer !== undefined) {
        window.clearInterval(heartbeatTimer);
        heartbeatTimer = undefined;
      }
      heartbeatSocket = null;
    }

    function sendHeartbeat(forSocket: WebSocket) {
      if (forSocket.readyState !== WebSocket.OPEN) {
        return;
      }

      forSocket.send(heartbeatMessage);
    }

    function startHeartbeat(forSocket: WebSocket) {
      clearHeartbeatTimer();
      heartbeatSocket = forSocket;
      sendHeartbeat(forSocket);
      heartbeatTimer = window.setInterval(() => sendHeartbeat(forSocket), HEARTBEAT_INTERVAL_MS);
    }

    function detachSocketHandlers(connection: WebSocket) {
      connection.onopen = null;
      connection.onclose = null;
      connection.onerror = null;
      connection.onmessage = null;
    }

    function scheduleReconnect() {
      if (!active || !enabled) {
        return;
      }

      clearRetryTimer();
      const delay = Math.min(INITIAL_RETRY_MS * 2 ** retryAttempt, MAX_RETRY_MS);
      retryAttempt += 1;
      retryTimer = window.setTimeout(() => {
        connect();
      }, delay);
    }

    function connect() {
      clearRetryTimer();

      if (!active || !enabled) {
        setState("closed");
        return;
      }

      const currentToken = getWebSocketIdToken();
      if (!currentToken) {
        setState("closed");
        scheduleReconnect();
        return;
      }

      setState("connecting");
      const connection = new WebSocket(buildChatWebSocketURL(currentToken));
      socket = connection;

      connection.onopen = () => {
        if (!active || socket !== connection) {
          return;
        }
        retryAttempt = 0;
        setState("open");
        startHeartbeat(connection);
      };

      connection.onclose = (event) => {
        if (socket !== connection) {
          return;
        }
        clearHeartbeatTimer(connection);
        socket = null;
        if (!active) {
          return;
        }
        setState("closed");
        if (event.code !== 1000) {
          scheduleReconnect();
        }
      };

      connection.onerror = () => {
        if (active && socket === connection) {
          setState("closed");
        }
      };

      connection.onmessage = (message) => {
        if (socket !== connection) {
          return;
        }

        if (isHeartbeatAckEvent(message.data)) {
          return;
        }

        const unreadEvent = parseChatUnreadEvent(message.data);
        if (unreadEvent) {
          onChatUnreadRef.current?.(unreadEvent);
          return;
        }

        const addedEvent = parseChatAddedEvent(message.data);
        if (addedEvent) {
          onChatAddedRef.current?.(addedEvent);
        }
      };
    }

    function handleTokenRefresh() {
      const previous = socket;
      socket = null;
      if (previous) {
        detachSocketHandlers(previous);
        previous.close();
      }
      retryAttempt = 0;
      connect();
    }

    connect();
    window.addEventListener(TOKEN_REFRESHED_EVENT, handleTokenRefresh);

    return () => {
      active = false;
      clearRetryTimer();
      clearHeartbeatTimer();
      window.removeEventListener(TOKEN_REFRESHED_EVENT, handleTokenRefresh);
      if (socket) {
        detachSocketHandlers(socket);
        socket.close(1000);
        socket = null;
      }
    };
  }, [enabled]);

  return state;
}
