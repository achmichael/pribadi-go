"use client";

import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Cpu, Cloud } from "lucide-react";

interface ModelSwitcherProps {
  model: string;
  setModel: (val: string) => void;
}

export function ModelSwitcher({ model, setModel }: ModelSwitcherProps) {
  return (
    <Select value={model} onValueChange={setModel}>
      <SelectTrigger className="w-auto min-w-[140px] h-8 bg-zinc-900/50 border border-zinc-800/80 shadow-sm focus:ring-1 focus:ring-indigo-500/50 font-medium hover:bg-zinc-900 transition-colors text-zinc-300 rounded-md text-xs">
        <SelectValue placeholder="Select Model" />
      </SelectTrigger>
      <SelectContent className="bg-zinc-950/95 backdrop-blur-xl border-zinc-800/80 text-zinc-300 shadow-xl rounded-lg">
        <SelectItem value="local" className="cursor-pointer focus:bg-zinc-800 focus:text-zinc-100 text-xs">
          <div className="flex items-center gap-2">
            <Cpu className="h-3.5 w-3.5 text-indigo-400" />
            <span className="font-mono">qwen3:4b</span>
          </div>
        </SelectItem>
        <SelectItem value="openai" className="cursor-pointer focus:bg-zinc-800 focus:text-zinc-100 text-xs">
          <div className="flex items-center gap-2">
            <Cloud className="h-3.5 w-3.5 text-zinc-500" />
            <span className="font-mono">gpt-4o</span>
          </div>
        </SelectItem>
        <SelectItem value="anthropic" className="cursor-pointer focus:bg-zinc-800 focus:text-zinc-100 text-xs">
          <div className="flex items-center gap-2">
            <Cloud className="h-3.5 w-3.5 text-zinc-500" />
            <span className="font-mono">claude-3.5</span>
          </div>
        </SelectItem>
      </SelectContent>
    </Select>
  );
}
