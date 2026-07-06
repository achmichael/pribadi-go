import { useState } from 'react';
import { Sparkles, Briefcase, Coffee, ArrowRight } from 'lucide-react';

interface Props {
  onNext: () => void;
  onBack: () => void;
}

const PRESETS = [
  { id: 'profesional', label: 'Profesional', icon: Briefcase, desc: 'Tegas, sopan, dan langsung ke intinya.' },
  { id: 'formal', label: 'Formal', icon: Briefcase, desc: 'Menggunakan tata bahasa baku dan resmi.' },
  { id: 'santai', label: 'Santai', icon: Coffee, desc: 'Ramah, kasual, sesekali menggunakan emoji.' },
  { id: 'ceria', label: 'Ceria', icon: Sparkles, desc: 'Bersemangat, sangat suportif, dan ekspresif.' },
];

export function PersonaSetup({ onNext, onBack }: Props) {
  const [name, setName] = useState('');
  const [selectedPreset, setSelectedPreset] = useState('profesional');

  return (
    <div className="flex flex-col items-center space-y-8 animate-in fade-in slide-in-from-right-8 duration-300 w-full">
      <div className="text-center space-y-2">
        <h2 className="text-2xl font-bold text-brand-900">Kepribadian AI</h2>
        <p className="text-brand-500">Berikan identitas agar AI bisa berkomunikasi lebih personal.</p>
      </div>

      <div className="w-full max-w-md space-y-6">
        <div className="space-y-2">
          <label className="text-sm font-semibold text-brand-700 ml-1">Nama Panggilan AI</label>
          <input
            type="text"
            value={name}
            onChange={(e) => setName(e.target.value)}
            placeholder="Misal: Jojo, Asisten, atau AI-ku"
            className="w-full bg-white border border-brand-200 rounded-xl px-4 py-3 text-brand-900 placeholder-brand-400 focus:bg-white focus:ring-2 focus:ring-emerald-500/20 focus:border-emerald-500 transition-all outline-none shadow-sm"
          />
        </div>

        <div className="space-y-3">
          <label className="text-sm font-semibold text-brand-700 ml-1">Pilih Gaya Bahasa</label>
          <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
            {PRESETS.map((preset) => {
              const Icon = preset.icon;
              const isSelected = selectedPreset === preset.id;
              
              return (
                <button
                  key={preset.id}
                  onClick={() => setSelectedPreset(preset.id)}
                  className={`text-left p-4 rounded-xl border-2 transition-all ${
                    isSelected
                      ? 'border-emerald-500 bg-emerald-50 shadow-sm'
                      : 'border-brand-200 bg-white hover:border-emerald-200 hover:bg-emerald-50/30'
                  }`}
                >
                  <div className="flex items-center gap-2 mb-1.5">
                    <Icon size={18} className={isSelected ? 'text-emerald-600' : 'text-brand-500'} />
                    <span className={`font-semibold ${isSelected ? 'text-emerald-900' : 'text-brand-700'}`}>
                      {preset.label}
                    </span>
                  </div>
                  <p className="text-xs text-brand-500 leading-relaxed">
                    {preset.desc}
                  </p>
                </button>
              );
            })}
          </div>
        </div>

        <div className="flex items-center gap-4 pt-4">
          <button 
            onClick={onBack} 
            className="px-6 py-3 text-brand-500 font-medium hover:text-brand-800 transition-colors"
          >
            Kembali
          </button>
          
          <button
            onClick={onNext}
            disabled={!name}
            className="flex-1 flex items-center justify-center gap-2 bg-emerald-600 text-white rounded-xl py-3 px-4 font-medium hover:bg-emerald-700 active:scale-[0.98] transition-all disabled:opacity-50 disabled:cursor-not-allowed shadow-sm"
          >
            <span>Simpan</span>
            <ArrowRight size={18} />
          </button>
        </div>
      </div>
    </div>
  );
}
