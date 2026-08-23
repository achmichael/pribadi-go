"use client";

import { useState, useRef, useEffect } from "react";
import { Send, Sparkles, Terminal, FileText, Database, Wrench } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Textarea } from "@/components/ui/textarea";
import { ScrollArea } from "@/components/ui/scroll-area";
import { useChatStore } from "@/store/chat";
import { API_BASE, getAuthHeader } from "@/lib/api";
import ReactMarkdown from "react-markdown";
import remarkGfm from "remark-gfm";
import { ModelSwitcher } from "@/components/chat/model-switcher";
import { FileUpload } from "@/components/chat/file-upload";
import { motion, AnimatePresence } from "framer-motion";

const SUGGESTED_PROMPTS = [
  { icon: FileText, text: "Summarize the latest document I uploaded" },
  { icon: Database, text: "Search my Qdrant memory for my preferences" },
  { icon: Terminal, text: "Write a bash script to parse JSON logs" },
  { icon: Sparkles, text: "What capabilities do you have?" },
];

export function ChatArea() {
  const [input, setInput] = useState("");
  const [model, setModel] = useState("local");
  const {
    messages,
    addMessage,
    appendStreamChunk,
    setIsStreaming,
    isStreaming,
    activeSessionId,
    setActiveSessionId,
    setSessions,
    sessions,
  } = useChatStore();
  const scrollRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (scrollRef.current) {
      scrollRef.current.scrollTop = scrollRef.current.scrollHeight;
    }
  }, [messages]);

  const handleSubmit = async (textToSubmit?: string) => {
    const text = textToSubmit || input;
    if (!text.trim() || isStreaming) return;

    setInput("");

    const tempId = Date.now().toString();
    addMessage({ id: tempId, role: "user", content: text, createdAt: new Date().toISOString() });
    setIsStreaming(true);

    const asstId = (Date.now() + 1).toString();
    addMessage({ id: asstId, role: "assistant", content: "", createdAt: new Date().toISOString() });

    let currentSessionId = activeSessionId;
    // if session id is not exist, this indices to create new chat session
    if (!activeSessionId) {
      const res = await fetch(`${API_BASE}/chat/sessions`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          ...getAuthHeader(),
        },
        body: JSON.stringify({
          message: text,
        }),
      });

      if (!res.ok) {
        if (res.status === 401 || res.status === 403) {
          if (typeof window !== 'undefined') {
            localStorage.removeItem('token');
            if (window.location.pathname !== '/login') {
              window.location.href = '/login';
            }
          }
          throw new Error('Unauthorized');
        }
        throw new Error("Create chat session failed");
      }

      const result = await res.json();
      currentSessionId = result.id;
      setActiveSessionId(result.id);
      setSessions([...sessions, result]);
    }

    try {
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
        }),
      });

      if (!res.ok) {
        if (res.status === 401 || res.status === 403) {
          if (typeof window !== 'undefined') {
            localStorage.removeItem('token');
            if (window.location.pathname !== '/login') {
              window.location.href = '/login';
            }
          }
          throw new Error('Unauthorized');
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
                }
              }
            }
          }
        }
      }
    } catch (e) {
      console.error(e);
      appendStreamChunk("\n\n*Error: Failed to fetch response*", undefined);
    } finally {
      setIsStreaming(false);
    }
  };

  return (
    <div className="flex flex-col h-full w-full mx-auto relative overflow-hidden bg-background">
      {/* Ambient background glows */}
      <div className="ambient-blob bg-blue-500/20 w-[600px] h-[600px] top-[-200px] right-[10%]"></div>
      <div className="ambient-blob bg-purple-500/10 w-[500px] h-[500px] bottom-[-100px] left-[5%]"></div>

      {/* Header */}
      <div className="absolute top-0 w-full z-10 px-6 py-4 flex items-center justify-between">
        <ModelSwitcher model={model} setModel={setModel} />
      </div>

      <div className="flex-1 overflow-y-auto px-4 md:px-8 mt-16 z-10" ref={scrollRef}>
        <div className="max-w-3xl mx-auto space-y-8 pb-32">
          {messages.length === 0 ? (
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
              {messages.map((m, idx) => (
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
                    </div>

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
                      m.id === messages[messages.length - 1]?.id &&
                      m.role === "assistant" && <span className="streaming-cursor" />}
                  </div>
                </motion.div>
              ))}
            </AnimatePresence>
          )}
        </div>
      </div>

      <div className="absolute bottom-0 w-full z-20 bg-gradient-to-t from-background via-background to-transparent pt-10 pb-6 px-4 md:px-8">
        <div className="max-w-3xl mx-auto">
          <div className="relative flex flex-col glow-effect rounded-2xl bg-zinc-900/50 backdrop-blur-md border border-white/10 transition-all">
            <Textarea
              value={input}
              onChange={(e) => setInput(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === "Enter" && !e.shiftKey) {
                  e.preventDefault();
                  handleSubmit();
                }
              }}
              placeholder="Ask anything..."
              className="min-h-[56px] max-h-[200px] border-0 focus-visible:ring-0 resize-none bg-transparent py-4 px-4 text-base placeholder:text-zinc-500"
              rows={1}
            />
            <div className="flex items-center justify-between p-2">
              <FileUpload />
              <Button
                onClick={() => handleSubmit()}
                disabled={!input.trim() || isStreaming}
                size="icon"
                className={`rounded-xl h-9 w-9 transition-all duration-300 ${input.trim() ? "bg-white text-black hover:bg-zinc-200" : "bg-zinc-800 text-zinc-500"}`}
              >
                <Send className="h-4 w-4" />
              </Button>
            </div>
          </div>
          <p className="text-[11px] text-center text-zinc-600 mt-3 font-medium">
            AI can make mistakes. Everything runs locally by default.
          </p>
        </div>
      </div>
    </div>
  );
}
