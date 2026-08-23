import { Sidebar } from "@/components/layout/sidebar";
import { ChatArea } from "@/components/layout/chat-area";

// Return at least one fake param to satisfy Next.js export constraints for dynamic routes
export function generateStaticParams() {
  return [{ sessionId: "default" }];
}

export default function ChatSessionPage() {
  return (
    <div className="flex h-screen w-full overflow-hidden bg-black text-white selection:bg-white/20">
      <Sidebar />
      <main className="flex-1 flex flex-col h-full relative border-l border-white/[0.02]">
        <ChatArea />
      </main>
    </div>
  );
}
