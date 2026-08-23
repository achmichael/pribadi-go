import { create } from 'zustand';

export interface ToolCall {
  function: {
    name: string;
    arguments: Record<string, any>;
  };
}

export interface Message {
  id: string;
  role: 'user' | 'assistant' | 'system';
  content: string;
  toolCalls?: ToolCall[];
  createdAt: string;
}

export interface Session {
  id: string;
  title: string;
  updatedAt: string;
  is_pinned?: boolean;
}

interface ChatState {
  sessions: Session[];
  activeSessionId: string | null;
  messages: Message[];
  isStreaming: boolean;
  
  setSessions: (sessions: Session[]) => void;
  updateSession: (id: string, updates: Partial<Session>) => void;
  removeSession: (id: string) => void;
  setActiveSessionId: (id: string | null) => void;
  setMessages: (messages: Message[]) => void;
  addMessage: (message: Message) => void;
  appendStreamChunk: (chunk: string) => void;
  setIsStreaming: (status: boolean) => void;
}

export const useChatStore = create<ChatState>((set) => ({
  sessions: [],
  activeSessionId: null,
  messages: [],
  isStreaming: false,
  
  setSessions: (sessions) => set({ sessions }),
  updateSession: (id, updates) => set((state) => ({
    sessions: state.sessions.map((s) => (s.id === id ? { ...s, ...updates } : s)),
  })),
  removeSession: (id) => set((state) => ({
    sessions: state.sessions.filter((s) => s.id !== id),
    activeSessionId: state.activeSessionId === id ? null : state.activeSessionId,
  })),
  setActiveSessionId: (id) => set({ activeSessionId: id }),
  setMessages: (messages) => set({ messages }),
  
  addMessage: (message) => set((state) => ({ 
    messages: [...state.messages, message] 
  })),
  
  appendStreamChunk: (chunk: string, toolCalls?: ToolCall[]) => set((state) => {
    const lastMsg = state.messages[state.messages.length - 1];
    if (!lastMsg || lastMsg.role !== 'assistant') return state;
    
    const updatedMessages = [...state.messages];
    
    const updatedMsg = { ...lastMsg };
    if (chunk) {
      updatedMsg.content = lastMsg.content + chunk;
    }
    if (toolCalls && toolCalls.length > 0) {
      updatedMsg.toolCalls = [...(lastMsg.toolCalls || []), ...toolCalls];
    }

    updatedMessages[updatedMessages.length - 1] = updatedMsg;
    
    return { messages: updatedMessages };
  }),
  
  setIsStreaming: (status) => set({ isStreaming: status })
}));
