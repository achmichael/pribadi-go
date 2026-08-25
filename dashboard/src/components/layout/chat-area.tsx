"use client";

import { useState, useRef, useEffect, memo } from "react";
import { useParams, useRouter } from "next/navigation";
import { Send, Square, Sparkles, Terminal, FileText, Database, Wrench } from "lucide-react";
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

const SUGGESTED_PROMPTS = [
  { icon: FileText, text: "Summarize the latest document I uploaded" },
  { icon: Database, text: "Search my Qdrant memory for my preferences" },
  { icon: Terminal, text: "Write a bash script to parse JSON logs" },
  { icon: Sparkles, text: "What capabilities do you have?" },
];

async function uploadFile(file: File, signal?: AbortSignal): Promise<string | null> {
  const formData = new FormData();
  formData.append("file", file);

  const res = await fetch(`${API_BASE}/chat/upload`, {
    method: "POST",
    headers: {
      ...getAuthHeader()
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
  setAttachedFile 
}: { 
  onSubmit: (text: string) => void; 
  onStop: () => void; 
  isStreaming: boolean;
  attachedFile: AttachedFile | null;
  setAttachedFile: (file: AttachedFile | null) => void;
}) {
  const [input, setInput] = useState("");
  const canSubmit = input.trim() || attachedFile;

  return (
    <div className="relative flex flex-col glow-effect rounded-2xl bg-zinc-900/50 backdrop-blur-md border border-white/10 transition-all">
      <Textarea
        value={input}
        onChange={(e) => setInput(e.target.value)}
        onKeyDown={(e) => {
          if (e.key === "Enter" && !e.shiftKey) {
            e.preventDefault();
            onSubmit(input);
            setInput("");
          }
        }}
        placeholder="Ask anything..."
        className="min-h-[56px] max-h-[200px] border-0 focus-visible:ring-0 resize-none bg-transparent py-4 px-4 text-base placeholder:text-zinc-500"
        rows={1}
      />
      <div className="flex items-center justify-between p-2">
        <FileUpload
          attachedFile={attachedFile}
          onFileSelect={setAttachedFile}
          disabled={isStreaming}
        />
        {isStreaming ? (
          <Button
            onClick={onStop}
            size="icon"
            className="rounded-xl h-9 w-9 bg-red-500/80 text-white hover:bg-red-500 transition-all duration-300"
          >
            <Square className="h-3.5 w-3.5 fill-current" />
          </Button>
        ) : (
          <Button
            onClick={() => { onSubmit(input); setInput(""); }}
            disabled={!canSubmit}
            size="icon"
            className={`rounded-xl h-9 w-9 transition-all duration-300 ${canSubmit ? "bg-white text-black hover:bg-zinc-200" : "bg-zinc-800 text-zinc-500"}`}
          >
            <Send className="h-4 w-4" />
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
                file_size: m.file_size
              }))
            );
          }
        })
        .catch((e) => {
          if (e.message !== 'Unauthorized') {
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
      createdAt: new Date().toISOString()
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
                  appendStreamChunk(data.content, undefined);
                } else if (eventType === "tool" && Array.isArray(data)) {
                  appendStreamChunk("", data);
                } else if (eventType === "thinking" && data.content) {
                  appendThinking(data.content);
                } else if (eventType === "stage") {
                  addStageToLastMessage(data);
                } else if (eventType === "interrupted") {
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

  const canSubmit = input.trim() || attachedFile;

  return (
    <div className="flex flex-col h-full w-full mx-auto relative overflow-hidden bg-background">
      <div className="ambient-blob bg-blue-500/20 w-[600px] h-[600px] top-[-200px] right-[10%]"></div>
      <div className="ambient-blob bg-purple-500/10 w-[500px] h-[500px] bottom-[-100px] left-[5%]"></div>

      <div className="absolute top-0 w-full z-10 px-6 py-4 flex items-center justify-between">
        <ModelSwitcher model={model} setModel={setModel} />
      </div>

      <div className="flex-1 overflow-y-auto px-4 md:px-8 mt-16 z-10" ref={scrollRef}>
        <div className="max-w-3xl mx-auto space-y-8 pb-32">
          {isLoadingHistory ? (
            <div className="flex justify-center items-center h-full mt-32">
              <div className="animate-spin h-6 w-6 border-2 border-zinc-500 border-t-transparent rounded-full" />
            </div>
          ) : historyError ? (
            <div className="flex justify-center items-center h-full mt-32 text-red-400">
              {historyError}
            </div>
          ) : messages.length === 0 ? (
            <motion.div
              initial={{ opacity: 0, y: 20 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ duration: 0.5, ease: "easeOut" }}
              className="flex flex-col items-center justify-center text-center mt-24 md:mt-32"
            >
              <div className="h-16 w-16 rounded-2xl bg-gradient-to-br from-zinc-800 to-black border border-white/10 flex items-center justify-center shadow-2xl mb-8 relative">
                <div className="absolute inset-0 bg-blue-500/10 blur-xl rounded-full"></div>
                <Sparkles className="h-7 w-7 text-white z-10" />
              </div>
              <h2 className="text-3xl font-medium text-gradient mb-3">Good evening.</h2>
              <p className="text-muted-foreground max-w-md text-sm mb-12 leading-relaxed">
                I am pribadi-go, your local intelligence. I can search documents, execute tools, and
                maintain persistent memory.
              </p>

              <div className="grid grid-cols-1 md:grid-cols-2 gap-3 w-full max-w-2xl">
                {SUGGESTED_PROMPTS.map((prompt, i) => (
                  <motion.button
                    key={i}
                    whileHover={{ scale: 1.02 }}
                    whileTap={{ scale: 0.98 }}
                    onClick={() => handleSubmit(prompt.text)}
                    className="flex items-center gap-3 p-4 rounded-xl border border-white/5 bg-white/[0.02] hover:bg-white/[0.05] transition-colors text-left"
                  >
                    <prompt.icon className="h-4 w-4 text-muted-foreground shrink-0" />
                    <span className="text-sm font-medium text-zinc-300">{prompt.text}</span>
                  </motion.button>
                ))}
              </div>
            </motion.div>
          ) : (
            <AnimatePresence>
              {messages.map((m, idx) => {
                const isLastAssistant =
                  m.role === "assistant" && m.id === messages[messages.length - 1]?.id;

                return (
                  <motion.div
                    key={m.id}
                    initial={{ opacity: 0, y: 10 }}
                    animate={{ opacity: 1, y: 0 }}
                    className="flex gap-4 items-start group"
                  >
                    <div className="shrink-0 mt-1">
                      {m.role === "user" ? (
                        <div className="h-7 w-7 rounded-full bg-zinc-800 flex items-center justify-center text-[10px] font-bold text-zinc-400 border border-white/10">
                          YOU
                        </div>
                      ) : (
                        <div className="h-7 w-7 rounded-md bg-white text-black flex items-center justify-center shadow-[0_0_15px_rgba(255,255,255,0.1)]">
                          <Sparkles className="h-4 w-4" />
                        </div>
                      )}
                    </div>

                    <div className="flex-1 space-y-2 min-w-0">
                      <div className="flex items-center gap-2">
                        <span className="text-sm font-medium text-zinc-200">
                          {m.role === "user" ? "You" : "pribadi-go"}
                        </span>
                        {m.interrupted && (
                          <span className="text-[10px] font-mono text-amber-500/70 border border-amber-500/20 px-1.5 py-0.5 rounded">
                            stopped
                          </span>
                        )}
                      </div>

                      {m.role === "user" && m.file_name && (
                        <div className="flex items-center gap-2 px-2.5 py-1.5 rounded-lg border border-zinc-800 bg-zinc-900/50 max-w-[240px] mb-2 mt-1">
                          {(m as any)._localPreview || (m.file_mime_type && m.file_mime_type.startsWith('image/')) ? (
                            <img
                              src={(m as any)._localPreview || '/placeholder-image.png'}
                              alt="preview"
                              className="h-8 w-8 rounded object-cover shrink-0"
                            />
                          ) : (
                            <div className="h-8 w-8 rounded bg-zinc-800 flex items-center justify-center shrink-0">
                              <FileText className="h-4 w-4 text-zinc-400" />
                            </div>
                          )}
                          <div className="flex flex-col min-w-0">
                            <span className="text-[11px] text-zinc-300 truncate font-medium">
                              {m.file_name}
                            </span>
                            {m.file_size && (
                              <span className="text-[10px] text-zinc-500">
                                {m.file_size < 1024 ? `${m.file_size} B` : 
                                 m.file_size < 1024 * 1024 ? `${(m.file_size / 1024).toFixed(1)} KB` : 
                                 `${(m.file_size / (1024 * 1024)).toFixed(1)} MB`}
                              </span>
                            )}
                          </div>
                        </div>
                      )}

                      {m.role === "assistant" && (m.stages?.length || m.thinking) && (
                        <ThinkingStages
                          stages={m.stages || []}
                          currentStage={isLastAssistant && isStreaming ? currentStage : null}
                          thinking={m.thinking}
                          isActive={isLastAssistant && isStreaming}
                        />
                      )}

                      {m.toolCalls && m.toolCalls.length > 0 && (
                        <div className="flex flex-wrap gap-2 mb-2">
                          {m.toolCalls.map((tc, tcIdx) => (
                            <div
                              key={tcIdx}
                              className="inline-flex items-center gap-1.5 bg-zinc-900 border border-zinc-800 px-2 py-1 rounded-md text-xs text-zinc-400 font-mono"
                            >
                              <Wrench className="h-3 w-3" />
                              <span>{tc.function.name}</span>
                            </div>
                          ))}
                        </div>
                      )}

                      {m.content && (
                        <div className="prose prose-invert prose-sm max-w-none prose-p:leading-relaxed prose-pre:bg-zinc-900 prose-pre:border prose-pre:border-zinc-800">
                          <ReactMarkdown
                            remarkPlugins={[remarkGfm]}
                            components={{
                              code({ node, inline, className, children, ...props }: any) {
                                const match = /language-(\w+)/.exec(className || "");
                                return !inline ? (
                                  <div className="relative rounded-lg overflow-hidden my-4 border border-zinc-800 bg-zinc-950">
                                    <div className="bg-zinc-900 px-4 py-2 flex items-center text-xs text-zinc-400 font-mono border-b border-zinc-800">
                                      {match?.[1] || "text"}
                                    </div>
                                    <pre className="p-4 m-0 overflow-x-auto text-sm font-mono leading-relaxed text-zinc-300">
                                      <code className={className} {...props}>
                                        {children}
                                      </code>
                                    </pre>
                                  </div>
                                ) : (
                                  <code
                                    className="bg-zinc-800/50 text-zinc-200 px-1.5 py-0.5 rounded-md font-mono text-sm border border-zinc-800"
                                    {...props}
                                  >
                                    {children}
                                  </code>
                                );
                              },
                            }}
                          >
                            {m.content}
                          </ReactMarkdown>
                        </div>
                      )}

                      {isStreaming &&
                        isLastAssistant &&
                        !m.content && !m.thinking && <span className="streaming-cursor" />}
                      {isStreaming &&
                        isLastAssistant &&
                        m.content && <span className="streaming-cursor" />}
                    </div>
                  </motion.div>
                );
              })}
            </AnimatePresence>
          )}
        </div>
      </div>

      <div className="absolute bottom-0 w-full z-20 bg-gradient-to-t from-background via-background to-transparent pt-10 pb-6 px-4 md:px-8">
        <div className="max-w-3xl mx-auto">
          <ChatInput 
            onSubmit={handleSubmit}
            onStop={handleStop}
            isStreaming={isStreaming}
            attachedFile={attachedFile}
            setAttachedFile={setAttachedFile}
          />
          <p className="text-[11px] text-center text-zinc-600 mt-3 font-medium">
            AI can make mistakes. Everything runs locally by default.
          </p>
        </div>
      </div>
    </div>
  );
}
