"use client";

import { useState } from 'react';
import { useMonitorStore } from '@/store/monitor';
import { Dialog, DialogContent, DialogHeader, DialogTitle } from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { Textarea } from '@/components/ui/textarea';

interface Props {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

export function MonitorFormDialog({ open, onOpenChange }: Props) {
  const { createTask, testExtract, testDeriveRule } = useMonitorStore();
  
  const [step, setStep] = useState(1);
  const [formData, setFormData] = useState({
    name: '',
    category: '',
    source_type: 'http_api',
    condition_mode: 'structural',
    poll_interval_seconds: 300,
  });
  
  // Source config state
  const [httpConfig, setHttpConfig] = useState({ url: '', method: 'GET', extract_path: '' });
  const [scrapeConfig, setScrapeConfig] = useState({ url: '', description: '' });
  
  // Condition config state
  const [structuralConfig, setStructuralConfig] = useState({ operator: 'above', condition_value: 0 });
  const [nlConfig, setNlConfig] = useState({ condition_prompt: '' });
  
  const [testResult, setTestResult] = useState<any>(null);
  const [isTesting, setIsTesting] = useState(false);

  const handleTest = async () => {
    setIsTesting(true);
    setTestResult(null);
    try {
      if (formData.source_type === 'http_api') {
        const res = await testExtract(httpConfig);
        setTestResult(res);
      } else {
        const res = await testDeriveRule(scrapeConfig.url, scrapeConfig.description);
        setTestResult(res);
      }
    } catch (e: any) {
      setTestResult({ success: false, error: e.message });
    }
    setIsTesting(false);
  };

  const handleSave = async () => {
    const payload = {
      ...formData,
      platform: 'whatsapp',
      source_config: JSON.stringify(formData.source_type === 'http_api' ? httpConfig : scrapeConfig),
      ...(formData.condition_mode === 'structural' ? structuralConfig : nlConfig),
    };
    await createTask(payload);
    onOpenChange(false);
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-2xl">
        <DialogHeader>
          <DialogTitle>Create New Monitor</DialogTitle>
        </DialogHeader>

        <div className="space-y-4 py-4">
          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-2">
              <Label>Name</Label>
              <Input value={formData.name} onChange={e => setFormData({...formData, name: e.target.value})} placeholder="e.g. AAPL Stock Price" />
            </div>
            <div className="space-y-2">
              <Label>Category (Optional)</Label>
              <Input value={formData.category} onChange={e => setFormData({...formData, category: e.target.value})} placeholder="e.g. stock, competitor" />
            </div>
          </div>

          <div className="space-y-2">
            <Label>Source Type</Label>
            <Select value={formData.source_type} onValueChange={v => setFormData({...formData, source_type: v})}>
              <SelectTrigger><SelectValue /></SelectTrigger>
              <SelectContent>
                <SelectItem value="http_api">JSON API (HTTP)</SelectItem>
                <SelectItem value="web_scrape">Web Scrape (AI Rule Generation)</SelectItem>
              </SelectContent>
            </Select>
          </div>

          {/* SOURCE CONFIG */}
          <div className="p-4 border rounded bg-slate-50 dark:bg-slate-900">
            {formData.source_type === 'http_api' ? (
              <div className="space-y-3">
                <Label>API URL</Label>
                <Input value={httpConfig.url} onChange={e => setHttpConfig({...httpConfig, url: e.target.value})} placeholder="https://api.example.com/data" />
                <Label>JSON Extract Path (GJSON syntax)</Label>
                <Input value={httpConfig.extract_path} onChange={e => setHttpConfig({...httpConfig, extract_path: e.target.value})} placeholder="data.0.price" />
              </div>
            ) : (
              <div className="space-y-3">
                <Label>Website URL</Label>
                <Input value={scrapeConfig.url} onChange={e => setScrapeConfig({...scrapeConfig, url: e.target.value})} placeholder="https://example.com/product" />
                <Label>What to extract (Natural Language)</Label>
                <Textarea value={scrapeConfig.description} onChange={e => setScrapeConfig({...scrapeConfig, description: e.target.value})} placeholder="The price of the product next to the 'Buy' button" />
              </div>
            )}
            
            <div className="mt-4">
              <Button variant="secondary" onClick={handleTest} disabled={isTesting}>
                {isTesting ? 'Testing...' : formData.source_type === 'http_api' ? 'Test Extract' : 'Test & Derive Rule'}
              </Button>
              {testResult && (
                <pre className="mt-2 text-xs p-2 bg-black text-green-400 rounded max-h-32 overflow-auto">
                  {JSON.stringify(testResult, null, 2)}
                </pre>
              )}
            </div>
          </div>

          {/* CONDITION CONFIG */}
          <div className="space-y-2 mt-4">
            <Label>Condition Mode</Label>
            <Select value={formData.condition_mode} onValueChange={v => {
              setFormData({...formData, condition_mode: v, poll_interval_seconds: v === 'nl_judge' ? 3600 : 300})
            }}>
              <SelectTrigger><SelectValue /></SelectTrigger>
              <SelectContent>
                <SelectItem value="structural">Structural (Numeric/Text compare, Free)</SelectItem>
                <SelectItem value="nl_judge">AI Judge (Natural Language, LLM cost)</SelectItem>
              </SelectContent>
            </Select>
          </div>

          <div className="p-4 border rounded bg-slate-50 dark:bg-slate-900">
            {formData.condition_mode === 'structural' ? (
              <div className="flex gap-4">
                <div className="flex-1 space-y-2">
                  <Label>Operator</Label>
                  <Select value={structuralConfig.operator} onValueChange={v => setStructuralConfig({...structuralConfig, operator: v})}>
                    <SelectTrigger><SelectValue /></SelectTrigger>
                    <SelectContent>
                      <SelectItem value="above">Above (&gt;)</SelectItem>
                      <SelectItem value="below">Below (&lt;)</SelectItem>
                      <SelectItem value="percent_change">Percent Change (±%)</SelectItem>
                    </SelectContent>
                  </Select>
                </div>
                <div className="flex-1 space-y-2">
                  <Label>Value</Label>
                  <Input type="number" value={structuralConfig.condition_value} onChange={e => setStructuralConfig({...structuralConfig, condition_value: parseFloat(e.target.value)})} />
                </div>
              </div>
            ) : (
              <div className="space-y-2">
                <Label>Condition Prompt (Tell AI when to notify you)</Label>
                <Textarea value={nlConfig.condition_prompt} onChange={e => setNlConfig({...nlConfig, condition_prompt: e.target.value})} placeholder="Notify me if the product is 'Out of Stock' or if the price drops below $50" />
                <p className="text-xs text-yellow-600 dark:text-yellow-500">
                  ⚠️ This uses the LLM every {formData.poll_interval_seconds / 60} minutes. Daily budget applies.
                </p>
              </div>
            )}
          </div>
          
          <div className="flex justify-end pt-4">
            <Button onClick={handleSave} disabled={!formData.name || (formData.source_type==='http_api'&&!httpConfig.url)}>
              Save Monitor
            </Button>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  );
}