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
      <SelectTrigger className="w-[180px] bg-transparent border-none shadow-none focus:ring-0 font-medium hover:bg-white/5 transition-colors text-zinc-300">
        <SelectValue placeholder="Select Model" />
      </SelectTrigger>
      <SelectContent className="bg-zinc-950 border-white/10 text-zinc-200 shadow-2xl">
        <SelectItem value="local" className="cursor-pointer focus:bg-zinc-900 focus:text-white">
          <div className="flex items-center gap-2">
            <Cpu className="h-4 w-4 text-emerald-400" />
            <span>Local (qwen3:4b)</span>
          </div>
        </SelectItem>
        <SelectItem value="openai" className="cursor-pointer focus:bg-zinc-900 focus:text-white">
          <div className="flex items-center gap-2">
            <Cloud className="h-4 w-4 text-blue-400" />
            <span>GPT-4o (BYOK)</span>
          </div>
        </SelectItem>
        <SelectItem value="anthropic" className="cursor-pointer focus:bg-zinc-900 focus:text-white">
          <div className="flex items-center gap-2">
            <Cloud className="h-4 w-4 text-amber-400" />
            <span>Claude 3.5 (BYOK)</span>
          </div>
        </SelectItem>
      </SelectContent>
    </Select>
  );
}
