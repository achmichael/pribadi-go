import { useState } from 'react';
import { QRScanner } from '../../../components/ui/QRScanner';
import { Smartphone, MessageCircle } from 'lucide-react';

interface Props {
  onNext: () => void;
  onBack: () => void;
}

export function ConnectPlatform({ onNext, onBack }: Props) {
  const [platform, setPlatform] = useState<'whatsapp' | 'telegram'>('whatsapp');

  return (
    <div className="flex flex-col items-center space-y-8 animate-in fade-in slide-in-from-right-8 duration-300 w-full">
      <div className="text-center space-y-2">
        <h2 className="text-2xl font-bold text-brand-900">Hubungkan Platform</h2>
        <p className="text-brand-500">Pilih platform pesan yang ingin Anda gunakan untuk asisten ini.</p>
      </div>

      <div className="flex gap-4 w-full max-w-sm">
        <button
          onClick={() => setPlatform('whatsapp')}
          className={`flex-1 flex flex-col items-center gap-3 p-4 rounded-xl border-2 transition-all ${
            platform === 'whatsapp' 
              ? 'border-emerald-500 bg-emerald-50 text-emerald-700' 
              : 'border-brand-200 bg-white text-brand-500 hover:border-emerald-200 hover:bg-emerald-50/50'
          }`}
        >
          <MessageCircle size={28} />
          <span className="font-semibold">WhatsApp</span>
        </button>
        
        <button
          onClick={() => setPlatform('telegram')}
          className={`flex-1 flex flex-col items-center gap-3 p-4 rounded-xl border-2 transition-all ${
            platform === 'telegram' 
              ? 'border-emerald-500 bg-emerald-50 text-emerald-700' 
              : 'border-brand-200 bg-white text-brand-500 hover:border-emerald-200 hover:bg-emerald-50/50'
          }`}
        >
          <Smartphone size={28} />
          <span className="font-semibold">Telegram</span>
        </button>
      </div>

      <div className="w-full">
        {platform === 'whatsapp' ? (
          <QRScanner onConnected={onNext} />
        ) : (
          <div className="bg-white border border-brand-200 p-8 rounded-2xl text-center max-w-sm mx-auto shadow-sm space-y-4">
            <h3 className="font-semibold text-brand-900">Setup Bot Telegram</h3>
            <p className="text-sm text-brand-500">
              Kirim pesan ke BotFather di Telegram untuk mendapatkan Token Bot Anda.
            </p>
            <input 
              type="text" 
              placeholder="Masukkan Token Bot..."
              className="w-full border border-brand-200 rounded-xl px-4 py-2 text-sm focus:ring-2 focus:ring-emerald-500/20 focus:border-emerald-500 outline-none"
            />
            <button 
              onClick={onNext}
              className="w-full bg-brand-900 text-white rounded-xl py-2 text-sm font-medium hover:bg-brand-800"
            >
              Simpan Token
            </button>
          </div>
        )}
      </div>

      <button onClick={onBack} className="text-brand-400 text-sm font-medium hover:text-brand-600 transition-colors">
        Kembali
      </button>
    </div>
  );
}
