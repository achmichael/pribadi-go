import { Bot, ArrowRight } from 'lucide-react';

interface Props {
  onNext: () => void;
}

export function WelcomeScreen({ onNext }: Props) {
  return (
    <div className="flex flex-col items-center text-center space-y-8 animate-in fade-in slide-in-from-bottom-4 duration-500">
      <div className="w-24 h-24 bg-brand-100 text-ink-muted rounded-3xl flex items-center justify-center shadow-sm">
        <Bot size={48} />
      </div>
      
      <div className="space-y-4 max-w-md mx-auto">
        <h1 className="font-sora text-3xl font-bold text-ink-primary">
          Asisten AI Pribadi Anda
        </h1>
        <p className="text-ink-muted text-lg leading-relaxed">
          Selamat datang! Control Panel ini akan membantu Anda mengelola asisten AI cerdas yang bekerja langsung melalui WhatsApp atau Telegram Anda.
        </p>
      </div>

      <button
        onClick={onNext}
        className="flex items-center gap-2 bg-accent-primary text-white px-8 py-3.5 rounded-xl font-medium hover:bg-accent-primary/90 active:scale-[0.98] transition-all shadow-sm"
      >
        <span>Mulai Setup</span>
        <ArrowRight size={20} />
      </button>
    </div>
  );
}
