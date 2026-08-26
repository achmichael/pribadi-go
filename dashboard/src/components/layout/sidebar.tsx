"use client";

import { useEffect, useState } from "react";
import { MessageSquare, Plus, Command, LogOut, MoreHorizontal, Pin, Edit2, Share, Trash, Archive, Settings } from "lucide-react";
import { Button } from "@/components/ui/button";
import { ScrollArea } from "@/components/ui/scroll-area";
import { useChatStore } from "@/store/chat";
import { fetchApi } from "@/lib/api";
import { useRouter } from "next/navigation";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
  DropdownMenuSeparator,
} from "@/components/ui/dropdown-menu";

import { Input } from "@/components/ui/input";

export function Sidebar() {
  const { sessions, setSessions, updateSession, removeSession, setActiveSessionId, activeSessionId, setMessages } =
    useChatStore();
  const router = useRouter();

  const [editingId, setEditingId] = useState<string | null>(null);
  const [editTitle, setEditTitle] = useState("");
  const [username, setUsername] = useState<string | null>(null);

  useEffect(() => {
    // get user info from JWT payload
    const token = localStorage.getItem("token");
    if (token) {
      try {
        const payloadBase64 = token.split('.')[1];
        if (payloadBase64) {
          const payloadJson = atob(payloadBase64);
          const payload = JSON.parse(payloadJson);
          if (payload && payload.username) {
            setUsername(payload.username);
          }
        }
      } catch (e) {
        console.error("Failed to parse token", e);
      }
    }
  }, []);

  useEffect(() => {
    fetchApi("/chat/sessions")
      .then((data) => {
        if (Array.isArray(data)) setSessions(data);
      })
      .catch((e) => {
        if (e.message !== 'Unauthorized') {
          console.error(e);
        }
      });
  }, []);

  const sortedSessions = [...sessions].sort((a, b) => {
    if (a.is_pinned && !b.is_pinned) return -1;
    if (!a.is_pinned && b.is_pinned) return 1;
    return new Date(b.updated_at || b.updatedAt).getTime() - new Date(a.updated_at || a.updatedAt).getTime();
  });

  const loadSession = async (id: string) => {
    setActiveSessionId(id);
    router.push(`/chat/${id}`);
  };

  const createNewSession = () => {
    setActiveSessionId(null);
    setMessages([]);
    router.push('/new');
  };

  const handleLogout = () => {
    localStorage.removeItem("token");
    router.push("/login");
  };

  const handleAction = async (e: React.MouseEvent, action: string, session: any) => {
    e.stopPropagation();
    try {
      if (action === 'delete') {
        await fetchApi(`/chat/sessions/${session.id}`, { method: 'DELETE' });
        removeSession(session.id);
        if (activeSessionId === session.id) createNewSession();
      } else if (action === 'pin') {
        const newPinnedState = !session.is_pinned;
        await fetchApi(`/chat/sessions/${session.id}`, {
          method: 'PUT',
          body: JSON.stringify({ is_pinned: newPinnedState }),
        });
        updateSession(session.id, { is_pinned: newPinnedState });
      } else if (action === 'rename') {
        setEditingId(session.id);
        setEditTitle(session.title || "New Thread");
      }
    } catch (err) {
      console.error(`Failed to ${action} session`, err);
    }
  };

  const handleRenameSubmit = async (e: React.KeyboardEvent | React.FocusEvent, id: string) => {
    if (e.type === 'keydown' && (e as React.KeyboardEvent).key !== 'Enter') return;
    
    try {
      if (editTitle.trim()) {
        await fetchApi(`/chat/sessions/${id}`, {
          method: 'PUT',
          body: JSON.stringify({ title: editTitle.trim() }),
        });
        updateSession(id, { title: editTitle.trim() });
      }
    } catch (err) {
      console.error("Failed to rename session", err);
    } finally {
      setEditingId(null);
    }
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
          {sortedSessions.length === 0 ? (
            <div className="px-3 py-4 text-xs text-zinc-600 font-medium">No previous threads.</div>
          ) : (
            sortedSessions.map((s) => {
              // Ensure we check routing/store active state correctly
              const isActive = activeSessionId === s.id;
              
              return (
              <div key={s.id} className="group relative">
                <button
                  onClick={() => loadSession(s.id)}
                  className={`w-full flex items-center gap-3 px-3 py-2 rounded-lg text-sm transition-colors text-left pr-16 ${
                    isActive
                      ? "bg-zinc-800/60 text-zinc-200 font-medium"
                      : "text-zinc-400 hover:bg-zinc-900/50 hover:text-zinc-300"
                  }`}
                >
                  <MessageSquare
                    className={`h-4 w-4 shrink-0 ${isActive ? "text-zinc-300" : "text-zinc-600"}`}
                  />
                  {editingId === s.id ? (
                    <Input
                      autoFocus
                      value={editTitle}
                      onChange={(e) => setEditTitle(e.target.value)}
                      onBlur={(e) => handleRenameSubmit(e, s.id)}
                      onKeyDown={(e) => handleRenameSubmit(e, s.id)}
                      className="h-6 text-sm bg-zinc-900 border-zinc-700 focus-visible:ring-1 focus-visible:ring-zinc-500 px-1"
                      onClick={(e) => e.stopPropagation()}
                    />
                  ) : (
                    <span className="truncate">{s.title || "New Thread"}</span>
                  )}
                </button>
                
                {editingId !== s.id && (
                  <div className="absolute right-2 top-1/2 -translate-y-1/2 flex items-center gap-1 opacity-0 group-hover:opacity-100 transition-opacity">
                    <button
                      onClick={(e) => handleAction(e, 'pin', s)}
                      className={`p-1 rounded hover:bg-zinc-700/50 ${s.is_pinned ? 'text-white' : 'text-zinc-500 hover:text-zinc-300'}`}
                    >
                      <Pin className="h-3.5 w-3.5" />
                    </button>
                    
                    <DropdownMenu>
                      <DropdownMenuTrigger asChild>
                        <button
                          onClick={(e) => e.stopPropagation()}
                          className="p-1 rounded text-zinc-500 hover:text-zinc-300 hover:bg-zinc-700/50"
                        >
                          <MoreHorizontal className="h-3.5 w-3.5" />
                        </button>
                      </DropdownMenuTrigger>
                      <DropdownMenuContent align="end" className="w-48 bg-zinc-950 border-white/10 text-zinc-300">
                        <DropdownMenuItem onClick={(e) => handleAction(e as any, 'rename', s)} className="gap-2 cursor-pointer focus:bg-zinc-800 focus:text-white">
                          <Edit2 className="h-4 w-4" /> Rename
                        </DropdownMenuItem>
                        <DropdownMenuItem onClick={(e) => handleAction(e as any, 'share', s)} className="gap-2 cursor-pointer focus:bg-zinc-800 focus:text-white">
                          <Share className="h-4 w-4" /> Share
                        </DropdownMenuItem>
                        <DropdownMenuItem onClick={(e) => handleAction(e as any, 'archive', s)} className="gap-2 cursor-pointer focus:bg-zinc-800 focus:text-white">
                          <Archive className="h-4 w-4" /> Archive
                        </DropdownMenuItem>
                        <DropdownMenuSeparator className="bg-white/10" />
                        <DropdownMenuItem onClick={(e) => handleAction(e as any, 'delete', s)} className="gap-2 cursor-pointer text-red-400 focus:bg-red-500/10 focus:text-red-300">
                          <Trash className="h-4 w-4" /> Delete
                        </DropdownMenuItem>
                      </DropdownMenuContent>
                    </DropdownMenu>
                  </div>
                )}
              </div>
            );
          })
          )}
        </div>
      </ScrollArea>

      <div className="p-3 mt-auto border-t border-zinc-800/50 space-y-2">
        <Button
          variant="ghost"
          className="w-full justify-start text-zinc-400 hover:text-zinc-200 hover:bg-zinc-900/50"
          onClick={() => router.push('/settings')}
        >
          <Settings className="h-4 w-4 mr-2" />
          {username || "Settings"}
        </Button>
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
