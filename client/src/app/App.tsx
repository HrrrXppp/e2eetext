import { OAUTH_CALLBACK_PATH } from "@/lib/api";
import { AuthCallback } from "@/app/AuthCallback";
import { ChatsPage } from "@/app/ChatsPage";
import { LandingPage } from "@/app/LandingPage";

export function App() {
  const { pathname } = window.location;

  if (pathname === OAUTH_CALLBACK_PATH) {
    return <AuthCallback />;
  }

  if (pathname === "/chats" || pathname.startsWith("/chats/")) {
    return <ChatsPage />;
  }

  return <LandingPage />;
}
