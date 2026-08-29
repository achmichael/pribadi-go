"use client";

import { useState, useRef, useEffect, memo } from "react";
import { useParams, useRouter } from "next/navigation";
import { Send, Square, FileText, Bot, Terminal, ShieldAlert, Zap, Globe, HardDrive } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Textarea } from "@/components/ui/textarea";
import { useChatStore } from "@/store/chat";
import { API_BASE, getAuthHeader, fetchApi } from "@/lib/api";
import ReactMarkdown from "react-markdown";
import remarkGfm from "remark-gfm";
import { ModelSwitcher } from "@/components/chat/model-switcher";
import { FileUpload, type AttachedFile } from "@/components/chat/file-upload";
import { ThinkingStages } from "@/components/chat/thinking-stages";
import { motion, AnimatePresence } from "framer-motion";

// Clean, functional brand tokens implemented as Tailwind classes
// Primary: bg-zinc-900
// Surface: bg-zinc-900/40 border-zinc-800/60
// Accent: bg-indigo-500 / text-indigo-400
// Alert/System: text-amber-500/80

const SUGGESTED_PROMPTS = [
  { icon: Terminal, text: "Parse latest system logs" },
  { icon: Globe, text: "Search web for recent API changes" },
  { icon: ShieldAlert, text: "Audit local project dependencies" },
];

async function uploadFile(file: File, signal?: AbortSignal): Promise<string | null> {
  const formData = new FormData();
  formData.append("file", file);

  const res = await fetch(`${API_BASE}/chat/upload`, {
    method: "POST",
    headers: {
      ...getAuthHeader(),
    },
    body: formData,
    signal,
  });

  if (!res.ok) return null;
  const job = await res.json();
  return job.id || null;
}

const ChatInput = memo(function ChatInput({
  onSubmit,
  onStop,
  isStreaming,
  attachedFile,
  setAttachedFile,
}: {
  onSubmit: (text: string) => void;
  onStop: () => void;
  isStreaming: boolean;
  attachedFile: AttachedFile | null;
  setAttachedFile: (file: AttachedFile | null) => void;
}) {
  const [input, setInput] = useState("");
  const canSubmit = input.trim() || attachedFile;
  const textareaRef = useRef<HTMLTextAreaElement>(null);

  useEffect(() => {
    if (textareaRef.current) {
      textareaRef.current.style.height = "auto";
      textareaRef.current.style.height = `${Math.min(textareaRef.current.scrollHeight, 200)}px`;
    }
  }, [input]);

  return (
    <div className="relative flex flex-col rounded-xl bg-zinc-950/80 border border-zinc-800/80 shadow-sm shadow-black/20 focus-within:border-indigo-500/30 focus-within:ring-1 focus-within:ring-indigo-500/10 transition-all backdrop-blur-xl">
      <Textarea
        ref={textareaRef}
        value={input}
        onChange={(e) => setInput(e.target.value)}
        onKeyDown={(e) => {
          if (e.key === "Enter" && !e.shiftKey) {
            e.preventDefault();
            onSubmit(input);
            setInput("");
            if (textareaRef.current) textareaRef.current.style.height = "auto";
          }
        }}
        placeholder="Type a command or question..."
        className="min-h-[52px] max-h-[200px] border-0 focus-visible:ring-0 resize-none bg-transparent py-3.5 px-4 text-sm font-sans placeholder:text-zinc-500/70 text-zinc-100"
        rows={1}
      />
      <div className="flex items-center justify-between px-3 pb-3 pt-1">
        <FileUpload
          attachedFile={attachedFile}
          onFileSelect={setAttachedFile}
          disabled={isStreaming}
        />
        {isStreaming ? (
          <Button
            onClick={onStop}
            size="icon"
            className="rounded-lg h-8 w-8 bg-zinc-800/80 text-zinc-400 hover:text-red-400 hover:bg-zinc-800 transition-colors border border-transparent hover:border-red-500/20"
          >
            <Square className="h-3.5 w-3.5 fill-current" />
          </Button>
        ) : (
          <Button
            onClick={() => {
              onSubmit(input);
              setInput("");
              if (textareaRef.current) textareaRef.current.style.height = "auto";
            }}
            disabled={!canSubmit}
            size="icon"
            className={`rounded-lg h-8 w-8 transition-colors ${canSubmit ? "bg-indigo-600 text-white hover:bg-indigo-500 shadow-sm" : "bg-zinc-900/50 text-zinc-600 border border-zinc-800/50"}`}
          >
            <Send className="h-3.5 w-3.5" />
          </Button>
        )}
      </div>
    </div>
  );
});

export function ChatArea() {
  const [model, setModel] = useState("local");
  const [attachedFile, setAttachedFile] = useState<AttachedFile | null>(null);
  const [isLoadingHistory, setIsLoadingHistory] = useState(false);
  const [historyError, setHistoryError] = useState<string | null>(null);
  const params = useParams();
  const router = useRouter();

  const {
    messages,
    addMessage,
    appendStreamChunk,
    appendThinking,
    setCurrentStage,
    addStageToLastMessage,
    markLastMessageInterrupted,
    setIsStreaming,
    isStreaming,
    currentStage,
    activeSessionId,
    setActiveSessionId,
    setMessages,
    setSessions,
    sessions,
    abortController,
    setAbortController,
  } = useChatStore();
  const scrollRef = useRef<HTMLDivElement>(null);

  // Sync route params with store state
  useEffect(() => {
    const routeSessionId = params?.sessionId as string | undefined;
    const currentState = useChatStore.getState();

    // If we're on /new, ensure state is clear
    if (!routeSessionId) {
      // Don't clear state if we just started streaming (e.g. transitioning from /new to /chat/[id])
      if (!currentState.isStreaming) {
        if (activeSessionId) setActiveSessionId(null);
        if (messages.length > 0) setMessages([]);
      }
      return;
    }

    // If route has sessionId but store doesn't match, update store and fetch
    if (routeSessionId && activeSessionId !== routeSessionId) {
      setActiveSessionId(routeSessionId);

      setIsLoadingHistory(true);
      setHistoryError(null);

      fetchApi(`/chat/sessions/${routeSessionId}/history`)
        .then((history) => {
          if (Array.isArray(history)) {
            setMessages(
              history.map((m: any) => ({
                id: m.id,
                role: m.role,
                content: m.content,
                createdAt: m.created_at || new Date().toISOString(),
                file_job_id: m.file_job_id,
                file_name: m.file_name,
                file_mime_type: m.file_mime_type,
                file_size: m.file_size,
              })),
            );
          }
        })
        .catch((e) => {
          if (e.message !== "Unauthorized") {
            console.error("Failed to load history", e);
            setHistoryError("Failed to load conversation history");
          }
        })
        .finally(() => {
          setIsLoadingHistory(false);
        });
    }
  }, [params?.sessionId, activeSessionId, setActiveSessionId, setMessages]);

  useEffect(() => {
    if (scrollRef.current) {
      scrollRef.current.scrollTop = scrollRef.current.scrollHeight;
    }
  }, [messages, isLoadingHistory]);

  const handleStop = () => {
    if (abortController) {
      abortController.abort();
      markLastMessageInterrupted();
      setIsStreaming(false);
      setAbortController(null);
      setCurrentStage(null);
    }
  };

  const handleSubmit = async (textToSubmit?: string) => {
    const text = textToSubmit || "";
    const hasFile = !!attachedFile;
    if ((!text.trim() && !hasFile) || isStreaming) return;

    const pendingFile = attachedFile;
    setAttachedFile(null);

    const controller = new AbortController();
    setAbortController(controller);

    const displayText = text.trim();

    const tempId = Date.now().toString();
    const newUserMessage: any = {
      id: tempId,
      role: "user",
      content: displayText,
      createdAt: new Date().toISOString(),
    };

    if (pendingFile) {
      // Add fake/local properties for optimistic UI if possible,
      // but file_name is enough for basic display
      newUserMessage.file_name = pendingFile.file.name;
      newUserMessage.file_size = pendingFile.file.size;
      newUserMessage.file_mime_type = pendingFile.file.type;

      // Store previewUrl internally in message if we want to render image preview
      // (This is a temporary hack for immediate display before refresh)
      if (pendingFile.previewUrl) {
        newUserMessage._localPreview = pendingFile.previewUrl;
      }
    }
    addMessage(newUserMessage);
    setIsStreaming(true);

    const asstId = (Date.now() + 1).toString();
    addMessage({ id: asstId, role: "assistant", content: "", createdAt: new Date().toISOString() });

    let currentSessionId = activeSessionId;
    if (!activeSessionId) {
      try {
        const res = await fetch(`${API_BASE}/chat/sessions`, {
          method: "POST",
          headers: {
            "Content-Type": "application/json",
            ...getAuthHeader(),
          },
          body: JSON.stringify({ message: text || pendingFile?.file.name || "New Chat" }),
          signal: controller.signal,
        });

        if (!res.ok) {
          if (res.status === 401 || res.status === 403) {
            if (typeof window !== "undefined") {
              localStorage.removeItem("token");
              if (window.location.pathname !== "/login") {
                window.location.href = "/login";
              }
            }
            throw new Error("Unauthorized");
          }
          throw new Error("Create chat session failed");
        }

        const result = await res.json();
        currentSessionId = result.id;
        setActiveSessionId(result.id);
        setSessions([...sessions, result]);

        // Use Next.js router to gracefully replace the URL
        router.replace(`/chat/${result.id}`);
      } catch (e) {
        if ((e as Error).name === "AbortError") {
          setIsStreaming(false);
          setAbortController(null);
          return;
        }
        throw e;
      }
    }

    try {
      let fileJobId = "";
      if (pendingFile) {
        fileJobId = (await uploadFile(pendingFile.file, controller.signal)) || "";
        // Don't revoke URL here since we might need it for optimistic UI
        // We can just rely on normal browser garbage collection or clean up on unmount
      }

      const res = await fetch(`${API_BASE}/chat/stream`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          ...getAuthHeader(),
        },
        body: JSON.stringify({
          message: text,
          session_id: currentSessionId,
          model: model,
          file_job_id: fileJobId,
        }),
        signal: controller.signal,
      });

      if (!res.ok) {
        if (res.status === 401 || res.status === 403) {
          if (typeof window !== "undefined") {
            localStorage.removeItem("token");
            if (window.location.pathname !== "/login") {
              window.location.href = "/login";
            }
          }
          throw new Error("Unauthorized");
        }
        throw new Error("Stream failed");
      }

      const reader = res.body?.getReader();
      const decoder = new TextDecoder();

      if (reader) {
        let buffer = "";
        while (true) {
          const { done, value } = await reader.read();
          if (done) break;

          buffer += decoder.decode(value, { stream: true });

          const lines = buffer.split("\n");
          buffer = lines.pop() || "";

          let eventType = "";
          for (const line of lines) {
            if (line.startsWith("event: ")) {
              eventType = line.replace("event: ", "").trim();
            } else if (line.startsWith("data: ")) {
              const dataStr = line.replace("data: ", "").trim();
              if (dataStr && dataStr !== "null") {
                const data = JSON.parse(dataStr);
                if (eventType === "token" && data.content) {
                  console.log("event type token")
                  appendStreamChunk(data.content, undefined);
                } else if (eventType === "tool" && Array.isArray(data)) {
                  console.log('event type tool')
                  appendStreamChunk("", data);
                } else if (eventType === "thinking" && data.content) {
                  console.log('event type thingking')
                  appendThinking(data.content);
                } else if (eventType === "stage") {
                  console.log('event type stage')
                  addStageToLastMessage(data);
                } else if (eventType === "interrupted") {
                  console.log('event type interrupted');
                  markLastMessageInterrupted();
                }
              }
            }
          }
        }
      }
    } catch (e) {
      if ((e as Error).name === "AbortError") {
        markLastMessageInterrupted();
      } else {
        console.error(e);
        appendStreamChunk("\n\n*Error: Failed to fetch response*", undefined);
      }
    } finally {
      setIsStreaming(false);
      setAbortController(null);
      setCurrentStage(null);
    }
  };

  return (
    <div className="flex flex-col h-full w-full mx-auto relative bg-[#09090b]">
      <div className="absolute top-0 w-full z-10 px-4 py-3 flex items-center justify-between border-b border-zinc-800/40 bg-[#09090b]/80 backdrop-blur-md">
        <ModelSwitcher model={model} setModel={setModel} />
        <div className="flex items-center gap-2">
          <span className="flex h-2 w-2 rounded-full bg-emerald-500/80 shadow-[0_0_8px_rgba(16,185,129,0.4)]"></span>
          <span className="text-[10px] font-mono text-zinc-500 uppercase tracking-widest">Local Engine</span>
        </div>
      </div>

      <div className="flex-1 overflow-y-auto px-4 mt-[52px]" ref={scrollRef}>
        <div className="max-w-3xl mx-auto pb-32 pt-8">
          {isLoadingHistory ? (
            <div className="flex justify-center items-center h-full mt-32">
              <div className="animate-spin h-5 w-5 border-2 border-indigo-500/30 border-t-indigo-500 rounded-full" />
            </div>
          ) : historyError ? (
            <div className="flex justify-center items-center h-full mt-32 text-amber-500/80 text-sm font-mono">
              [Error] {historyError}
            </div>
          ) : messages.length === 0 ? (
            <div className="flex flex-col items-center mt-16 md:mt-24 max-w-xl mx-auto w-full text-center">
              <motion.div
                initial={{ opacity: 0, y: 10 }}
                animate={{ opacity: 1, y: 0 }}
                transition={{ duration: 0.4 }}
                className="mb-8"
              >
                <div className="h-12 w-12 rounded-xl bg-zinc-900 border border-zinc-800 flex items-center justify-center mx-auto mb-6 shadow-sm">
                  <Terminal className="h-5 w-5 text-indigo-400" />
                </div>
                <h2 className="text-xl font-medium text-zinc-200 mb-2 font-sans tracking-tight">System Ready</h2>
                <p className="text-zinc-500 text-sm font-sans leading-relaxed">
                  Local intelligence initialized. Ready for execution.
                </p>
              </motion.div>

              <motion.div 
                initial={{ opacity: 0, y: 10 }}
                animate={{ opacity: 1, y: 0 }}
                transition={{ duration: 0.4, delay: 0.1 }}
                className="grid grid-cols-1 sm:grid-cols-2 gap-3 w-full"
              >
                {SUGGESTED_PROMPTS.map((prompt, i) => (
                  <button
                    key={i}
                    onClick={() => handleSubmit(prompt.text)}
                    className="flex items-center gap-3 p-3 rounded-lg border border-zinc-800/60 bg-zinc-900/30 hover:bg-zinc-800/50 hover:border-zinc-700 transition-all text-left group"
                  >
                    <prompt.icon className="h-4 w-4 text-zinc-500 group-hover:text-indigo-400 transition-colors shrink-0" />
                    <span className="text-xs font-medium text-zinc-400 group-hover:text-zinc-200 transition-colors leading-tight">{prompt.text}</span>
                  </button>
                ))}
              </motion.div>
            </div>
          ) : (
            <AnimatePresence initial={false}>
              {messages.map((m, idx) => {
                const isLastAssistant =
                  m.role === "assistant" && m.id === messages[messages.length - 1]?.id;
                const isUser = m.role === "user";

                return (
                  <motion.div
                    key={m.id}
                    initial={{ opacity: 0, y: 5 }}
                    animate={{ opacity: 1, y: 0 }}
                    transition={{ duration: 0.2 }}
                    className={`flex flex-col w-full mb-8 ${isUser ? 'items-end' : 'items-start'}`}
                  >
                    {!isUser && (
                      <div className="flex items-center gap-2 mb-2 ml-1">
                        <div className="h-5 w-5 rounded md bg-indigo-500/10 flex items-center justify-center border border-indigo-500/20">
                          <Bot className="h-3 w-3 text-indigo-400" />
                        </div>
                        <span className="text-[11px] font-mono font-medium text-zinc-500 uppercase tracking-wider">pribadi-go</span>
                        {m.interrupted && (
                          <span className="text-[9px] font-mono text-amber-500/70 border border-amber-500/20 px-1 py-0 rounded uppercase tracking-wider ml-1">
                            halted
                          </span>
                        )}
                      </div>
                    )}

                    <div className={`flex flex-col max-w-[85%] ${isUser ? 'items-end' : 'items-start'}`}>
                      {isUser && m.file_name && (
                        <div className="flex items-center gap-2 px-2.5 py-1.5 rounded-lg border border-zinc-800/80 bg-zinc-900/80 mb-2 shadow-sm">
                          {(m as any)._localPreview ||
                          (m.file_mime_type && m.file_mime_type.startsWith("image/")) ? (
                            <img
                              src={(m as any)._localPreview || "/placeholder-image.png"}
                              alt="preview"
                              className="h-7 w-7 rounded object-cover shrink-0 border border-zinc-800"
                            />
                          ) : (
                            <div className="h-7 w-7 rounded bg-zinc-800/80 flex items-center justify-center shrink-0 border border-zinc-700/50">
                              <FileText className="h-3.5 w-3.5 text-zinc-400" />
                            </div>
                          )}
                          <div className="flex flex-col min-w-0 pr-2">
                            <span className="text-[11px] text-zinc-200 truncate font-medium">
                              {m.file_name}
                            </span>
                            {m.file_size && (
                              <span className="text-[9px] text-zinc-500 font-mono mt-0.5">
                                {m.file_size < 1024
                                  ? `${m.file_size} B`
                                  : m.file_size < 1024 * 1024
                                    ? `${(m.file_size / 1024).toFixed(1)} KB`
                                    : `${(m.file_size / (1024 * 1024)).toFixed(1)} MB`}
                              </span>
                            )}
                          </div>
                        </div>
                      )}

                      {!isUser && (m.stages?.length || m.thinking) ? (
                        <div className="mb-3 w-full">
                          <ThinkingStages
                            stages={m.stages || []}
                            currentStage={isLastAssistant && isStreaming ? currentStage : null}
                            thinking={m.thinking}
                            isActive={isLastAssistant && isStreaming}
                          />
                        </div>
                      ) : null}

                      {!isUser && m.toolCalls && m.toolCalls.length > 0 && (
                        <div className="flex flex-wrap gap-1.5 mb-3">
                          {m.toolCalls.map((tc, tcIdx) => (
                            <div
                              key={tcIdx}
                              className="inline-flex items-center gap-1.5 bg-zinc-900/60 border border-zinc-800/60 px-2 py-1 rounded text-[10px] text-zinc-400 font-mono"
                            >
                              <Terminal className="h-2.5 w-2.5 text-zinc-500" />
                              <span>{tc.function.name}</span>
                            </div>
                          ))}
                        </div>
                      )}

                      {m.content && (
                        <div className={`relative px-4 py-3 text-sm shadow-sm ${
                          isUser 
                            ? "bg-zinc-800/80 text-zinc-100 rounded-2xl rounded-tr-sm border border-zinc-700/50" 
                            : "bg-transparent text-zinc-300 w-full"
                        }`}>
                          <div className={`prose prose-sm max-w-none prose-p:leading-relaxed ${
                            isUser ? 'prose-invert' : 'prose-invert prose-pre:bg-zinc-900/80 prose-pre:border prose-pre:border-zinc-800/60'
                          }`}>
                            {isUser ? (
                                <div className="whitespace-pre-wrap font-sans">{m.content}</div>
                            ) : (
                              <ReactMarkdown
                                remarkPlugins={[remarkGfm]}
                                components={{
                                  code({ node, inline, className, children, ...props }: any) {
                                    const match = /language-(\w+)/.exec(className || "");
                                    return !inline ? (
                                      <div className="relative rounded-lg overflow-hidden my-4 border border-zinc-800/60 bg-zinc-950 shadow-sm">
                                        <div className="bg-zinc-900/80 px-3 py-1.5 flex items-center text-[10px] text-zinc-400 font-mono uppercase tracking-wider border-b border-zinc-800/60">
                                          {match?.[1] || "text"}
                                        </div>
                                        <pre className="p-3 m-0 overflow-x-auto text-[13px] font-mono leading-relaxed text-zinc-300">
                                          <code className={className} {...props}>
                                            {children}
                                          </code>
                                        </pre>
                                      </div>
                                    ) : (
                                      <code
                                        className="bg-zinc-800/60 text-zinc-200 px-1 py-0.5 rounded text-[13px] font-mono border border-zinc-700/50"
                                        {...props}
                                      >
                                        {children}
                                      </code>
                                    );
                                  },
                                  p({ children }) {
                                    return <p className="mb-4 last:mb-0 text-[14.5px] text-zinc-300/90">{children}</p>;
                                  },
                                  li({ children }) {
                                    return <li className="text-[14.5px] text-zinc-300/90 marker:text-zinc-500">{children}</li>;
                                  }
                                }}
                              >
                                {m.content}
                              </ReactMarkdown>
                            )}
                          </div>
                          
                          {isStreaming && isLastAssistant && (
                            <span className="inline-block w-1.5 h-3.5 bg-indigo-500 ml-1 translate-y-[2px] animate-pulse" />
                          )}
                        </div>
                      )}
                      
                      {!m.content && !m.thinking && isStreaming && isLastAssistant && (
                          <div className="px-1 py-2">
                             <span className="inline-block w-1.5 h-3.5 bg-indigo-500 animate-pulse" />
                          </div>
                      )}
                    </div>
                  </motion.div>
                );
              })}
            </AnimatePresence>
          )}
        </div>
      </div>

      <div className="absolute bottom-0 w-full z-20 bg-gradient-to-t from-[#09090b] via-[#09090b]/95 to-transparent pt-8 pb-4 px-4">
        <div className="max-w-3xl mx-auto">
          <ChatInput
            onSubmit={handleSubmit}
            onStop={handleStop}
            isStreaming={isStreaming}
            attachedFile={attachedFile}
            setAttachedFile={setAttachedFile}
          />
          <div className="flex justify-between items-center mt-2.5 px-1">
             <p className="text-[10px] text-zinc-500 font-mono uppercase tracking-wider">
               pribadi-go v0.1.0
             </p>
             <p className="text-[10px] text-zinc-600 font-mono uppercase tracking-wider flex items-center gap-1">
               <ShieldAlert className="h-2.5 w-2.5" /> Local Execution
             </p>
          </div>
        </div>
      </div>
    </div>
  );
}
