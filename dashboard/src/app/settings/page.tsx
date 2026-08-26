"use client";

import { useState, useEffect } from "react";
import { useRouter } from "next/navigation";
import { Sidebar } from "@/components/layout/sidebar";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { 
  User, 
  Settings, 
  Paintbrush, 
  LogOut,
  Bot,
  Key,
  Database,
  Trash2,
  FileText
} from "lucide-react";
import { fetchApi } from "@/lib/api";

type Tab = 'preferences' | 'personalize' | 'config' | 'storage' | 'logout';

export default function SettingsPage() {
  const router = useRouter();
  const [activeTab, setActiveTab] = useState<Tab>('config');

  // Config State
  const [saving, setSaving] = useState(false);
  const [loading, setLoading] = useState(true);
  const [formData, setFormData] = useState({
    systemPrompt: "",
    openAIKey: "",
    anthropicKey: ""
  });
  const [keyStatus, setKeyStatus] = useState({
    hasOpenAI: false,
    hasAnthropic: false
  });

  // Storage State
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const [uploads, setUploads] = useState<any[]>([]);
  const [loadingUploads, setLoadingUploads] = useState(false);

  useEffect(() => {
    fetchApi('/chat/settings')
      .then(data => {
        setFormData(prev => ({
          ...prev,
          systemPrompt: data.system_prompt || ""
        }));
        setKeyStatus({
          hasOpenAI: !!data.has_openai_key,
          hasAnthropic: !!data.has_anthropic_key
        });
        setLoading(false);
      })
      .catch(err => {
        console.error("Failed to load settings:", err);
        setLoading(false);
      });
  }, []);

  useEffect(() => {
    if (activeTab === 'storage') {
      const fetchUploads = async () => {
        setLoadingUploads(true);
        try {
          const data = await fetchApi('/chat/uploads');
          setUploads(data || []);
        } catch (err) {
          console.error(err);
        } finally {
          setLoadingUploads(false);
        }
      };
      fetchUploads();
    }
  }, [activeTab]);

  const handleDeleteUpload = async (id: string) => {
    try {
      await fetchApi(`/chat/uploads/${id}`, { method: 'DELETE' });
      setUploads(prev => prev.filter(u => u.id !== id));
    } catch (err) {
      console.error("Failed to delete upload", err);
    }
  };

  const handlePurgeUploads = async () => {
    if (!confirm("Are you sure you want to delete ALL uploaded files? This cannot be undone.")) return;
    try {
      await fetchApi('/chat/uploads', { method: 'DELETE' });
      setUploads([]);
    } catch (err) {
      console.error("Failed to purge uploads", err);
    }
  };

  const handleLogout = () => {
    localStorage.removeItem("token");
    router.push("/login");
  };

  const handleSaveConfig = async () => {
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
      // Update key status based on whether new keys were provided
      if (formData.openAIKey) setKeyStatus(prev => ({ ...prev, hasOpenAI: true }));
      if (formData.anthropicKey) setKeyStatus(prev => ({ ...prev, hasAnthropic: true }));
      
      // Clear input fields for security, but keep system prompt
      setFormData(prev => ({ ...prev, openAIKey: "", anthropicKey: "" })); 
    } catch (e) {
      console.error(e);
    } finally {
      setSaving(false);
    }
  };

  const renderTabContent = () => {
    switch (activeTab) {
      case 'preferences':
        return (
          <div className="space-y-6">
            <div>
              <h3 className="text-lg font-medium text-white">General Preferences</h3>
              <p className="text-sm text-zinc-400">Manage your general application settings.</p>
            </div>
            <div className="h-px bg-zinc-800" />
            <div className="text-zinc-500 text-sm italic">Coming soon...</div>
          </div>
        );
      case 'personalize':
        return (
          <div className="space-y-6">
            <div>
              <h3 className="text-lg font-medium text-white">Personalize</h3>
              <p className="text-sm text-zinc-400">Customize how the assistant behaves and looks.</p>
            </div>
            <div className="h-px bg-zinc-800" />
            <div className="text-zinc-500 text-sm italic">Coming soon...</div>
          </div>
        );
      case 'config':
        return (
          <div className="space-y-6 max-w-2xl">
            <div>
              <h3 className="text-lg font-medium text-white">Model Configuration</h3>
              <p className="text-sm text-zinc-400">Manage system prompts and external provider keys.</p>
            </div>
            <div className="h-px bg-zinc-800" />
            
            {loading ? (
              <div className="text-sm text-zinc-500">Loading settings...</div>
            ) : (
            <div className="space-y-6">
              <div className="space-y-3">
                <div className="flex items-center gap-2 text-sm font-medium text-zinc-300">
                  <Bot className="h-4 w-4" />
                  <span>Persona & Instructions</span>
                </div>
                <Textarea 
                  placeholder="You are a helpful assistant..."
                  className="resize-none h-32 bg-zinc-900 border-zinc-800 focus-visible:ring-1 focus-visible:ring-zinc-700 text-sm placeholder:text-zinc-600"
                  value={formData.systemPrompt}
                  onChange={e => setFormData({...formData, systemPrompt: e.target.value})}
                />
              </div>

              <div className="space-y-3 pt-4 border-t border-zinc-800/50">
                <div className="flex items-center gap-2 text-sm font-medium text-zinc-300">
                  <Key className="h-4 w-4" />
                  <span>External Providers (BYOK)</span>
                </div>
                <p className="text-[13px] text-zinc-500">
                  Keys are encrypted at rest and never exposed to the frontend.
                </p>
                
                <div className="space-y-4">
                  <div className="space-y-1.5">
                    <label className="text-xs font-medium text-zinc-400">
                      OpenAI API Key {keyStatus.hasOpenAI && <span className="text-green-500 ml-2">(Set)</span>}
                    </label>
                    <Input 
                      type="password" 
                      placeholder={keyStatus.hasOpenAI ? "Enter new key to replace..." : "sk-..."}
                      className="bg-zinc-900 border-zinc-800 focus-visible:ring-1 focus-visible:ring-zinc-700 text-sm placeholder:text-zinc-600 h-10 max-w-md"
                      value={formData.openAIKey}
                      onChange={e => setFormData({...formData, openAIKey: e.target.value})}
                    />
                  </div>
                  <div className="space-y-1.5">
                    <label className="text-xs font-medium text-zinc-400">
                      Anthropic API Key {keyStatus.hasAnthropic && <span className="text-green-500 ml-2">(Set)</span>}
                    </label>
                    <Input 
                      type="password" 
                      placeholder={keyStatus.hasAnthropic ? "Enter new key to replace..." : "sk-ant-..."}
                      className="bg-zinc-900 border-zinc-800 focus-visible:ring-1 focus-visible:ring-zinc-700 text-sm placeholder:text-zinc-600 h-10 max-w-md"
                      value={formData.anthropicKey}
                      onChange={e => setFormData({...formData, anthropicKey: e.target.value})}
                    />
                  </div>
                </div>
              </div>

              <div className="pt-4 flex justify-start">
                <Button 
                  onClick={handleSaveConfig} 
                  disabled={saving} 
                  className="bg-white text-black hover:bg-zinc-200"
                >
                  {saving ? "Saving..." : "Save Configuration"}
                </Button>
              </div>
            </div>
            )}
          </div>
        );
      case 'storage':
        return (
          <div className="space-y-6">
            <div className="flex items-center justify-between">
              <div>
                <h3 className="text-lg font-medium text-white">Storage Management</h3>
                <p className="text-sm text-zinc-400">Manage files uploaded across all platforms (Web, Telegram, WhatsApp).</p>
              </div>
              <Button 
                variant="destructive" 
                size="sm" 
                className="bg-red-500/10 text-red-400 hover:bg-red-500/20 hover:text-red-300 border border-red-500/20"
                onClick={handlePurgeUploads}
                disabled={uploads.length === 0}
              >
                Purge All
              </Button>
            </div>
            <div className="h-px bg-zinc-800" />
            
            <div className="border border-zinc-800 rounded-lg overflow-hidden bg-zinc-900/30">
              <div className="grid grid-cols-12 gap-4 p-4 border-b border-zinc-800 text-xs font-medium text-zinc-400 bg-zinc-900/50">
                <div className="col-span-5">Filename</div>
                <div className="col-span-2">Status</div>
                <div className="col-span-2">Size</div>
                <div className="col-span-2">Date</div>
                <div className="col-span-1 text-right">Action</div>
              </div>
              <div className="divide-y divide-zinc-800/50">
                {loadingUploads ? (
                  <div className="p-4 text-center text-sm text-zinc-500">Loading uploads...</div>
                ) : uploads.length === 0 ? (
                  <div className="p-4 text-center text-sm text-zinc-500">No files found.</div>
                ) : (
                  uploads.map((job) => (
                    <div key={job.id} className="grid grid-cols-12 gap-4 p-4 items-center text-sm text-zinc-300 hover:bg-zinc-800/30 transition-colors">
                      <div className="col-span-5 flex items-center gap-2 truncate">
                        <FileText className="h-4 w-4 text-zinc-500 shrink-0" />
                        <span className="truncate" title={job.file_name}>{job.file_name}</span>
                      </div>
                      <div className="col-span-2 flex items-center gap-1.5">
                        <span className={`text-xs px-2 py-0.5 rounded-full border ${
                          job.status === 'completed' ? 'bg-green-500/10 text-green-400 border-green-500/20' :
                          job.status === 'failed' ? 'bg-red-500/10 text-red-400 border-red-500/20' :
                          'bg-blue-500/10 text-blue-400 border-blue-500/20'
                        }`}>
                          {job.status}
                        </span>
                      </div>
                      <div className="col-span-2 text-zinc-500">
                        {(job.file_size / (1024 * 1024)).toFixed(2)} MB
                      </div>
                      <div className="col-span-2 text-zinc-500">
                        {new Date(job.created_at).toLocaleDateString()}
                      </div>
                      <div className="col-span-1 flex justify-end">
                        <Button 
                          variant="ghost" 
                          size="icon" 
                          className="h-8 w-8 text-zinc-500 hover:text-red-400 hover:bg-red-500/10"
                          onClick={() => handleDeleteUpload(job.id)}
                        >
                          <Trash2 className="h-4 w-4" />
                        </Button>
                      </div>
                    </div>
                  ))
                )}
              </div>
            </div>
          </div>
        );
      default:
        return null;
    }
  };

  return (
    <div className="flex h-screen bg-black">
      <Sidebar />
      <div className="flex-1 flex flex-col min-w-0">
        <header className="h-16 flex items-center px-6 border-b border-zinc-800/50 bg-[#09090b]">
          <h1 className="text-lg font-semibold text-white">Settings</h1>
        </header>
        
        <div className="flex-1 flex overflow-hidden">
          {/* Settings Nav */}
          <div className="w-64 border-r border-zinc-800/50 bg-zinc-950/50 p-4 flex flex-col gap-1 overflow-y-auto">
            <div className="text-xs font-semibold text-zinc-500 uppercase tracking-wider mb-2 px-2">Account</div>
            
            <button
              onClick={() => setActiveTab('preferences')}
              className={`flex items-center gap-3 px-3 py-2 rounded-md text-sm transition-colors ${
                activeTab === 'preferences' 
                  ? 'bg-zinc-800 text-white font-medium' 
                  : 'text-zinc-400 hover:bg-zinc-900 hover:text-zinc-200'
              }`}
            >
              <User className="h-4 w-4" />
              Preferences
            </button>
            
            <button
              onClick={() => setActiveTab('personalize')}
              className={`flex items-center gap-3 px-3 py-2 rounded-md text-sm transition-colors ${
                activeTab === 'personalize' 
                  ? 'bg-zinc-800 text-white font-medium' 
                  : 'text-zinc-400 hover:bg-zinc-900 hover:text-zinc-200'
              }`}
            >
              <Paintbrush className="h-4 w-4" />
              Personalize
            </button>

            <div className="text-xs font-semibold text-zinc-500 uppercase tracking-wider mb-2 mt-6 px-2">System</div>
            
            <button
              onClick={() => setActiveTab('config')}
              className={`flex items-center gap-3 px-3 py-2 rounded-md text-sm transition-colors ${
                activeTab === 'config' 
                  ? 'bg-zinc-800 text-white font-medium' 
                  : 'text-zinc-400 hover:bg-zinc-900 hover:text-zinc-200'
              }`}
            >
              <Settings className="h-4 w-4" />
              Configuration
            </button>
            
            <button
              onClick={() => setActiveTab('storage')}
              className={`flex items-center gap-3 px-3 py-2 rounded-md text-sm transition-colors ${
                activeTab === 'storage' 
                  ? 'bg-zinc-800 text-white font-medium' 
                  : 'text-zinc-400 hover:bg-zinc-900 hover:text-zinc-200'
              }`}
            >
              <Database className="h-4 w-4" />
              Storage
            </button>

            <div className="mt-auto pt-4 border-t border-zinc-800/50">
              <button
                onClick={handleLogout}
                className="w-full flex items-center gap-3 px-3 py-2 rounded-md text-sm text-red-400 hover:bg-red-500/10 hover:text-red-300 transition-colors"
              >
                <LogOut className="h-4 w-4" />
                Logout
              </button>
            </div>
          </div>
          
          {/* Content Area */}
          <div className="flex-1 overflow-y-auto bg-black p-8">
            <div className="max-w-4xl mx-auto">
              {renderTabContent()}
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
