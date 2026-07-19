import { useState } from 'react';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import * as z from 'zod';
import { 
  Bot, 
  MessageSquare, 
  Settings2, 
  Info, 
  Save,
  Loader2
} from 'lucide-react';
import { Toast } from '../components/ui/Toast';

// --- Schema Validation ---
const configSchema = z.object({
  botName: z.string().min(2, 'Nama harus minimal 2 karakter').max(50, 'Nama maksimal 50 karakter'),
  persona: z.enum(['formal', 'santai', 'ceria', 'profesional']),
  systemPrompt: z.string().max(2000, 'Instruksi maksimal 2000 karakter').optional(),
  autoReply: z.boolean(),
});

type ConfigFormValues = z.infer<typeof configSchema>;

export default function AgentConfig() {
  const [isSaving, setIsSaving] = useState(false);
  const [showToast, setShowToast] = useState(false);

  // Initialize React Hook Form
  const {
    register,
    handleSubmit,
    watch,
    formState: { errors, isDirty },
  } = useForm<ConfigFormValues>({
    resolver: zodResolver(configSchema),
    defaultValues: {
      botName: 'Asisten BisnisKu',
      persona: 'profesional',
      systemPrompt: 'Kamu adalah asisten AI untuk toko pakaian "GayaKini". Jawab pertanyaan pelanggan dengan sopan. Jam buka toko adalah 08:00 - 17:00.',
      autoReply: true,
    }
  });

  const autoReplyState = watch('autoReply');

  const onSubmit = (data: ConfigFormValues) => {
    setIsSaving(true);
    // Simulate API Call
    setTimeout(() => {
      console.log('Saved Config:', data);
      setIsSaving(false);
      setShowToast(true);
    }, 1500);
  };

  return (
    <div className="space-y-6 animate-in fade-in duration-500 pb-24 lg:pb-8">
      
      {/* Header */}
      <div>
        <h1 className="font-sora text-2xl font-bold text-ink-primary mb-2">Konfigurasi Agen AI</h1>
        <p className="text-ink-muted">
          Atur identitas, kepribadian, dan cara AI merespons pesan pengguna.
        </p>
      </div>

      <form onSubmit={handleSubmit(onSubmit)} className="space-y-6">
        
        {/* 2-Column Layout */}
        <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
          
          {/* Main Column (2/3 width on desktop) */}
          <div className="lg:col-span-2 space-y-6">
            
            {/* Identitas AI */}
            <div className="bg-surface border border-ink-primary/10 rounded-2xl p-6 shadow-sm">
              <div className="flex items-center gap-2 text-ink-primary font-bold mb-6">
                <Bot size={20} className="text-accent-primary" />
                <h2>Identitas AI</h2>
              </div>
              
              <div className="space-y-5">
                <div>
                  <label className="block text-sm font-semibold text-ink-primary mb-1.5 ml-1">Nama Asisten</label>
                  <input
                    {...register('botName')}
                    type="text"
                    className="w-full bg-canvas border border-ink-primary/10 rounded-xl px-4 py-2.5 text-ink-primary focus:bg-surface focus:ring-2 focus:ring-accent-primary/15 focus:border-accent-primary transition-all outline-none"
                    placeholder="Contoh: CS Toko, Jojo, AI Assistant..."
                  />
                  {errors.botName && (
                    <p className="text-rose-500 text-sm mt-1 ml-1">{errors.botName.message}</p>
                  )}
                </div>

                <div>
                  <label className="block text-sm font-semibold text-ink-primary mb-1.5 ml-1">Gaya Bahasa</label>
                  <select
                    {...register('persona')}
                    className="w-full bg-canvas border border-ink-primary/10 rounded-xl px-4 py-2.5 text-ink-primary focus:bg-surface focus:ring-2 focus:ring-accent-primary/15 focus:border-accent-primary transition-all outline-none appearance-none"
                  >
                    <option value="profesional">Profesional (Tegas & Langsung ke inti)</option>
                    <option value="formal">Formal (Baku & Resmi)</option>
                    <option value="santai">Santai (Kasual & Ramah)</option>
                    <option value="ceria">Ceria (Ekspresif & Menggunakan Emoji)</option>
                  </select>
                </div>
              </div>
            </div>

            {/* Perilaku Inti */}
            <div className="bg-surface border border-ink-primary/10 rounded-2xl p-6 shadow-sm">
              <div className="flex items-center gap-2 text-ink-primary font-bold mb-6">
                <MessageSquare size={20} className="text-accent-primary" />
                <h2>Perilaku Inti (System Prompt)</h2>
              </div>

              <div>
                <div className="flex items-center justify-between mb-1.5 ml-1">
                  <label className="text-sm font-semibold text-ink-primary">Instruksi Khusus</label>
                  <div className="group relative cursor-help">
                    <Info size={16} className="text-ink-muted/70 hover:text-ink-muted" />
                    <div className="absolute bottom-full right-0 mb-2 w-64 bg-brand-900 text-brand-50 text-xs p-3 rounded-lg opacity-0 group-hover:opacity-100 transition-opacity pointer-events-none shadow-xl z-10">
                      Berikan konteks tambahan seperti jam buka, kebijakan refund, atau cara AI harus menjawab jika ia tidak mengetahui jawabannya.
                    </div>
                  </div>
                </div>
                
                <textarea
                  {...register('systemPrompt')}
                  rows={8}
                  className="w-full bg-canvas border border-ink-primary/10 rounded-xl px-4 py-3 text-ink-primary focus:bg-surface focus:ring-2 focus:ring-accent-primary/15 focus:border-accent-primary transition-all outline-none resize-y"
                  placeholder="Ceritakan siapa AI ini dan apa tugas utamanya..."
                />
                <div className="flex justify-between items-center mt-1 ml-1">
                  {errors.systemPrompt ? (
                    <p className="text-rose-500 text-sm">{errors.systemPrompt.message}</p>
                  ) : (
                    <p className="text-ink-muted/70 text-xs">Maksimal 2000 karakter</p>
                  )}
                </div>
              </div>
            </div>

          </div>

          {/* Sidebar Column (1/3 width on desktop) */}
          <div className="space-y-6">
            
            {/* Preferensi Sistem */}
            <div className="bg-surface border border-ink-primary/10 rounded-2xl p-6 shadow-sm">
              <div className="flex items-center gap-2 text-ink-primary font-bold mb-6">
                <Settings2 size={20} className="text-accent-primary" />
                <h2>Preferensi Sistem</h2>
              </div>

              <div className="flex items-start justify-between gap-4 p-4 bg-canvas rounded-xl border border-brand-100">
                <div>
                  <h3 className="font-sora text-sm font-semibold text-ink-primary">Balas Otomatis</h3>
                  <p className="text-xs text-ink-muted mt-1 leading-relaxed">
                    Izinkan AI untuk langsung membalas pesan pengguna.
                  </p>
                </div>
                <label className="relative inline-flex items-center cursor-pointer mt-1 shrink-0">
                  <input 
                    type="checkbox" 
                    {...register('autoReply')}
                    className="sr-only peer" 
                  />
                  <div className="w-11 h-6 bg-brand-200 peer-focus:outline-none rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-surface after:border-ink-primary/20 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-accent-primary"></div>
                </label>
              </div>

              {!autoReplyState && (
                <div className="mt-4 p-3 bg-amber-50 border border-accent-warm/30 rounded-xl">
                  <p className="text-xs text-accent-warm font-medium">
                    ⚠️ AI tidak akan membalas pesan. Ia hanya akan membuat draft balasan di dashboard (fitur ini sedang dalam pengembangan).
                  </p>
                </div>
              )}
            </div>

          </div>
        </div>

        {/* Action Bottom Bar */}
        <div className="fixed bottom-0 left-0 right-0 lg:left-64 bg-surface border-t border-ink-primary/10 p-4 px-6 flex justify-end shadow-[0_-4px_6px_-1px_rgba(0,0,0,0.05)] z-20">
          <button
            type="submit"
            disabled={!isDirty || isSaving}
            className="flex items-center gap-2 bg-accent-primary text-white px-6 py-2.5 rounded-xl font-medium hover:bg-accent-primary/90 active:scale-[0.98] transition-all disabled:opacity-50 disabled:cursor-not-allowed shadow-sm"
          >
            {isSaving ? (
              <Loader2 size={18} className="animate-spin" />
            ) : (
              <Save size={18} />
            )}
            <span>{isSaving ? 'Menyimpan...' : 'Simpan Konfigurasi'}</span>
          </button>
        </div>

      </form>

      <Toast 
        message="Konfigurasi disimpan. AI akan beradaptasi di pesan berikutnya." 
        isVisible={showToast} 
        onClose={() => setShowToast(false)} 
      />

    </div>
  );
}
