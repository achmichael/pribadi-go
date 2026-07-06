import { PartyPopper } from 'lucide-react';
import { useNavigate } from 'react-router-dom';

export function SuccessScreen() {
  const navigate = useNavigate();

  return (
    <div className="flex flex-col items-center text-center space-y-8 animate-in zoom-in-95 duration-500">
      <div className="w-24 h-24 bg-emerald-100 text-emerald-600 rounded-full flex items-center justify-center shadow-sm">
        <PartyPopper size={48} />
      </div>
      
      <div className="space-y-4 max-w-md mx-auto">
        <h1 className="text-3xl font-bold text-brand-900">
          Semua Selesai!
        </h1>
        <p className="text-brand-500 text-lg leading-relaxed">
          Asisten AI Anda sudah aktif dan terhubung. Anda sekarang dapat mulai melatihnya dan memantau kinerjanya dari Dashboard.
        </p>
      </div>

      <button
        onClick={() => navigate('/')}
        className="bg-brand-900 text-white px-8 py-3.5 rounded-xl font-medium hover:bg-brand-800 active:scale-[0.98] transition-all shadow-sm"
      >
        Buka Dashboard
      </button>
    </div>
  );
}
