"use client";

import { useState, useEffect } from "react";
import { useRouter } from "next/navigation";
import { Sidebar } from "@/components/layout/sidebar";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { Accordion, AccordionContent, AccordionItem, AccordionTrigger } from "@/components/ui/accordion";
import { AlertDialog, AlertDialogAction, AlertDialogCancel, AlertDialogContent, AlertDialogDescription, AlertDialogFooter, AlertDialogHeader, AlertDialogTitle, AlertDialogTrigger } from "@/components/ui/alert-dialog";
import { RadioGroup, RadioGroupItem } from "@/components/ui/radio-group";
import { Slider } from "@/components/ui/slider";
import { Switch } from "@/components/ui/switch";
import { Label } from "@/components/ui/label";

import { 
  User, 
  Settings, 
  Paintbrush, 
  LogOut,
  Database,
  Trash2,
  FileText,
  Cpu,
  Link as LinkIcon,
  Smartphone,
  Globe,
  Plus
} from "lucide-react";
import { fetchApi } from "@/lib/api";

type Tab = 'preferences' | 'personalize' | 'models' | 'connections' | 'storage' | 'logout';

export default function SettingsPage() {
  const router = useRouter();
  const [activeTab, setActiveTab] = useState<Tab>('preferences');

  // Preferences State
  const [prefLang, setPrefLang] = useState('en');
  const [prefLanding, setPrefLanding] = useState('new');
  const [prefAutoArchive, setPrefAutoArchive] = useState('');
  const [prefTimezone, setPrefTimezone] = useState('UTC');

  // Personalize State
  const [persNickname, setPersNickname] = useState('');
  const [persSystemPrompt, setPersSystemPrompt] = useState('');
  const [persStyle, setPersStyle] = useState('concise');
  const [persReplyLang, setPersReplyLang] = useState('en');

  // Memory manager mock data
  const [memoryFacts, setMemoryFacts] = useState([
    { id: '1', fact: 'User likes concise answers', date: '2026-08-20', source: 'Thread 1' }
  ]);

  // Models State
  const [modelSaving, setModelSaving] = useState(false);
  const [modelChanged, setModelChanged] = useState(false);
  
  // External Provider states
  const [openaiKey, setOpenaiKey] = useState('');
  const [openaiModel, setOpenaiModel] = useState('');
  const [anthropicKey, setAnthropicKey] = useState('');
  const [anthropicModel, setAnthropicModel] = useState('');
  const [geminiKey, setGeminiKey] = useState('');
  const [geminiModel, setGeminiModel] = useState('');
  const [grokKey, setGrokKey] = useState('');
  const [grokModel, setGrokModel] = useState('');
  
  const [hasOpenAIKey, setHasOpenAIKey] = useState(false);
  const [hasAnthropicKey, setHasAnthropicKey] = useState(false);
  const [hasGeminiKey, setHasGeminiKey] = useState(false);
  const [hasGrokKey, setHasGrokKey] = useState(false);
  
  const [plannerModel, setPlannerModel] = useState('qwen3:4b');
  const [executorModel, setExecutorModel] = useState('qwen3:4b');
  const [diagModel, setDiagModel] = useState('qwen3:4b');
  const [keepAlive, setKeepAlive] = useState([5]);
  const [thinkMode, setThinkMode] = useState({ planner: true, executor: false, diag: false });
  
  // Fetch existing config from API on mount
  useEffect(() => {
    const loadSettings = async () => {
      try {
        const data = await fetchApi('/chat/settings');
        if (data) {
          if (data.system_prompt) setPersSystemPrompt(data.system_prompt);
          setHasOpenAIKey(data.has_openai_key || false);
          setHasAnthropicKey(data.has_anthropic_key || false);
          setHasGeminiKey(data.has_gemini_key || false);
          setHasGrokKey(data.has_grok_key || false);
          
          if (data.openai_model) setOpenaiModel(data.openai_model);
          if (data.anthropic_model) setAnthropicModel(data.anthropic_model);
          if (data.gemini_model) setGeminiModel(data.gemini_model);
          if (data.grok_model) setGrokModel(data.grok_model);
        }
      } catch (err) {
        console.error("Failed to load settings", err);
      }
    };
    loadSettings();
  }, []);
  
  // Storage State
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const [uploads, setUploads] = useState<any[]>([]);
  const [loadingUploads, setLoadingUploads] = useState(false);

  // Autosave simulation
  const handleAutoSave = (field: string, value: any) => {
    // In real app, make API call here and show toast
    console.log(`Auto-saved ${field}: ${value}`);
  };

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

  const [openaiTesting, setOpenaiTesting] = useState(false);
  const [anthropicTesting, setAnthropicTesting] = useState(false);
  const [geminiTesting, setGeminiTesting] = useState(false);
  const [grokTesting, setGrokTesting] = useState(false);

  const handleTestConnection = async (provider: string) => {
    let key = '';
    let model = '';
    let setter = (b: boolean) => {};
    let hasKey = false;

    if (provider === 'openai') {
      key = openaiKey; model = openaiModel; setter = setOpenaiTesting; hasKey = hasOpenAIKey;
    } else if (provider === 'anthropic') {
      key = anthropicKey; model = anthropicModel; setter = setAnthropicTesting; hasKey = hasAnthropicKey;
    } else if (provider === 'gemini') {
      key = geminiKey; model = geminiModel; setter = setGeminiTesting; hasKey = hasGeminiKey;
    } else if (provider === 'grok') {
      key = grokKey; model = grokModel; setter = setGrokTesting; hasKey = hasGrokKey;
    }

    if (!key && !hasKey) {
      alert("Please enter an API key first");
      return;
    }

    setter(true);
    try {
      // In a real app we'd fetch the saved key if the input is empty (e.g. they just loaded the page)
      // For this test endpoint, we need to pass the explicit key. If the field is empty but it says "Configured",
      // the backend would ideally handle a request without a key by fetching it from the DB. 
      // To keep it simple, we require them to input the key to test if it's a new one, or if they haven't saved it.
      if (!key) {
        alert("Please enter the API key in the field to test (we don't fetch your saved secret key to the frontend for security)");
        setter(false);
        return;
      }

      const res = await fetchApi('/chat/settings/test', {
        method: 'POST',
        body: JSON.stringify({
          provider,
          api_key: key,
          model: model
        })
      });
      
      alert(res.message || "Connection successful!");
    } catch (err: any) {
      alert(`Test failed:\n${err.message || "Unknown error"}`);
    } finally {
      setter(false);
    }
  };
  
  const handleModelSave = async () => {
    setModelSaving(true);
    try {
      const payload: any = {};
      
      // Add external keys/models to payload if they were touched
      if (openaiKey) payload.openai_key = openaiKey;
      if (openaiModel) payload.openai_model = openaiModel;
      
      if (anthropicKey) payload.anthropic_key = anthropicKey;
      if (anthropicModel) payload.anthropic_model = anthropicModel;
      
      if (Object.keys(payload).length > 0) {
        await fetchApi('/chat/settings', {
          method: 'PUT',
          body: JSON.stringify(payload)
        });
        
        if (openaiKey) setHasOpenAIKey(true);
        if (anthropicKey) setHasAnthropicKey(true);
        if (geminiKey) setHasGeminiKey(true);
        if (grokKey) setHasGrokKey(true);
      }
      
      setModelChanged(false);
    } catch (err) {
      console.error("Failed to save models config", err);
    } finally {
      setModelSaving(false);
    }
  };
  const handlePurgeDocuments = async () => {
    try {
      await fetchApi('/chat/uploads', { method: 'DELETE' });
      setUploads([]);
      alert('All document vectors and files have been cleared successfully.');
    } catch (err) {
      console.error("Failed to purge documents", err);
      alert('Failed to clear document vectors.');
    }
  };
  const handleDeleteUpload = async (id: string) => {
    try {
      await fetchApi(`/chat/uploads/${id}`, { method: 'DELETE' });
      setUploads(prev => prev.filter(u => u.id !== id));
    } catch (err) {
      console.error("Failed to delete upload", err);
    }
  };

  const handleLogout = () => {
    localStorage.removeItem("token");
    router.push("/login");
  };

  const renderTabContent = () => {
    switch (activeTab) {
      case 'preferences':
        return (
          <div className="space-y-8 max-w-2xl">
            <div>
              <h3 className="text-lg font-medium text-white">General Preferences</h3>
              <p className="text-sm text-zinc-400">Manage your application UI settings.</p>
            </div>
            
            <div className="space-y-6">
              <div className="space-y-2">
                <Label className="text-zinc-300">Language (UI)</Label>
                <Select value={prefLang} onValueChange={(v) => { setPrefLang(v); handleAutoSave('lang', v); }}>
                  <SelectTrigger className="w-full bg-zinc-900 border-zinc-800 text-zinc-300">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent className="bg-zinc-950 border-white/10 text-zinc-200">
                    <SelectItem value="en" className="focus:bg-zinc-900 focus:text-white">English</SelectItem>
                    <SelectItem value="id" className="focus:bg-zinc-900 focus:text-white">Bahasa Indonesia</SelectItem>
                  </SelectContent>
                </Select>
              </div>

              <div className="space-y-3">
                <Label className="text-zinc-300">Default Landing</Label>
                <RadioGroup value={prefLanding} onValueChange={(v) => { setPrefLanding(v); handleAutoSave('landing', v); }} className="space-y-2">
                  <div className="flex items-center space-x-2 text-zinc-300">
                    <RadioGroupItem value="new" id="r1" className="border-zinc-700 text-zinc-300" />
                    <Label htmlFor="r1">Always new thread</Label>
                  </div>
                  <div className="flex items-center space-x-2 text-zinc-300">
                    <RadioGroupItem value="last" id="r2" className="border-zinc-700 text-zinc-300" />
                    <Label htmlFor="r2">Last opened thread</Label>
                  </div>
                </RadioGroup>
              </div>

              <div className="space-y-2">
                <Label className="text-zinc-300">Thread Auto-archive (Days)</Label>
                <Input 
                  type="number" 
                  placeholder="0 (Disabled)" 
                  value={prefAutoArchive}
                  onChange={(e) => { setPrefAutoArchive(e.target.value); handleAutoSave('archive_days', e.target.value); }}
                  className="bg-zinc-900 border-zinc-800 text-zinc-300 placeholder:text-zinc-600"
                />
                <p className="text-xs text-zinc-500">Leave empty or 0 to disable auto-archiving.</p>
              </div>

              <div className="space-y-2">
                <Label className="text-zinc-300">Timezone</Label>
                <Select value={prefTimezone} onValueChange={(v) => { setPrefTimezone(v); handleAutoSave('timezone', v); }}>
                  <SelectTrigger className="w-full bg-zinc-900 border-zinc-800 text-zinc-300">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent className="bg-zinc-950 border-white/10 text-zinc-200">
                    <SelectItem value="UTC" className="focus:bg-zinc-900 focus:text-white">UTC</SelectItem>
                    <SelectItem value="Asia/Jakarta" className="focus:bg-zinc-900 focus:text-white">Asia/Jakarta (GMT+7)</SelectItem>
                  </SelectContent>
                </Select>
              </div>
            </div>
          </div>
        );

      case 'personalize':
        return (
          <div className="space-y-8 max-w-3xl">
            <div>
              <h3 className="text-lg font-medium text-white">Personalize</h3>
              <p className="text-sm text-zinc-400">Customize how the assistant behaves and remembers you.</p>
            </div>
            
            <div className="space-y-6">
              <div className="space-y-2">
                <Label className="text-zinc-300">Assistant Nickname</Label>
                <Input 
                  value={persNickname}
                  onChange={(e) => { setPersNickname(e.target.value); handleAutoSave('nickname', e.target.value); }}
                  placeholder="e.g. JARVIS"
                  className="bg-zinc-900 border-zinc-800 text-zinc-300 max-w-md"
                />
              </div>

              <div className="space-y-2">
                <Label className="text-zinc-300">Custom System Prompt</Label>
                <Textarea 
                  value={persSystemPrompt}
                  onChange={(e) => { setPersSystemPrompt(e.target.value); handleAutoSave('system_prompt', e.target.value); }}
                  placeholder="Additional instructions for the AI..."
                  className="bg-zinc-900 border-zinc-800 text-zinc-300 min-h-[100px]"
                />
              </div>

              <div className="grid grid-cols-2 gap-4 max-w-md">
                <div className="space-y-2">
                  <Label className="text-zinc-300">Response Style</Label>
                  <Select value={persStyle} onValueChange={(v) => { setPersStyle(v); handleAutoSave('style', v); }}>
                    <SelectTrigger className="w-full bg-zinc-900 border-zinc-800 text-zinc-300">
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent className="bg-zinc-950 border-white/10 text-zinc-200">
                      <SelectItem value="concise" className="focus:bg-zinc-900 focus:text-white">Concise</SelectItem>
                      <SelectItem value="detailed" className="focus:bg-zinc-900 focus:text-white">Detailed</SelectItem>
                    </SelectContent>
                  </Select>
                </div>
                <div className="space-y-2">
                  <Label className="text-zinc-300">Reply Language</Label>
                  <Select value={persReplyLang} onValueChange={(v) => { setPersReplyLang(v); handleAutoSave('reply_lang', v); }}>
                    <SelectTrigger className="w-full bg-zinc-900 border-zinc-800 text-zinc-300">
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent className="bg-zinc-950 border-white/10 text-zinc-200">
                      <SelectItem value="en" className="focus:bg-zinc-900 focus:text-white">English</SelectItem>
                      <SelectItem value="id" className="focus:bg-zinc-900 focus:text-white">Bahasa Indonesia</SelectItem>
                      <SelectItem value="auto" className="focus:bg-zinc-900 focus:text-white">Auto-detect</SelectItem>
                    </SelectContent>
                  </Select>
                </div>
              </div>
            </div>

            <div className="pt-6 border-t border-zinc-800/50">
              <div className="flex items-center justify-between mb-4">
                <div>
                  <h4 className="text-base font-medium text-zinc-200">Memory Manager (User Facts)</h4>
                  <p className="text-xs text-zinc-500">Persistent facts the AI knows about you.</p>
                </div>
                <Button size="sm" variant="outline" className="bg-zinc-900 border-zinc-800 text-zinc-300 hover:bg-zinc-800 hover:text-white">
                  <Plus className="h-4 w-4 mr-2" /> Add Fact
                </Button>
              </div>

              <div className="border border-zinc-800 rounded-md overflow-hidden">
                <Table>
                  <TableHeader className="bg-zinc-900/50">
                    <TableRow className="border-zinc-800 hover:bg-transparent">
                      <TableHead className="text-zinc-400 font-medium">Fact</TableHead>
                      <TableHead className="text-zinc-400 font-medium w-32">Source</TableHead>
                      <TableHead className="text-zinc-400 font-medium w-32">Date</TableHead>
                      <TableHead className="text-right text-zinc-400 font-medium w-24">Actions</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {/* TODO: connect to /api/settings/memory */}
                    {memoryFacts.map((fact) => (
                      <TableRow key={fact.id} className="border-zinc-800/50 hover:bg-zinc-900/30">
                        <TableCell className="text-zinc-300">{fact.fact}</TableCell>
                        <TableCell className="text-zinc-500 text-xs">
                          <span className="cursor-pointer hover:underline hover:text-zinc-300">{fact.source}</span>
                        </TableCell>
                        <TableCell className="text-zinc-500 text-xs">{fact.date}</TableCell>
                        <TableCell className="text-right">
                          <AlertDialog>
                            <AlertDialogTrigger asChild>
                              <Button variant="ghost" size="icon" className="h-7 w-7 text-zinc-500 hover:text-red-400 hover:bg-red-500/10">
                                <Trash2 className="h-3.5 w-3.5" />
                              </Button>
                            </AlertDialogTrigger>
                            <AlertDialogContent className="bg-zinc-950 border-white/10 text-white">
                              <AlertDialogHeader>
                                <AlertDialogTitle>Delete memory fact?</AlertDialogTitle>
                                <AlertDialogDescription className="text-zinc-400">
                                  This will remove this fact from the assistant's context permanently.
                                </AlertDialogDescription>
                              </AlertDialogHeader>
                              <AlertDialogFooter>
                                <AlertDialogCancel className="bg-zinc-900 border-zinc-800 hover:bg-zinc-800 hover:text-white">Cancel</AlertDialogCancel>
                                <AlertDialogAction className="bg-red-500 text-white hover:bg-red-600">Delete</AlertDialogAction>
                              </AlertDialogFooter>
                            </AlertDialogContent>
                          </AlertDialog>
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </div>
            </div>
          </div>
        );

      case 'models':
        return (
          <div className="space-y-8 max-w-2xl">
            <div>
              <h3 className="text-lg font-medium text-white flex items-center gap-2">
                Models Configuration
                {modelChanged && <span className="text-[10px] uppercase font-bold bg-amber-500/20 text-amber-500 px-2 py-0.5 rounded-full ml-2 tracking-wider">Unsaved Changes</span>}
              </h3>
              <p className="text-sm text-zinc-400">Configure local Ollama and external cloud models.</p>
            </div>

            <Accordion type="single" collapsible defaultValue="local" className="w-full space-y-4">
              <AccordionItem value="local" className="border border-zinc-800/50 rounded-lg bg-zinc-900/10 px-4">
                <AccordionTrigger className="hover:no-underline py-4">
                  <div className="flex items-center gap-3 text-white">
                    <Cpu className="h-5 w-5 text-emerald-400" />
                    <div className="text-left">
                      <div className="font-medium">Local Models (Ollama)</div>
                      <div className="text-xs text-zinc-500 font-normal mt-0.5">Primary execution (Zero Data Exposure)</div>
                    </div>
                  </div>
                </AccordionTrigger>
                <AccordionContent className="pt-2 pb-4 space-y-6">
                  <div className="bg-zinc-950/50 border border-zinc-800/50 rounded-xl p-4 flex flex-col gap-2 mb-6">
                    <div className="flex justify-between items-center">
                      <span className="text-xs font-semibold text-zinc-400 uppercase tracking-wider">Live Resource Usage</span>
                      <span className="text-xs text-zinc-300 font-mono">3.1 GB / 4.0 GB VRAM</span>
                    </div>
                    <div className="h-2 w-full bg-zinc-900 rounded-full overflow-hidden">
                      <div className="h-full bg-emerald-400 w-[77%]"></div>
                    </div>
                  </div>
                  
                  {[
                    { label: 'Planner Model', val: plannerModel, setter: setPlannerModel, key: 'planner', think: thinkMode.planner },
                    { label: 'Executor Model', val: executorModel, setter: setExecutorModel, key: 'executor', think: thinkMode.executor },
                    { label: 'Diagnostician Model', val: diagModel, setter: setDiagModel, key: 'diag', think: thinkMode.diag }
                  ].map((item) => (
                    <div key={item.key} className="space-y-2 p-4 border border-zinc-800/50 rounded-lg bg-zinc-950">
                      <div className="flex justify-between items-start mb-2">
                        <Label className="text-zinc-200">{item.label}</Label>
                        <div className="flex items-center space-x-2">
                          <Switch 
                            checked={item.think} 
                            onCheckedChange={(v) => {
                              setThinkMode({...thinkMode, [item.key]: v});
                              setModelChanged(true);
                            }} 
                            className="data-[state=checked]:bg-emerald-500"
                          />
                          <Label className="text-xs text-zinc-400">Think Mode</Label>
                        </div>
                      </div>
                      <Select value={item.val} onValueChange={(v) => { item.setter(v); setModelChanged(true); }}>
                        <SelectTrigger className="w-full bg-zinc-900 border-zinc-800 text-zinc-300">
                          <SelectValue />
                        </SelectTrigger>
                        <SelectContent className="bg-zinc-950 border-white/10 text-zinc-200">
                          <SelectItem value="qwen3:4b" className="focus:bg-zinc-900 focus:text-white">qwen3:4b (Local)</SelectItem>
                          <SelectItem value="llama3:8b" className="focus:bg-zinc-900 focus:text-white">llama3:8b (Local)</SelectItem>
                        </SelectContent>
                      </Select>
                    </div>
                  ))}

                  <div className="space-y-4 pt-2">
                    <div className="flex justify-between items-center">
                      <Label className="text-zinc-300">Keep Alive Duration: {keepAlive[0]}m</Label>
                    </div>
                    <Slider 
                      value={keepAlive} 
                      onValueChange={(v) => { setKeepAlive(v); setModelChanged(true); }}
                      max={60} 
                      step={5}
                      className="py-2"
                    />
                    <p className="text-xs text-zinc-500">Longer keep alive consumes VRAM constantly but prevents cold-start delays.</p>
                  </div>
                </AccordionContent>
              </AccordionItem>

              <AccordionItem value="cloud" className="border border-zinc-800/50 rounded-lg bg-zinc-900/10 px-4">
                <AccordionTrigger className="hover:no-underline py-4">
                  <div className="flex items-center gap-3 text-white">
                    <Globe className="h-5 w-5 text-blue-400" />
                    <div className="text-left">
                      <div className="font-medium">Cloud Providers (Hybrid Routing)</div>
                      <div className="text-xs text-zinc-500 font-normal mt-0.5">Used selectively for non-sensitive generic tasks</div>
                    </div>
                  </div>
                </AccordionTrigger>
                <AccordionContent className="pt-2 pb-4 space-y-6">
                  <div className="space-y-4 p-4 border border-zinc-800/50 rounded-lg bg-zinc-950">
                    <div className="flex items-center justify-between mb-2">
                      <Label className="text-zinc-200 text-base">OpenAI Configuration</Label>
                      {hasOpenAIKey && <span className="text-[10px] font-medium text-emerald-400 uppercase tracking-wider bg-emerald-500/10 px-2 py-1 rounded">Saved</span>}
                    </div>
                    
                    <div className="space-y-2">
                      <Label className="text-zinc-400 text-xs">API Key</Label>
                      <Input 
                        type="password"
                        placeholder={hasOpenAIKey ? "••••••••••••••••••••••••••••••••" : "sk-..."}
                        value={openaiKey}
                        onChange={(e) => { setOpenaiKey(e.target.value); setModelChanged(true); }}
                        className="bg-zinc-900 border-zinc-800 text-zinc-300 font-mono text-sm"
                      />
                    </div>
                    
                    <div className="space-y-2">
                      <Label className="text-zinc-400 text-xs">Model</Label>
                      <Input 
                        placeholder="e.g. gpt-4o-mini (default)"
                        value={openaiModel}
                        onChange={(e) => { setOpenaiModel(e.target.value); setModelChanged(true); }}
                        className="bg-zinc-900 border-zinc-800 text-zinc-300 font-mono text-sm"
                      />
                    </div>
                    
                    <div className="pt-2 flex justify-end">
                       <Button 
                        variant="outline" 
                        size="sm" 
                        disabled={openaiTesting || (!openaiKey && !hasOpenAIKey)}
                        onClick={() => handleTestConnection('openai')}
                        className="bg-zinc-900 border-zinc-700 text-zinc-300 hover:text-white"
                      >
                        {openaiTesting ? "Testing..." : "Test Connection"}
                      </Button>
                    </div>
                  </div>
                  
                  <div className="space-y-4 p-4 border border-zinc-800/50 rounded-lg bg-zinc-950">
                    <div className="flex items-center justify-between mb-2">
                      <Label className="text-zinc-200 text-base">Anthropic</Label>
                      {hasAnthropicKey && <span className="text-[10px] font-medium text-emerald-400 uppercase tracking-wider bg-emerald-500/10 px-2 py-1 rounded">Saved</span>}
                    </div>
                    
                    <div className="space-y-2">
                      <Label className="text-zinc-400 text-xs">API Key</Label>
                      <Input 
                        type="password"
                        placeholder={hasAnthropicKey ? "••••••••••••••••••••••••••••••••" : "sk-ant-..."}
                        value={anthropicKey}
                        onChange={(e) => { setAnthropicKey(e.target.value); setModelChanged(true); }}
                        className="bg-zinc-900 border-zinc-800 text-zinc-300 font-mono text-sm"
                      />
                    </div>
                    
                    <div className="space-y-2">
                      <Label className="text-zinc-400 text-xs">Model</Label>
                      <Input 
                        placeholder="e.g. claude-3-haiku-20240307 (default)"
                        value={anthropicModel}
                        onChange={(e) => { setAnthropicModel(e.target.value); setModelChanged(true); }}
                        className="bg-zinc-900 border-zinc-800 text-zinc-300 font-mono text-sm"
                      />
                    </div>
                    
                    <div className="pt-2 flex justify-end">
                       <Button 
                        variant="outline" 
                        size="sm" 
                        disabled={anthropicTesting || (!anthropicKey && !hasAnthropicKey)}
                        onClick={() => handleTestConnection('anthropic')}
                        className="bg-zinc-900 border-zinc-700 text-zinc-300 hover:text-white"
                      >
                        {anthropicTesting ? "Testing..." : "Test Connection"}
                      </Button>
                    </div>
                  </div>

                  <div className="space-y-4 p-4 border border-zinc-800/50 rounded-lg bg-zinc-950">
                    <div className="flex items-center justify-between mb-2">
                      <Label className="text-zinc-200 text-base">Google Gemini</Label>
                      {hasGeminiKey && <span className="text-[10px] font-medium text-emerald-400 uppercase tracking-wider bg-emerald-500/10 px-2 py-1 rounded">Saved</span>}
                    </div>
                    
                    <div className="space-y-2">
                      <Label className="text-zinc-400 text-xs">API Key</Label>
                      <Input 
                        type="password"
                        placeholder={hasGeminiKey ? "••••••••••••••••••••••••••••••••" : "AIzaSy..."}
                        value={geminiKey}
                        onChange={(e) => { setGeminiKey(e.target.value); setModelChanged(true); }}
                        className="bg-zinc-900 border-zinc-800 text-zinc-300 font-mono text-sm"
                      />
                    </div>
                    
                    <div className="space-y-2">
                      <Label className="text-zinc-400 text-xs">Model</Label>
                      <Input 
                        placeholder="e.g. gemini-1.5-flash (default)"
                        value={geminiModel}
                        onChange={(e) => { setGeminiModel(e.target.value); setModelChanged(true); }}
                        className="bg-zinc-900 border-zinc-800 text-zinc-300 font-mono text-sm"
                      />
                    </div>
                    
                    <div className="pt-2 flex justify-end">
                       <Button 
                        variant="outline" 
                        size="sm" 
                        disabled={geminiTesting || (!geminiKey && !hasGeminiKey)}
                        onClick={() => handleTestConnection('gemini')}
                        className="bg-zinc-900 border-zinc-700 text-zinc-300 hover:text-white"
                      >
                        {geminiTesting ? "Testing..." : "Test Connection"}
                      </Button>
                    </div>
                  </div>

                  <div className="space-y-4 p-4 border border-zinc-800/50 rounded-lg bg-zinc-950">
                    <div className="flex items-center justify-between mb-2">
                      <Label className="text-zinc-200 text-base">xAI Grok</Label>
                      {hasGrokKey && <span className="text-[10px] font-medium text-emerald-400 uppercase tracking-wider bg-emerald-500/10 px-2 py-1 rounded">Saved</span>}
                    </div>
                    
                    <div className="space-y-2">
                      <Label className="text-zinc-400 text-xs">API Key</Label>
                      <Input 
                        type="password"
                        placeholder={hasGrokKey ? "••••••••••••••••••••••••••••••••" : "xai-..."}
                        value={grokKey}
                        onChange={(e) => { setGrokKey(e.target.value); setModelChanged(true); }}
                        className="bg-zinc-900 border-zinc-800 text-zinc-300 font-mono text-sm"
                      />
                    </div>
                    
                    <div className="space-y-2">
                      <Label className="text-zinc-400 text-xs">Model</Label>
                      <Input 
                        placeholder="e.g. grok-beta (default)"
                        value={grokModel}
                        onChange={(e) => { setGrokModel(e.target.value); setModelChanged(true); }}
                        className="bg-zinc-900 border-zinc-800 text-zinc-300 font-mono text-sm"
                      />
                    </div>
                    
                    <div className="pt-2 flex justify-end">
                       <Button 
                        variant="outline" 
                        size="sm" 
                        disabled={grokTesting || (!grokKey && !hasGrokKey)}
                        onClick={() => handleTestConnection('grok')}
                        className="bg-zinc-900 border-zinc-700 text-zinc-300 hover:text-white"
                      >
                        {grokTesting ? "Testing..." : "Test Connection"}
                      </Button>
                    </div>
                  </div>
                </AccordionContent>
              </AccordionItem>
            </Accordion>

            <div className="pt-6 border-t border-zinc-800/50">
              <Button 
                disabled={!modelChanged || modelSaving}
                onClick={handleModelSave}
                className="bg-white text-black hover:bg-zinc-200 disabled:bg-zinc-800 disabled:text-zinc-500 w-full sm:w-auto"
              >
                {modelSaving ? "Saving Configuration..." : "Save Model Configuration"}
              </Button>
            </div>
          </div>
        );

      case 'connections':
        return (
          <div className="space-y-8 max-w-3xl">
            <div>
              <h3 className="text-lg font-medium text-white">Integrations & Connections</h3>
              <p className="text-sm text-zinc-400">Manage external platforms and active web sessions.</p>
            </div>

            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              {/* WhatsApp Card */}
              <div className="flex flex-col border border-zinc-800/80 rounded-xl bg-[#09090b] overflow-hidden">
                <div className="p-5 flex-1">
                  <div className="flex items-start justify-between mb-4">
                    <div className="flex items-center gap-3">
                      <div className="h-10 w-10 rounded-lg bg-[#25D366]/10 flex items-center justify-center border border-[#25D366]/20">
                        <Smartphone className="h-5 w-5 text-[#25D366]" />
                      </div>
                      <div>
                        <h4 className="font-medium text-zinc-200">WhatsApp</h4>
                        <div className="text-xs text-zinc-500 mt-0.5">Local Gateway</div>
                      </div>
                    </div>
                    
                    <div className="flex items-center gap-1.5 px-2 py-1 rounded-md bg-emerald-500/10 border border-emerald-500/20">
                      <span className="relative flex h-1.5 w-1.5">
                        <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-emerald-400 opacity-75"></span>
                        <span className="relative inline-flex rounded-full h-1.5 w-1.5 bg-emerald-500"></span>
                      </span>
                      <span className="text-[10px] font-medium text-emerald-400 uppercase tracking-wider">Connected</span>
                    </div>
                  </div>
                  
                  <p className="text-sm text-zinc-400">
                    Currently linked to <span className="text-zinc-200 font-medium">+62 812-****-4567</span>.
                  </p>
                </div>
                
                <div className="border-t border-zinc-800/80 p-3 bg-zinc-900/30 flex gap-2">
                  <Button variant="ghost" size="sm" className="flex-1 bg-zinc-800/50 hover:bg-zinc-800 text-zinc-300 h-8 text-xs">
                    Relink Device
                  </Button>
                  <AlertDialog>
                    <AlertDialogTrigger asChild>
                      <Button variant="ghost" size="sm" className="flex-1 bg-transparent hover:bg-red-500/10 text-zinc-400 hover:text-red-400 h-8 text-xs">
                        Disconnect
                      </Button>
                    </AlertDialogTrigger>
                    <AlertDialogContent className="bg-zinc-950 border-white/10 text-white">
                      <AlertDialogHeader>
                        <AlertDialogTitle>Disconnect WhatsApp?</AlertDialogTitle>
                        <AlertDialogDescription className="text-zinc-400">
                          This will immediately terminate the WhatsApp session. You will need to scan a QR code to reconnect.
                        </AlertDialogDescription>
                      </AlertDialogHeader>
                      <AlertDialogFooter>
                        <AlertDialogCancel className="bg-zinc-900 border-zinc-800 hover:bg-zinc-800 hover:text-white">Cancel</AlertDialogCancel>
                        <AlertDialogAction className="bg-red-500 text-white hover:bg-red-600">Disconnect</AlertDialogAction>
                      </AlertDialogFooter>
                    </AlertDialogContent>
                  </AlertDialog>
                </div>
              </div>

              {/* Telegram Card */}
              <div className="flex flex-col border border-zinc-800/80 rounded-xl bg-[#09090b] overflow-hidden">
                <div className="p-5 flex-1 flex flex-col">
                  <div className="flex items-start justify-between mb-4">
                    <div className="flex items-center gap-3">
                      <div className="h-10 w-10 rounded-lg bg-[#0088cc]/10 flex items-center justify-center border border-[#0088cc]/20">
                        <LinkIcon className="h-5 w-5 text-[#0088cc]" />
                      </div>
                      <div>
                        <h4 className="font-medium text-zinc-200">Telegram</h4>
                        <div className="text-xs text-zinc-500 mt-0.5">Bot API</div>
                      </div>
                    </div>
                    
                    <div className="flex items-center gap-1.5 px-2 py-1 rounded-md bg-zinc-800/50 border border-zinc-700/50">
                      <span className="relative flex h-1.5 w-1.5">
                        <span className="relative inline-flex rounded-full h-1.5 w-1.5 bg-zinc-500"></span>
                      </span>
                      <span className="text-[10px] font-medium text-zinc-400 uppercase tracking-wider">Offline</span>
                    </div>
                  </div>
                  
                  <div className="mt-auto space-y-2">
                    <Label className="text-xs text-zinc-500">Bot Token</Label>
                    <Input type="password" placeholder="e.g. 123456789:ABCdefGHIjkl..." className="bg-zinc-950 border-zinc-800 text-xs h-9 focus-visible:ring-1 focus-visible:ring-zinc-700 placeholder:text-zinc-700" />
                  </div>
                </div>
                
                <div className="border-t border-zinc-800/80 p-3 bg-zinc-900/30 flex">
                  <Button variant="ghost" size="sm" className="w-full bg-zinc-800/50 hover:bg-zinc-800 text-zinc-300 h-8 text-xs">
                    Test & Connect
                  </Button>
                </div>
              </div>
            </div>

            <div className="pt-6">
              <div className="mb-4">
                <h4 className="text-base font-medium text-zinc-200">Active Web Sessions</h4>
                <p className="text-xs text-zinc-500">Other devices currently logged in to your account.</p>
              </div>
              <div className="grid grid-cols-1 gap-3">
                <div className="flex items-center justify-between p-4 border border-zinc-800/80 rounded-xl bg-zinc-900/20">
                  <div className="flex items-center gap-4">
                    <div className="h-10 w-10 rounded-full bg-zinc-800 flex items-center justify-center shrink-0">
                      <Globe className="h-5 w-5 text-zinc-400" />
                    </div>
                    <div>
                      <div className="text-sm font-medium text-zinc-200 flex items-center gap-2">
                        Chrome on Linux
                        <span className="text-[10px] uppercase font-bold bg-emerald-500/10 text-emerald-400 px-1.5 py-0.5 rounded tracking-wider">Current</span>
                      </div>
                      <div className="text-xs text-zinc-500 mt-0.5">Jakarta, ID • Active now</div>
                    </div>
                  </div>
                </div>

                <div className="flex items-center justify-between p-4 border border-zinc-800/80 rounded-xl bg-zinc-900/20 group">
                  <div className="flex items-center gap-4">
                    <div className="h-10 w-10 rounded-full bg-zinc-800 flex items-center justify-center shrink-0">
                      <Smartphone className="h-5 w-5 text-zinc-400" />
                    </div>
                    <div>
                      <div className="text-sm font-medium text-zinc-200">
                        Safari on iPhone
                      </div>
                      <div className="text-xs text-zinc-500 mt-0.5">Unknown Location • Last active 2 hours ago</div>
                    </div>
                  </div>
                  <AlertDialog>
                    <AlertDialogTrigger asChild>
                      <Button variant="ghost" size="sm" className="text-zinc-500 hover:text-red-400 hover:bg-red-500/10 opacity-0 group-hover:opacity-100 transition-opacity">
                        Revoke
                      </Button>
                    </AlertDialogTrigger>
                    <AlertDialogContent className="bg-zinc-950 border-white/10 text-white">
                      <AlertDialogHeader>
                        <AlertDialogTitle>Revoke session?</AlertDialogTitle>
                        <AlertDialogDescription className="text-zinc-400">
                          This will log out Safari on iPhone immediately.
                        </AlertDialogDescription>
                      </AlertDialogHeader>
                      <AlertDialogFooter>
                        <AlertDialogCancel className="bg-zinc-900 border-zinc-800 hover:bg-zinc-800 hover:text-white">Cancel</AlertDialogCancel>
                        <AlertDialogAction className="bg-red-500 text-white hover:bg-red-600">Revoke</AlertDialogAction>
                      </AlertDialogFooter>
                    </AlertDialogContent>
                  </AlertDialog>
                </div>
              </div>
            </div>
          </div>
        );

      case 'storage':
        return (
          <div className="space-y-12 max-w-4xl">
            {/* Section A: Vector DB */}
            <div className="space-y-6">
              <div>
                <h3 className="text-lg font-medium text-white">Vector DB (Qdrant)</h3>
                <p className="text-sm text-zinc-400">Manage semantic memory and document embeddings.</p>
              </div>
              
              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                <div className="border border-zinc-800 rounded-xl p-5 bg-zinc-900/30">
                  <h4 className="text-zinc-400 text-sm font-medium mb-1">Collection: documents</h4>
                  <div className="text-3xl font-light text-white mb-4">1,248 <span className="text-base text-zinc-500 font-normal">vectors</span></div>
                  <div className="flex gap-2">
                    <Button size="sm" variant="outline" className="bg-zinc-900 border-zinc-800 text-zinc-300 hover:bg-zinc-800 hover:text-white w-full">Reindex</Button>
                    <AlertDialog>
                      <AlertDialogTrigger asChild>
                        <Button size="sm" variant="outline" className="bg-zinc-900 border-zinc-800 text-red-400 hover:bg-red-500/10 hover:border-red-500/20 w-full">Clear</Button>
                      </AlertDialogTrigger>
                      <AlertDialogContent className="bg-zinc-950 border-white/10 text-white">
                        <AlertDialogHeader>
                          <AlertDialogTitle>Clear Collection?</AlertDialogTitle>
                          <AlertDialogDescription className="text-zinc-400">
                            This deletes all embedded document vectors. Document files won't be deleted but will need reindexing.
                          </AlertDialogDescription>
                        </AlertDialogHeader>
                        <AlertDialogFooter>
                          <AlertDialogCancel className="bg-zinc-900 border-zinc-800">Cancel</AlertDialogCancel>
                          <AlertDialogAction className="bg-red-500 text-white hover:bg-red-600">Clear Data</AlertDialogAction>
                        </AlertDialogFooter>
                      </AlertDialogContent>
                    </AlertDialog>
                  </div>
                </div>

                <div className="border border-zinc-800 rounded-xl p-5 bg-zinc-900/30">
                  <h4 className="text-zinc-400 text-sm font-medium mb-1">Collection: user_facts</h4>
                  <div className="text-3xl font-light text-white mb-4">42 <span className="text-base text-zinc-500 font-normal">vectors</span></div>
                  <div className="flex gap-2">
                    <Button size="sm" variant="outline" className="bg-zinc-900 border-zinc-800 text-zinc-300 hover:bg-zinc-800 hover:text-white w-full">Reindex</Button>
                    <AlertDialog>
                      <AlertDialogTrigger asChild>
                        <Button size="sm" variant="outline" className="bg-zinc-900 border-zinc-800 text-red-400 hover:bg-red-500/10 hover:border-red-500/20 w-full">Clear</Button>
                      </AlertDialogTrigger>
                      <AlertDialogContent className="bg-zinc-950 border-white/10 text-white">
                        <AlertDialogHeader>
                          <AlertDialogTitle>Clear Collection?</AlertDialogTitle>
                          <AlertDialogDescription className="text-zinc-400">
                            This deletes all AI-learned facts about you.
                          </AlertDialogDescription>
                        </AlertDialogHeader>
                        <AlertDialogFooter>
                          <AlertDialogCancel className="bg-zinc-900 border-zinc-800">Cancel</AlertDialogCancel>
                          <AlertDialogAction className="bg-red-500 text-white hover:bg-red-600">Clear Data</AlertDialogAction>
                        </AlertDialogFooter>
                      </AlertDialogContent>
                    </AlertDialog>
                  </div>
                </div>
              </div>
            </div>

            {/* Section B: Document Library */}
            <div className="space-y-6">
              <div className="flex items-center justify-between">
                <div>
                  <h3 className="text-lg font-medium text-white">Document Library</h3>
                  <p className="text-sm text-zinc-400">Manage uploaded files and their indexing status.</p>
                </div>
              </div>
              <div className="border border-zinc-800 rounded-lg overflow-hidden">
                <div className="p-3 bg-zinc-900/50 border-b border-zinc-800">
                  <Input placeholder="Search documents..." className="h-8 text-xs bg-zinc-900 border-zinc-700 w-64" />
                </div>
                <Table>
                  <TableHeader className="bg-zinc-900/30">
                    <TableRow className="border-zinc-800">
                      <TableHead className="text-zinc-400">File Name</TableHead>
                      <TableHead className="text-zinc-400">Date</TableHead>
                      <TableHead className="text-zinc-400">Chunks</TableHead>
                      <TableHead className="text-zinc-400">Source</TableHead>
                      <TableHead className="text-right text-zinc-400">Actions</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {/* TODO: connect to real API */}
                    <TableRow className="border-zinc-800/50">
                      <TableCell className="text-zinc-300 font-medium flex items-center gap-2">
                        <FileText className="h-4 w-4 text-zinc-500" /> report-2024.pdf
                      </TableCell>
                      <TableCell className="text-zinc-500 text-xs">2026-08-20</TableCell>
                      <TableCell className="text-zinc-500 text-xs">45</TableCell>
                      <TableCell className="text-zinc-500 text-xs underline cursor-pointer hover:text-zinc-300">Thread 12</TableCell>
                      <TableCell className="text-right">
                        <Button variant="ghost" size="sm" className="h-7 text-xs text-zinc-400 hover:text-white mr-1">Reindex</Button>
                        <AlertDialog>
                          <AlertDialogTrigger asChild>
                            <Button variant="ghost" size="icon" className="h-7 w-7 text-zinc-500 hover:text-red-400 hover:bg-red-500/10">
                              <Trash2 className="h-3.5 w-3.5" />
                            </Button>
                          </AlertDialogTrigger>
                          <AlertDialogContent className="bg-zinc-950 border-white/10 text-white">
                            <AlertDialogHeader>
                              <AlertDialogTitle>Delete document?</AlertDialogTitle>
                              <AlertDialogDescription className="text-zinc-400">
                                This will remove the file from storage and its vectors from Qdrant.
                              </AlertDialogDescription>
                            </AlertDialogHeader>
                        <AlertDialogFooter>
                          <AlertDialogCancel className="bg-zinc-900 border-zinc-800">Cancel</AlertDialogCancel>
                          <AlertDialogAction 
                            onClick={handlePurgeDocuments}
                            className="bg-red-500 text-white hover:bg-red-600"
                          >
                            Clear Data
                          </AlertDialogAction>
                        </AlertDialogFooter>
                          </AlertDialogContent>
                        </AlertDialog>
                      </TableCell>
                    </TableRow>
                  </TableBody>
                </Table>
              </div>
            </div>

            {/* Section C: Threads DB */}
            <div className="space-y-6">
              <div>
                <h3 className="text-lg font-medium text-white">Threads Database (SQLite)</h3>
                <p className="text-sm text-zinc-400">Manage chat history storage.</p>
              </div>
              <div className="flex items-center gap-4 bg-zinc-900/30 p-4 border border-zinc-800 rounded-xl">
                <Database className="h-8 w-8 text-zinc-500" />
                <div className="flex-1">
                  <div className="text-zinc-200 font-medium">pribadi.db</div>
                  <div className="text-xs text-zinc-500">Size: 14.2 MB • WAL Mode: Enabled</div>
                </div>
                <div className="flex gap-2">
                  <Button variant="outline" size="sm" className="bg-zinc-900 border-zinc-800 text-zinc-300 hover:bg-zinc-800 hover:text-white">
                    Export Backup
                  </Button>
                  <AlertDialog>
                    <AlertDialogTrigger asChild>
                      <Button variant="outline" size="sm" className="bg-zinc-900 border-zinc-800 text-red-400 hover:bg-red-500/10 hover:border-red-500/20">
                        Bulk Delete Old
                      </Button>
                    </AlertDialogTrigger>
                    <AlertDialogContent className="bg-zinc-950 border-white/10 text-white">
                      <AlertDialogHeader>
                        <AlertDialogTitle>Bulk Delete Threads</AlertDialogTitle>
                        <AlertDialogDescription className="text-zinc-400">
                          Delete threads older than 30 days?
                        </AlertDialogDescription>
                      </AlertDialogHeader>
                        <AlertDialogFooter>
                          <AlertDialogCancel className="bg-zinc-900 border-zinc-800">Cancel</AlertDialogCancel>
                          <AlertDialogAction 
                            onClick={(e) => { e.preventDefault(); handlePurgeDocuments(); }}
                            className="bg-red-500 text-white hover:bg-red-600"
                          >
                            Clear Data
                          </AlertDialogAction>
                        </AlertDialogFooter>
                    </AlertDialogContent>
                  </AlertDialog>
                </div>
              </div>
            </div>

            {/* Section D: Danger Zone */}
            <div className="space-y-6 pt-12 mt-12 border-t border-red-500/20">
              <div>
                <h3 className="text-lg font-medium text-red-400">Danger Zone</h3>
                <p className="text-sm text-zinc-500">Irreversible actions.</p>
              </div>
              <div className="flex items-center justify-between p-4 border border-red-500/20 rounded-xl bg-red-500/5">
                <div>
                  <div className="font-medium text-zinc-200">Delete all data</div>
                  <div className="text-xs text-zinc-500 mt-1">This will permanently delete your account, all threads, vectors, and uploaded files.</div>
                </div>
                <AlertDialog>
                  <AlertDialogTrigger asChild>
                    <Button variant="destructive" className="bg-red-500 hover:bg-red-600 text-white font-medium shadow-none">
                      Delete Everything
                    </Button>
                  </AlertDialogTrigger>
                  <AlertDialogContent className="bg-zinc-950 border-red-500/30 text-white">
                    <AlertDialogHeader>
                      <AlertDialogTitle className="text-red-400">Are you absolutely sure?</AlertDialogTitle>
                      <AlertDialogDescription className="text-zinc-400">
                        This action cannot be undone. This will permanently delete your
                        account and remove your data from our servers.
                      </AlertDialogDescription>
                    </AlertDialogHeader>
                    <AlertDialogFooter>
                      <AlertDialogCancel className="bg-zinc-900 border-zinc-800">Cancel</AlertDialogCancel>
                      <AlertDialogAction className="bg-red-500 text-white hover:bg-red-600 font-bold">Yes, delete my account</AlertDialogAction>
                    </AlertDialogFooter>
                  </AlertDialogContent>
                </AlertDialog>
              </div>
            </div>
          </div>
        );

      default:
        return null;
    }
  };

  return (
    <div className="flex h-screen bg-black text-white selection:bg-white/20 overflow-hidden">
      <Sidebar />
      <div className="flex-1 flex flex-col min-w-0">
        <header className="h-16 flex items-center px-8 border-b border-white/[0.02] bg-transparent">
          <h1 className="text-lg font-semibold text-white">Settings</h1>
        </header>
        
        <div className="flex-1 flex overflow-hidden">
          {/* Settings Nav */}
          <div className="w-64 border-r border-white/[0.02] p-6 flex flex-col gap-1 overflow-y-auto">
            <div className="text-xs font-semibold text-zinc-500 uppercase tracking-wider mb-3 mt-2 px-2">Account</div>
            
            <button
              onClick={() => setActiveTab('preferences')}
              className={`flex items-center gap-3 px-3 py-2 rounded-lg text-sm transition-colors ${
                activeTab === 'preferences' 
                  ? 'bg-zinc-800/80 text-white font-medium shadow-sm' 
                  : 'text-zinc-400 hover:bg-zinc-900 hover:text-zinc-200'
              }`}
            >
              <User className="h-4 w-4" />
              Preferences
            </button>
            
            <button
              onClick={() => setActiveTab('personalize')}
              className={`flex items-center gap-3 px-3 py-2 rounded-lg text-sm transition-colors ${
                activeTab === 'personalize' 
                  ? 'bg-zinc-800/80 text-white font-medium shadow-sm' 
                  : 'text-zinc-400 hover:bg-zinc-900 hover:text-zinc-200'
              }`}
            >
              <Paintbrush className="h-4 w-4" />
              Personalize
            </button>

            <div className="text-xs font-semibold text-zinc-500 uppercase tracking-wider mb-3 mt-8 px-2">System</div>
            
            <button
              onClick={() => setActiveTab('models')}
              className={`flex items-center gap-3 px-3 py-2 rounded-lg text-sm transition-colors ${
                activeTab === 'models' 
                  ? 'bg-zinc-800/80 text-white font-medium shadow-sm' 
                  : 'text-zinc-400 hover:bg-zinc-900 hover:text-zinc-200'
              }`}
            >
              <Cpu className="h-4 w-4" />
              Models
            </button>
            
            <div className="text-xs font-semibold text-zinc-500 uppercase tracking-wider mb-3 mt-8 px-2">Integrations</div>

            <button
              onClick={() => setActiveTab('connections')}
              className={`flex items-center gap-3 px-3 py-2 rounded-lg text-sm transition-colors ${
                activeTab === 'connections' 
                  ? 'bg-zinc-800/80 text-white font-medium shadow-sm' 
                  : 'text-zinc-400 hover:bg-zinc-900 hover:text-zinc-200'
              }`}
            >
              <LinkIcon className="h-4 w-4" />
              Connections
            </button>

            <div className="text-xs font-semibold text-zinc-500 uppercase tracking-wider mb-3 mt-8 px-2">Data & Privacy</div>
            
            <button
              onClick={() => setActiveTab('storage')}
              className={`flex items-center gap-3 px-3 py-2 rounded-lg text-sm transition-colors ${
                activeTab === 'storage' 
                  ? 'bg-zinc-800/80 text-white font-medium shadow-sm' 
                  : 'text-zinc-400 hover:bg-zinc-900 hover:text-zinc-200'
              }`}
            >
              <Database className="h-4 w-4" />
              Storage
            </button>

            <div className="mt-auto pt-6">
              <button
                onClick={handleLogout}
                className="w-full flex items-center gap-3 px-3 py-2 rounded-lg text-sm text-zinc-400 hover:bg-red-500/10 hover:text-red-400 transition-colors"
              >
                <LogOut className="h-4 w-4" />
                Logout
              </button>
            </div>
          </div>
          
          {/* Content Area */}
          <div className="flex-1 overflow-y-auto px-8 md:px-12 py-10">
            {renderTabContent()}
          </div>
        </div>
      </div>
    </div>
  );
}
