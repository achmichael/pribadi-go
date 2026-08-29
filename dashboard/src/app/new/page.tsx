import { Sidebar } from "@/components/layout/sidebar";
import { ChatArea } from "@/components/layout/chat-area";

export default function NewChatPage() {
  return (
    <div className="flex h-[100dvh] w-full overflow-hidden bg-[#09090b] text-zinc-200 selection:bg-indigo-500/30 font-sans">
      <Sidebar />
      <main className="flex-1 flex flex-col h-full relative border-l border-zinc-800/40">
        <ChatArea />
      </main>
    </div>
  );
}
