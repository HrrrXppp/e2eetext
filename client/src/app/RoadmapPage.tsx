import { Footer } from "@/components/layout/Footer";
import { SiteHeader } from "@/components/layout/SiteHeader";

type RoadmapStatus = "in-progress" | "planned";

const statusLabels: Record<RoadmapStatus, string> = {
  "in-progress": "In progress",
  planned: "Planned",
};

export const roadmap: ReadonlyArray<{
  title: string;
  quarter: string;
  status: RoadmapStatus;
  description: string;
}> = [
  {
    title: "Base improvement",
    quarter: "Q4 2026",
    status: "in-progress",
    description:
      "Polish of the core product: reliability, UX fixes, and hardening of the existing end-to-end encrypted text chat, sign-in and notifications.",
  },
  {
    title: "Separated services with custom LB for good scale",
    quarter: "Q4 2026",
    status: "planned",
    description:
      "Split the backend into independently deployable services behind a custom load balancer so chat, auth and delivery can scale separately.",
  },
  {
    title: "Audio calls",
    quarter: "Q1 2027",
    status: "planned",
    description:
      "End-to-end encrypted one-to-one voice calls, using the same post-quantum key agreement (ML-KEM-1024) as chats.",
  },
  {
    title: "Video calls",
    quarter: "Q1 2027",
    status: "planned",
    description: "End-to-end encrypted video calls built on top of the audio call stack.",
  },
];

export function RoadmapPage() {
  return (
    <>
      <SiteHeader />
      <main className="legal roadmap">
        <h1 className="legal__title">Roadmap</h1>
        <p className="legal__updated">What we are building next.</p>
        <ol className="roadmap__list">
          {roadmap.map((item) => (
            <li key={item.title} className="roadmap__item">
              <div className="roadmap__meta">
                <span className="roadmap__quarter">{item.quarter}</span>
                <span className={`roadmap__status roadmap__status--${item.status}`}>
                  {statusLabels[item.status]}
                </span>
              </div>
              <h2 className="roadmap__title">{item.title}</h2>
              <p className="roadmap__description">{item.description}</p>
            </li>
          ))}
        </ol>
      </main>
      <Footer />
    </>
  );
}
