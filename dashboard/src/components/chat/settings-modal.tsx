"use client";

import { useState } from "react";
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogTrigger } from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { Settings, Key, Bot } from "lucide-react";
import { fetchApi } from "@/lib/api";

export function SettingsModal() {
  const [open, setOpen] = useState(false);
  const [saving, setSaving] = useState(false);
  const [formData, setFormData] = useState({
    systemPrompt: "",
    openAIKey: "",
    anthropicKey: ""
  });

  const handleSave = async () => {
    setSaving(true);
    try {
      await fetchApi('/chat/settings', {
        method: 'PUT',
        body: JSON.stringify({
          system_prompt: formData.systemPrompt,
          openai_key: formData.openAIKey,
          anthropic_key: formData.anthropicKey
        })
      });
      setOpen(false);
      setFormData({ systemPrompt: "", openAIKey: "", anthropicKey: "" }); // Reset write-only
    } catch (e) {
      console.error(e);
    } finally {
      setSaving(false);
    }
  };

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>
        <Button variant="ghost" className="w-full justify-start gap-2 text-zinc-400 hover:bg-zinc-900/50 hover:text-zinc-300">
          <Settings className="h-4 w-4" />
          Settings
        </Button>
      </DialogTrigger>
      <DialogContent className="sm:max-w-[500px] bg-zinc-950 border-white/10 text-zinc-200 p-0 overflow-hidden shadow-2xl">
        <DialogHeader className="p-6 pb-2">
          <DialogTitle className="text-lg font-medium flex items-center gap-2 text-white">
            <Settings className="h-4 w-4" /> Configuration
          </DialogTitle>
        </DialogHeader>
        
        <div className="space-y-6 p-6 pt-2">
          {/* System Prompt Section */}
          <div className="space-y-3">
            <div className="flex items-center gap-2 text-sm font-medium text-zinc-400">
              <Bot className="h-4 w-4" />
              <span>Persona & Instructions</span>
            </div>
            <div className="space-y-1">
              <Textarea 
                placeholder="You are a helpful assistant..."
                className="resize-none h-24 bg-zinc-900 border-white/5 focus-visible:ring-1 focus-visible:ring-white/20 text-sm placeholder:text-zinc-600"
                value={formData.systemPrompt}
                onChange={e => setFormData({...formData, systemPrompt: e.target.value})}
              />
            </div>
          </div>

          <div className="w-full h-px bg-white/5" />

          {/* BYOK Section */}
          <div className="space-y-3">
            <div className="flex items-center gap-2 text-sm font-medium text-zinc-400">
              <Key className="h-4 w-4" />
              <span>External Providers (BYOK)</span>
            </div>
            <p className="text-[11px] text-zinc-500">
              Keys are encrypted at rest with AES-GCM and never exposed to the frontend.
            </p>
            
            <div className="space-y-3">
              <div className="space-y-1">
                <Input 
                  type="password" 
                  placeholder="OpenAI API Key (sk-...)"
                  className="bg-zinc-900 border-white/5 focus-visible:ring-1 focus-visible:ring-white/20 text-sm placeholder:text-zinc-600 h-9"
                  value={formData.openAIKey}
                  onChange={e => setFormData({...formData, openAIKey: e.target.value})}
                />
              </div>
              <div className="space-y-1">
                <Input 
                  type="password" 
                  placeholder="Anthropic API Key (sk-ant-...)"
                  className="bg-zinc-900 border-white/5 focus-visible:ring-1 focus-visible:ring-white/20 text-sm placeholder:text-zinc-600 h-9"
                  value={formData.anthropicKey}
                  onChange={e => setFormData({...formData, anthropicKey: e.target.value})}
                />
              </div>
            </div>
          </div>
        </div>

        <div className="flex justify-end gap-2 p-4 bg-zinc-900/50 border-t border-white/5">
          <Button variant="ghost" onClick={() => setOpen(false)} className="h-8 text-xs font-medium hover:bg-white/5">Cancel</Button>
          <Button onClick={handleSave} disabled={saving} className="h-8 text-xs font-medium bg-white text-black hover:bg-zinc-200">
            {saving ? "Saving..." : "Save Changes"}
          </Button>
        </div>
      </DialogContent>
    </Dialog>
  );
}
