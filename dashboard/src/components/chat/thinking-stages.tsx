"use client";

import { useState } from "react";
import { ChevronDown, ChevronRight, Search, Cpu, Brain, Square } from "lucide-react";
import { motion, AnimatePresence } from "framer-motion";
import type { StageEvent } from "@/store/chat";

const STAGE_CONFIG: Record<string, { icon: typeof Search; label: string }> = {
  retrieving_context: { icon: Search, label: "Searching documents" },
  context_retrieved: { icon: Search, label: "Context ready" },
  generating: { icon: Cpu, label: "Generating" },
  executing_tool: { icon: Square, label: "Executing tool" },
};

function stageLabel(stage: StageEvent): string {
  const config = STAGE_CONFIG[stage.stage];
  let label = config?.label || stage.stage;
  if (stage.tool) label += ` · ${stage.tool}`;
  if (stage.message && !config) label += ` — ${stage.message}`;
  return label;
}

function StageIcon({ stage }: { stage: string }) {
  const config = STAGE_CONFIG[stage];
  const Icon = config?.icon || Brain;
  return <Icon className="h-3 w-3" />;
}

interface ThinkingStagesProps {
  stages: StageEvent[];
  currentStage: StageEvent | null;
  thinking?: string;
  isActive: boolean;
}

export function ThinkingStages({ stages, currentStage, thinking, isActive }: ThinkingStagesProps) {
  const [thinkingExpanded, setThinkingExpanded] = useState(false);

  const hasThinking = thinking && thinking.length > 0;
  const hasStages = stages.length > 0;

  if (!hasStages && !hasThinking) return null;

  return (
    <div className="mb-3 space-y-1.5">
      {hasStages && (
        <div className="flex flex-wrap gap-1.5">
          {stages.map((s, i) => {
            const isCurrent = isActive && currentStage?.stage === s.stage && i === stages.length - 1;
            return (
              <motion.div
                key={`${s.stage}-${i}`}
                initial={{ opacity: 0, scale: 0.9 }}
                animate={{ opacity: 1, scale: 1 }}
                className={`inline-flex items-center gap-1.5 px-2 py-0.5 rounded-md text-[11px] font-mono border ${
                  isCurrent
                    ? "border-blue-500/30 bg-blue-500/5 text-blue-400"
                    : "border-zinc-800 bg-zinc-900/50 text-zinc-500"
                }`}
              >
                <StageIcon stage={s.stage} />
                <span>{stageLabel(s)}</span>
                {isCurrent && (
                  <span className="inline-block w-1 h-1 rounded-full bg-blue-400 animate-pulse" />
                )}
              </motion.div>
            );
          })}
        </div>
      )}

      {hasThinking && (
        <div className="border border-zinc-800 rounded-lg overflow-hidden bg-zinc-950/50">
          <button
            onClick={() => setThinkingExpanded(!thinkingExpanded)}
            className="flex items-center gap-2 w-full px-3 py-1.5 text-[11px] font-mono text-zinc-400 hover:text-zinc-300 transition-colors"
          >
            {thinkingExpanded ? (
              <ChevronDown className="h-3 w-3" />
            ) : (
              <ChevronRight className="h-3 w-3" />
            )}
            <Brain className="h-3 w-3" />
            <span>Reasoning{isActive ? "..." : ""}</span>
            <span className="text-zinc-600 ml-auto">{thinking.length} chars</span>
          </button>

          <AnimatePresence>
            {thinkingExpanded && (
              <motion.div
                initial={{ height: 0, opacity: 0 }}
                animate={{ height: "auto", opacity: 1 }}
                exit={{ height: 0, opacity: 0 }}
                transition={{ duration: 0.2 }}
              >
                <div className="px-3 pb-2 max-h-[200px] overflow-y-auto">
                  <pre className="text-[11px] leading-relaxed text-zinc-500 whitespace-pre-wrap font-mono break-words">
                    {thinking}
                    {isActive && <span className="streaming-cursor" />}
                  </pre>
                </div>
              </motion.div>
            )}
          </AnimatePresence>
        </div>
      )}
    </div>
  );
}
