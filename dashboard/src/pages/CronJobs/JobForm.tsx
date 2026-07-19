import { useState } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import { useForm } from 'react-hook-form';
import { ArrowLeft, Save, CalendarClock, Smartphone, User, Loader2 } from 'lucide-react';

interface JobFormValues {
  name: string;
  frequency: string;
  dateTime: string;
  targetJid: string;
  message: string;
}

export default function JobForm() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const [isSaving, setIsSaving] = useState(false);

  const {
    register,
    handleSubmit,
    watch,
    formState: { errors }
  } = useForm<JobFormValues>({
    defaultValues: {
      name: id ? 'Follow-up Pembayaran' : '',
      frequency: 'once',
      dateTime: '',
      targetJid: '',
      message: 'Halo, saya asisten dari Toko X. Apakah ada yang bisa kami bantu hari ini?'
    }
  });

  const messageText = watch('message');

  const onSubmit = (data: JobFormValues) => {
    setIsSaving(true);
    // Simulate API save
    setTimeout(() => {
      console.log('Saved Job:', data);
      setIsSaving(false);
      navigate('/schedules');
    }, 1000);
  };

  return (
    <div className="space-y-6 animate-in fade-in duration-500 pb-24 lg:pb-8">
      
      {/* Header */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
        <div className="flex items-center gap-3">
          <button 
            onClick={() => navigate('/schedules')}
            className="p-2 text-ink-muted/70 hover:text-ink-primary bg-surface border border-ink-primary/10 rounded-xl hover:bg-canvas transition-colors shadow-sm"
          >
            <ArrowLeft size={20} />
          </button>
          <div>
            <h1 className="font-sora text-2xl font-bold text-ink-primary">{id ? 'Edit Jadwal' : 'Buat Jadwal Baru'}</h1>
            <p className="text-sm text-ink-muted">Atur pesan yang akan dikirim otomatis oleh AI.</p>
          </div>
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6 items-start">
        
        {/* Main Form Column */}
        <div className="lg:col-span-2 space-y-6">
          <form id="job-form" onSubmit={handleSubmit(onSubmit)} className="bg-surface border border-ink-primary/10 rounded-2xl p-6 shadow-sm space-y-6">
            
            <div className="flex items-center gap-2 text-ink-primary font-bold mb-2">
              <CalendarClock size={20} className="text-accent-primary" />
              <h2>Pengaturan Jadwal</h2>
            </div>

            <div className="grid grid-cols-1 md:grid-cols-2 gap-5">
              <div>
                <label className="block text-sm font-semibold text-ink-primary mb-1.5 ml-1">Nama Tugas</label>
                <input
                  {...register('name', { required: 'Nama tugas wajib diisi' })}
                  type="text"
                  placeholder="Misal: Tagihan SPP Bulanan"
                  className="w-full bg-canvas border border-ink-primary/10 rounded-xl px-4 py-2.5 text-ink-primary focus:bg-surface focus:ring-2 focus:ring-accent-primary/15 focus:border-accent-primary transition-all outline-none"
                />
                {errors.name && <p className="text-rose-500 text-xs mt-1 ml-1">{errors.name.message}</p>}
              </div>

              <div>
                <label className="block text-sm font-semibold text-ink-primary mb-1.5 ml-1">Frekuensi Pengiriman</label>
                <select
                  {...register('frequency')}
                  className="w-full bg-canvas border border-ink-primary/10 rounded-xl px-4 py-2.5 text-ink-primary focus:bg-surface focus:ring-2 focus:ring-accent-primary/15 focus:border-accent-primary transition-all outline-none appearance-none"
                >
                  <option value="once">Satu Kali</option>
                  <option value="daily">Harian</option>
                  <option value="weekly">Mingguan</option>
                  <option value="monthly">Bulanan</option>
                </select>
              </div>
            </div>

            <div className="grid grid-cols-1 md:grid-cols-2 gap-5 border-b border-brand-100 pb-6">
              <div>
                <label className="block text-sm font-semibold text-ink-primary mb-1.5 ml-1">Waktu Pelaksanaan</label>
                <input
                  {...register('dateTime', { required: 'Waktu wajib diisi' })}
                  type="datetime-local"
                  className="w-full bg-canvas border border-ink-primary/10 rounded-xl px-4 py-2.5 text-ink-primary focus:bg-surface focus:ring-2 focus:ring-accent-primary/15 focus:border-accent-primary transition-all outline-none"
                />
                {errors.dateTime && <p className="text-rose-500 text-xs mt-1 ml-1">{errors.dateTime.message}</p>}
              </div>

              <div>
                <label className="block text-sm font-semibold text-ink-primary mb-1.5 ml-1">Target Nomor (WA)</label>
                <input
                  {...register('targetJid', { required: 'Nomor wajib diisi' })}
                  type="text"
                  placeholder="081234..."
                  className="w-full bg-canvas border border-ink-primary/10 rounded-xl px-4 py-2.5 text-ink-primary focus:bg-surface focus:ring-2 focus:ring-accent-primary/15 focus:border-accent-primary transition-all outline-none"
                />
                {errors.targetJid && <p className="text-rose-500 text-xs mt-1 ml-1">{errors.targetJid.message}</p>}
              </div>
            </div>

            <div>
              <label className="block text-sm font-semibold text-ink-primary mb-1.5 ml-1">Pesan Utama (Atau Instruksi AI)</label>
              <p className="text-xs text-ink-muted mb-3 ml-1">
                Ketik pesan yang akan dikirim, atau instruksikan AI untuk merangkai pesannya sendiri berdasarkan pedoman (system prompt) yang ada.
              </p>
              <textarea
                {...register('message', { required: 'Pesan tidak boleh kosong' })}
                rows={6}
                className="w-full bg-canvas border border-ink-primary/10 rounded-xl px-4 py-3 text-ink-primary focus:bg-surface focus:ring-2 focus:ring-accent-primary/15 focus:border-accent-primary transition-all outline-none resize-y"
                placeholder="Halo, ini adalah pengingat untuk..."
              />
              {errors.message && <p className="text-rose-500 text-xs mt-1 ml-1">{errors.message.message}</p>}
            </div>

            <div className="flex justify-end pt-4">
              <button
                type="submit"
                disabled={isSaving}
                className="flex items-center gap-2 bg-accent-primary text-white px-6 py-2.5 rounded-xl font-medium hover:bg-accent-primary/90 transition-colors disabled:opacity-50 disabled:cursor-not-allowed shadow-sm w-full md:w-auto"
              >
                {isSaving ? <Loader2 size={18} className="animate-spin" /> : <Save size={18} />}
                <span>{isSaving ? 'Menyimpan...' : 'Simpan Jadwal'}</span>
              </button>
            </div>
          </form>
        </div>

        {/* Sidebar Column: Live Preview */}
        <div className="sticky top-24">
          <div className="bg-slate-100 border-[8px] border-slate-800 rounded-[2.5rem] h-[550px] w-full max-w-[320px] mx-auto overflow-hidden flex flex-col shadow-2xl relative">
            {/* Phone Notch/Header */}
            <div className="bg-slate-800 h-6 w-full absolute top-0 z-10 flex justify-center">
              <div className="w-24 h-4 bg-black rounded-b-xl"></div>
            </div>
            
            {/* WhatsApp App Header Mock */}
            <div className="bg-[#075E54] pt-8 pb-3 px-4 flex items-center gap-3 text-white shadow-md z-0">
              <ArrowLeft size={18} />
              <div className="w-8 h-8 bg-surface/20 rounded-full flex items-center justify-center">
                <User size={16} />
              </div>
              <div>
                <div className="font-semibold text-sm">Pelanggan</div>
                <div className="text-[10px] opacity-80">Terakhir dilihat hari ini 12:00</div>
              </div>
            </div>

            {/* Chat Area */}
            <div className="flex-1 bg-[#E5DDD5] p-4 flex flex-col justify-end overflow-y-auto">
              <div className="w-full flex justify-end">
                <div className="bg-[#DCF8C6] p-3 rounded-xl rounded-tr-none shadow-sm max-w-[85%] text-sm text-slate-800 relative">
                  <span className="whitespace-pre-wrap word-break">
                    {messageText || <span className="italic opacity-50">Menunggu pesan...</span>}
                  </span>
                  <div className="text-right text-[10px] text-accent-primary/60 mt-1">10:00</div>
                  <div className="absolute top-0 -right-2 w-0 h-0 border-t-[10px] border-t-[#DCF8C6] border-r-[12px] border-r-transparent"></div>
                </div>
              </div>
            </div>
            
            {/* Keyboard Mock */}
            <div className="bg-slate-50 h-16 w-full border-t border-slate-200 px-4 flex items-center justify-center text-slate-400 text-xs">
              <div className="bg-surface w-full rounded-full h-10 border border-slate-200 flex items-center px-4">
                Ketik pesan...
              </div>
            </div>
          </div>
          
          <div className="text-center mt-4 text-xs text-ink-muted/70 flex items-center justify-center gap-1">
            <Smartphone size={14} />
            <span>Simulasi pratinjau pesan di layar ponsel</span>
          </div>
        </div>

      </div>
    </div>
  );
}
