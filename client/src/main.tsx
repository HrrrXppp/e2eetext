import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import { App } from "@/app/App";
import { DevInstanceBanner } from "@/components/layout/DevInstanceBanner";
import { AuthProvider } from "@/hooks/useAuth";
import "@/styles/index.css";

createRoot(document.getElementById("root")!).render(
  <StrictMode>
    <DevInstanceBanner />
    <AuthProvider>
      <App />
    </AuthProvider>
  </StrictMode>,
);
