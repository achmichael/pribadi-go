import { create } from 'zustand';

export interface ToolCall {
  function: {
    name: string;
    arguments: Record<string, any>;
  };
}

export interface StageEvent {
  stage: string;
  tool?: string;
  message?: string;
}

export interface Message {
  id: string;
  role: 'user' | 'assistant' | 'system';
  content: string;
  thinking?: string;
  toolCalls?: ToolCall[];
  createdAt: string;
  interrupted?: boolean;
  stages?: StageEvent[];
  file_job_id?: string;
  file_name?: string;
  file_mime_type?: string;
  file_size?: number;
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
  currentStage: StageEvent | null;
  abortController: AbortController | null;
  
  setSessions: (sessions: Session[]) => void;
  updateSession: (id: string, updates: Partial<Session>) => void;
  removeSession: (id: string) => void;
  setActiveSessionId: (id: string | null) => void;
  setMessages: (messages: Message[]) => void;
  addMessage: (message: Message) => void;
  appendStreamChunk: (chunk: string, toolCalls?: ToolCall[]) => void;
  appendThinking: (thinking: string) => void;
  setCurrentStage: (stage: StageEvent | null) => void;
  addStageToLastMessage: (stage: StageEvent) => void;
  markLastMessageInterrupted: () => void;
  setIsStreaming: (status: boolean) => void;
  setAbortController: (controller: AbortController | null) => void;
}

export const useChatStore = create<ChatState>((set) => ({
  sessions: [],
  activeSessionId: null,
  messages: [],
  isStreaming: false,
  currentStage: null,
  abortController: null,
  
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
    console.log('updated messages', updatedMsg);
    if (chunk) {
      updatedMsg.content = lastMsg.content + chunk;
    }
    if (toolCalls && toolCalls.length > 0) {
      updatedMsg.toolCalls = [...(lastMsg.toolCalls || []), ...toolCalls];
    }

    updatedMessages[updatedMessages.length - 1] = updatedMsg;

    console.log('final updated messages', updatedMessages)
    return { messages: updatedMessages };
  }),

  appendThinking: (thinking: string) => set((state) => {
    const lastMsg = state.messages[state.messages.length - 1];
    if (!lastMsg || lastMsg.role !== 'assistant') return state;

    const updatedMessages = [...state.messages];
    updatedMessages[updatedMessages.length - 1] = {
      ...lastMsg,
      thinking: (lastMsg.thinking || '') + thinking,
    };
    return { messages: updatedMessages };
  }),

  setCurrentStage: (stage) => set({ currentStage: stage }),

  addStageToLastMessage: (stage: StageEvent) => set((state) => {
    const lastMsg = state.messages[state.messages.length - 1];
    if (!lastMsg || lastMsg.role !== 'assistant') return state;

    const updatedMessages = [...state.messages];
    updatedMessages[updatedMessages.length - 1] = {
      ...lastMsg,
      stages: [...(lastMsg.stages || []), stage],
    };
    return { messages: updatedMessages, currentStage: stage };
  }),

  markLastMessageInterrupted: () => set((state) => {
    const lastMsg = state.messages[state.messages.length - 1];
    if (!lastMsg || lastMsg.role !== 'assistant') return state;

    const updatedMessages = [...state.messages];
    updatedMessages[updatedMessages.length - 1] = {
      ...lastMsg,
      interrupted: true,
    };
    return { messages: updatedMessages };
  }),
  
  setIsStreaming: (status) => set((state) => ({
    isStreaming: status,
    currentStage: status ? state.currentStage : null,
  })),

  setAbortController: (controller) => set({ abortController: controller }),
}));
