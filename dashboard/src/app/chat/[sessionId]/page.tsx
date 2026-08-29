import { Sidebar } from "@/components/layout/sidebar";
import { ChatArea } from "@/components/layout/chat-area";

// Return at least one fake param to satisfy Next.js export constraints for dynamic routes
export function generateStaticParams() {
  return [{ sessionId: "default" }];
}

export default function ChatSessionPage() {
  return (
    <div className="flex h-[100dvh] w-full overflow-hidden bg-[#09090b] text-zinc-200 selection:bg-indigo-500/30 font-sans">
      <Sidebar />
      <main className="flex-1 flex flex-col h-full relative border-l border-zinc-800/40">
        <ChatArea />
      </main>
    </div>
  );
}
