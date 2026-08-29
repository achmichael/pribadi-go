"use client";

import { useEffect, useState, useRef, useCallback } from "react";
import { MessageSquare, Plus, Terminal, LogOut, MoreHorizontal, Pin, Edit2, Share, Trash, Archive, Settings, PanelLeftClose, PanelLeftOpen, GripVertical } from "lucide-react";
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

  // Resize and Collapse State
  const [isCollapsed, setIsCollapsed] = useState(false);
  const [width, setWidth] = useState(260); // Default width 260px
  const [isResizing, setIsResizing] = useState(false);
  const minWidth = 200;
  const maxWidth = 480;

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
    return new Date(b.updated_at || b.updatedAt).getTime() - new Date(a.updated_at || a.updatedAt).getTime();
  });

  const pinnedSessions = sortedSessions.filter(s => s.is_pinned);
  const unpinnedSessions = sortedSessions.filter(s => !s.is_pinned);

  const today = new Date();
  today.setHours(0, 0, 0, 0);
  
  const sevenDaysAgo = new Date(today);
  sevenDaysAgo.setDate(today.getDate() - 7);

  const todaySessions = unpinnedSessions.filter(s => {
    const d = new Date(s.updated_at || s.updatedAt);
    d.setHours(0, 0, 0, 0);
    return d.getTime() >= today.getTime();
  });

  const previous7DaysSessions = unpinnedSessions.filter(s => {
    const d = new Date(s.updated_at || s.updatedAt);
    d.setHours(0, 0, 0, 0);
    return d.getTime() >= sevenDaysAgo.getTime() && d.getTime() < today.getTime();
  });

  const olderSessions = unpinnedSessions.filter(s => {
    const d = new Date(s.updated_at || s.updatedAt);
    d.setHours(0, 0, 0, 0);
    return d.getTime() < sevenDaysAgo.getTime();
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

  // Retrieve saved width and collapse state from localStorage
  useEffect(() => {
    const savedWidth = localStorage.getItem("sidebarWidth");
    if (savedWidth) {
      setWidth(parseInt(savedWidth, 10));
    }
    const savedCollapsed = localStorage.getItem("sidebarCollapsed");
    if (savedCollapsed) {
      setIsCollapsed(savedCollapsed === "true");
    }
  }, []);

  const handleResize = useCallback((e: MouseEvent) => {
    if (!isResizing) return;
    const newWidth = Math.min(Math.max(e.clientX, minWidth), maxWidth);
    setWidth(newWidth);
    localStorage.setItem("sidebarWidth", newWidth.toString());
  }, [isResizing]);

  const stopResizing = useCallback(() => {
    setIsResizing(false);
    document.body.style.cursor = 'default';
  }, []);

  useEffect(() => {
    if (isResizing) {
      window.addEventListener("mousemove", handleResize);
      window.addEventListener("mouseup", stopResizing);
      document.body.style.cursor = 'col-resize';
    } else {
      window.removeEventListener("mousemove", handleResize);
      window.removeEventListener("mouseup", stopResizing);
      document.body.style.cursor = 'default';
    }
    return () => {
      window.removeEventListener("mousemove", handleResize);
      window.removeEventListener("mouseup", stopResizing);
      document.body.style.cursor = 'default';
    };
  }, [isResizing, handleResize, stopResizing]);

  const toggleCollapse = () => {
    const newVal = !isCollapsed;
    setIsCollapsed(newVal);
    localStorage.setItem("sidebarCollapsed", newVal.toString());
  };

  const renderSessionItem = (s: any) => {
    const isActive = activeSessionId === s.id;
    return (
      <div key={s.id} className="group relative">
        <button
          onClick={() => loadSession(s.id)}
          className={`w-full flex items-center gap-2.5 px-3 py-2 rounded-md text-[13px] transition-all text-left pr-12 ${
            isActive
              ? "bg-zinc-800/80 text-zinc-100 font-medium shadow-sm border border-zinc-700/50"
              : "text-zinc-400 hover:bg-zinc-900/60 hover:text-zinc-200 border border-transparent"
          }`}
        >
          <MessageSquare
            className={`h-3.5 w-3.5 shrink-0 ${isActive ? "text-indigo-400" : "text-zinc-600"}`}
          />
          {editingId === s.id ? (
            <Input
              autoFocus
              value={editTitle}
              onChange={(e) => setEditTitle(e.target.value)}
              onBlur={(e) => handleRenameSubmit(e, s.id)}
              onKeyDown={(e) => handleRenameSubmit(e, s.id)}
              className="h-6 text-[13px] bg-zinc-900 border-indigo-500/50 focus-visible:ring-1 focus-visible:ring-indigo-500/50 px-1.5 rounded-sm"
              onClick={(e) => e.stopPropagation()}
            />
          ) : (
            <span className="truncate">{s.title || "New Thread"}</span>
          )}
        </button>
        
        {editingId !== s.id && (
          <div className="absolute right-1.5 top-1/2 -translate-y-1/2 flex items-center gap-0.5 opacity-0 group-hover:opacity-100 transition-opacity">
            <button
              onClick={(e) => handleAction(e, 'pin', s)}
              className={`p-1.5 rounded-md hover:bg-zinc-800 transition-colors ${s.is_pinned ? 'text-indigo-400' : 'text-zinc-500 hover:text-zinc-300'}`}
            >
              <Pin className="h-3 w-3" />
            </button>
            
            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <button
                  onClick={(e) => e.stopPropagation()}
                  className="p-1.5 rounded-md text-zinc-500 hover:text-zinc-300 hover:bg-zinc-800 transition-colors"
                >
                  <MoreHorizontal className="h-3 w-3" />
                </button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end" className="w-40 bg-zinc-950/95 backdrop-blur-xl border-zinc-800/80 text-zinc-300 shadow-xl rounded-lg text-xs">
                <DropdownMenuItem onClick={(e) => handleAction(e as any, 'rename', s)} className="gap-2 cursor-pointer focus:bg-zinc-800 focus:text-zinc-100 py-1.5">
                  <Edit2 className="h-3.5 w-3.5" /> Rename
                </DropdownMenuItem>
                <DropdownMenuItem onClick={(e) => handleAction(e as any, 'share', s)} className="gap-2 cursor-pointer focus:bg-zinc-800 focus:text-zinc-100 py-1.5">
                  <Share className="h-3.5 w-3.5" /> Share
                </DropdownMenuItem>
                <DropdownMenuItem onClick={(e) => handleAction(e as any, 'archive', s)} className="gap-2 cursor-pointer focus:bg-zinc-800 focus:text-zinc-100 py-1.5">
                  <Archive className="h-3.5 w-3.5" /> Archive
                </DropdownMenuItem>
                <DropdownMenuSeparator className="bg-zinc-800/60" />
                <DropdownMenuItem onClick={(e) => handleAction(e as any, 'delete', s)} className="gap-2 cursor-pointer text-red-400/90 focus:bg-red-500/10 focus:text-red-400 py-1.5">
                  <Trash className="h-3.5 w-3.5" /> Delete
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
          </div>
        )}
      </div>
    );
  };

  if (isCollapsed) {
    return (
      <div className="w-14 border-r border-zinc-800/40 bg-[#09090b] flex flex-col h-full shrink-0 items-center py-4 relative z-20 transition-all duration-300">
        <button 
          onClick={toggleCollapse} 
          className="p-2 mb-4 rounded-md text-zinc-500 hover:text-zinc-200 hover:bg-zinc-900 transition-colors"
          title="Expand Sidebar"
        >
          <PanelLeftOpen className="h-4 w-4" />
        </button>
        
        <button 
          onClick={createNewSession}
          className="p-2 mb-4 rounded-md bg-zinc-900 text-zinc-300 hover:bg-zinc-800 transition-colors border border-zinc-800/80"
          title="New Thread"
        >
          <Plus className="h-4 w-4" />
        </button>

        <div className="flex-1 overflow-hidden" />

        <div className="space-y-3">
          <button 
            onClick={() => router.push('/settings')}
            className="p-2 rounded-md text-zinc-500 hover:text-zinc-200 hover:bg-zinc-900 transition-colors block"
            title="Settings"
          >
            <Settings className="h-4 w-4" />
          </button>
          
          <button 
            onClick={handleLogout}
            className="p-2 rounded-md text-zinc-500 hover:text-red-400 hover:bg-red-500/10 transition-colors block"
            title="Logout"
          >
            <LogOut className="h-4 w-4" />
          </button>
        </div>
      </div>
    );
  }

  return (
    <div 
      className="border-r border-zinc-800/40 bg-[#09090b] flex flex-col h-full pt-4 shrink-0 relative z-20 group/sidebar transition-all duration-300 ease-in-out overflow-y-auto"
      style={{ width: `${width}px` }}
    >
      {/* Resizer Handle */}
      <div 
        className="absolute right-0 top-0 w-1.5 h-full cursor-col-resize opacity-0 group-hover/sidebar:opacity-100 hover:bg-indigo-500/10 z-30 flex items-center justify-center transition-colors"
        onMouseDown={(e) => {
          e.preventDefault();
          setIsResizing(true);
        }}
      >
        <div className="h-8 w-1 rounded-full bg-zinc-700/50" />
      </div>

      <div className="h-12 flex items-center justify-between px-4 mb-2">
        <div className="flex items-center gap-2.5 text-zinc-100 font-medium tracking-tight overflow-hidden">
          <div className="h-6 w-6 shrink-0 rounded bg-indigo-500/10 text-indigo-400 flex items-center justify-center border border-indigo-500/20">
            <Terminal className="h-3 w-3" />
          </div>
          <span className="truncate text-sm font-sans tracking-wide">pribadi-go</span>
        </div>
        <button 
          onClick={toggleCollapse} 
          className="p-1.5 shrink-0 rounded-md text-zinc-500 hover:text-zinc-200 hover:bg-zinc-900 transition-colors"
          title="Collapse Sidebar"
        >
          <PanelLeftClose className="h-4 w-4" />
        </button>
      </div>

      <div className="px-3 pb-2 border-b border-zinc-800/40">
        <Button
          onClick={createNewSession}
          className="w-full justify-start gap-2 bg-zinc-900 hover:bg-zinc-800 text-zinc-200 border border-zinc-800/80 shadow-sm h-9"
          variant="outline"
        >
          <Plus className="h-4 w-4 text-zinc-400" />
          <span className="text-[13px]">New Thread</span>
        </Button>
      </div>

      <ScrollArea className="flex-1 px-2 pt-2">
        <div className="space-y-4 pb-4">
          {sortedSessions.length === 0 ? (
            <div className="px-3 py-4 text-xs text-zinc-600 font-medium">No previous threads.</div>
          ) : (
            <>
              {pinnedSessions.length > 0 && (
                <div className="space-y-0.5">
                  <div className="px-3 py-1.5 text-[10px] font-mono uppercase tracking-wider text-zinc-500 mb-1">Pinned</div>
                  {pinnedSessions.map(renderSessionItem)}
                </div>
              )}
              {todaySessions.length > 0 && (
                <div className="space-y-0.5 mt-4">
                  <div className="px-3 py-1.5 text-[10px] font-mono uppercase tracking-wider text-zinc-500 mb-1">Today</div>
                  {todaySessions.map(renderSessionItem)}
                </div>
              )}
              {previous7DaysSessions.length > 0 && (
                <div className="space-y-0.5 mt-4">
                  <div className="px-3 py-1.5 text-[10px] font-mono uppercase tracking-wider text-zinc-500 mb-1">Previous 7 Days</div>
                  {previous7DaysSessions.map(renderSessionItem)}
                </div>
              )}
              {olderSessions.length > 0 && (
                <div className="space-y-0.5 mt-4">
                  <div className="px-3 py-1.5 text-[10px] font-mono uppercase tracking-wider text-zinc-500 mb-1">Older</div>
                  {olderSessions.map(renderSessionItem)}
                </div>
              )}
            </>
          )}
        </div>
      </ScrollArea>

      <div className="p-3 mt-auto border-t border-zinc-800/40 space-y-1">
        <Button
          variant="ghost"
          className="w-full justify-start text-zinc-400 hover:text-zinc-200 hover:bg-zinc-900 h-9"
          onClick={() => router.push('/settings')}
        >
          <Settings className="h-4 w-4 mr-2" />
          <span className="text-[13px]">{username || "Settings"}</span>
        </Button>
        <Button
          variant="ghost"
          className="w-full justify-start text-zinc-400 hover:text-red-400 hover:bg-red-500/10 h-9"
          onClick={handleLogout}
        >
          <LogOut className="h-4 w-4 mr-2" />
          <span className="text-[13px]">Logout</span>
        </Button>
      </div>
    </div>
  );
}
