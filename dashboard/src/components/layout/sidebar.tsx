"use client";

import { useEffect } from "react";
import { MessageSquare, Plus, Command, LogOut } from "lucide-react";
import { Button } from "@/components/ui/button";
import { ScrollArea } from "@/components/ui/scroll-area";
import { useChatStore } from "@/store/chat";
import { fetchApi } from "@/lib/api";
import { useRouter } from "next/navigation";

import { SettingsModal } from "@/components/chat/settings-modal";

export function Sidebar() {
  const { sessions, setSessions, setActiveSessionId, activeSessionId, setMessages } = useChatStore();
  const router = useRouter();

  useEffect(() => {
    fetchApi('/chat/sessions')
      .then(data => {
        if (Array.isArray(data)) setSessions(data);
      })
      .catch(console.error);
  }, []);

  const loadSession = async (id: string) => {
    setActiveSessionId(id);
    try {
      const history = await fetchApi(`/chat/sessions/${id}/history`);
      if (Array.isArray(history)) {
        setMessages(history.map((m: any) => ({
          id: m.id,
          role: m.role,
          content: m.content,
          createdAt: m.created_at || new Date().toISOString()
        })));
      }
    } catch (e) {
      console.error("Failed to load history", e);
    }
  };

  const createNewSession = async () => {
    try {
      const sess = await fetchApi('/chat/sessions', {
        method: 'POST',
        body: JSON.stringify({ title: 'New Chat' })
      });
      setSessions([sess, ...sessions]);
      setActiveSessionId(sess.id);
      setMessages([]);
    } catch (e) {
      console.error("Failed to create session", e);
    }
  };

  const handleLogout = () => {
    localStorage.removeItem("token");
    router.push("/login");
  };

  return (
    <div className="w-64 border-r border-zinc-800/50 bg-[#09090b] flex flex-col h-full shrink-0 relative z-20">
      <div className="h-16 flex items-center justify-between px-4 border-b border-zinc-800/50">
        <div className="flex items-center gap-2 text-zinc-200 font-medium tracking-tight">
          <div className="h-6 w-6 rounded bg-white text-black flex items-center justify-center">
            <Command className="h-3 w-3" />
          </div>
          pribadi-go
        </div>
      </div>
      
      <div className="p-3">
        <Button 
          onClick={createNewSession}
          className="w-full justify-start gap-2 bg-zinc-900 hover:bg-zinc-800 text-zinc-300 border border-zinc-800/80 shadow-sm" 
          variant="outline"
        >
          <Plus className="h-4 w-4" />
          New Thread
        </Button>
      </div>

      <ScrollArea className="flex-1 px-2">
        <div className="space-y-0.5 mt-2">
          {sessions.length === 0 ? (
            <div className="px-3 py-4 text-xs text-zinc-600 font-medium">No previous threads.</div>
          ) : (
            sessions.map((s) => (
              <button
                key={s.id}
                onClick={() => loadSession(s.id)}
                className={`w-full flex items-center gap-3 px-3 py-2 rounded-lg text-sm transition-colors text-left ${
                  activeSessionId === s.id 
                    ? 'bg-zinc-800/60 text-zinc-200 font-medium' 
                    : 'text-zinc-400 hover:bg-zinc-900/50 hover:text-zinc-300'
                }`}
              >
                <MessageSquare className={`h-4 w-4 shrink-0 ${activeSessionId === s.id ? 'text-zinc-300' : 'text-zinc-600'}`} />
                <span className="truncate">{s.title || "New Thread"}</span>
              </button>
            ))
          )}
        </div>
      </ScrollArea>

      <div className="p-3 mt-auto border-t border-zinc-800/50 space-y-2">
        <SettingsModal />
        <Button 
          variant="ghost" 
          className="w-full justify-start text-zinc-400 hover:text-zinc-200 hover:bg-zinc-900/50"
          onClick={handleLogout}
        >
          <LogOut className="h-4 w-4 mr-2" />
          Logout
        </Button>
      </div>
    </div>
  );
}
