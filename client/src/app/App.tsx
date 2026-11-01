import { OAUTH_CALLBACK_PATH } from "@/lib/api";
import { AuthCallback } from "@/app/AuthCallback";
import { ChatsPage } from "@/app/ChatsPage";
import { EncryptionPage } from "@/app/EncryptionPage";
import { LandingPage } from "@/app/LandingPage";
import { PrivacyPolicyPage } from "@/app/PrivacyPolicyPage";
import { RoadmapPage } from "@/app/RoadmapPage";
import { TermsPage } from "@/app/TermsPage";

export function App() {
  const { pathname } = window.location;

  if (pathname === OAUTH_CALLBACK_PATH) {
    return <AuthCallback />;
  }

  if (pathname === "/chats" || pathname.startsWith("/chats/")) {
    return <ChatsPage />;
  }

  if (pathname === "/privacy") {
    return <PrivacyPolicyPage />;
  }

  if (pathname === "/terms") {
    return <TermsPage />;
  }

  if (pathname === "/roadmap") {
    return <RoadmapPage />;
  }

  if (pathname === "/encryption") {
    return <EncryptionPage />;
  }

  return <LandingPage />;
}
