import { Sidebar } from "@/components/layout/sidebar";
import { ChatArea } from "@/components/layout/chat-area";

export default function Home() {
  return (
    <div className="flex h-screen w-full overflow-hidden bg-black text-white selection:bg-white/20">
      <Sidebar />
      <main className="flex-1 flex flex-col h-full relative border-l border-white/[0.02]">
        <ChatArea />
      </main>
    </div>
  );
}
