import { useEffect, useState } from 'react';
import { Loader2, CheckCircle2, QrCode } from 'lucide-react';

interface QRScannerProps {
  onConnected: () => void;
}

export function QRScanner({ onConnected }: QRScannerProps) {
  const [status, setStatus] = useState<'loading' | 'waiting' | 'connected'>('loading');

  useEffect(() => {
    // Mocking API call to fetch QR Code
    let isMounted = true;
    
    const fetchQR = () => {
      if (!isMounted) return;
      // In a real app, this would be an API call to get the base64 QR image
      setStatus('waiting');
    };

    const timer = setTimeout(fetchQR, 1000);

    return () => {
      isMounted = false;
      clearTimeout(timer);
    };
  }, []);

  useEffect(() => {
    // Mocking auto-polling for connection status
    if (status !== 'waiting') return;

    const pollInterval = setInterval(() => {
      // In a real app, this would call /api/v1/whatsapp/status
    }, 2000);

    const demoTimeout = setTimeout(() => {
      setStatus('connected');
      setTimeout(() => {
        onConnected();
      }, 1500); // Wait a bit before advancing
    }, 5000);

    return () => {
      clearInterval(pollInterval);
      clearTimeout(demoTimeout);
    };
  }, [status, onConnected]);

  return (
    <div className="flex flex-col items-center justify-center space-y-6 bg-surface p-8 rounded-2xl border border-ink-primary/10 max-w-sm w-full mx-auto shadow-sm">
      <div className="relative flex items-center justify-center w-64 h-64 bg-canvas rounded-xl border-2 border-dashed border-ink-primary/10">
        {status === 'loading' && (
          <div className="flex flex-col items-center text-ink-muted space-y-3 animate-pulse">
            <Loader2 size={32} className="animate-spin" />
            <span className="text-sm font-medium">Memuat kode QR...</span>
          </div>
        )}
        
        {status === 'waiting' && (
          <div className="absolute inset-0 p-4">
            {/* Pulsing border effect */}
            <div className="absolute inset-0 border-4 border-accent-primary rounded-xl animate-pulse opacity-20 pointer-events-none"></div>
            
            {/* Mock QR Code representation */}
            <div className="w-full h-full bg-surface rounded-lg flex items-center justify-center border border-brand-100 shadow-sm">
               <QrCode size={120} className="text-slate-800" />
            </div>
          </div>
        )}

        {status === 'connected' && (
          <div className="flex flex-col items-center text-emerald-500 space-y-3 animate-in zoom-in duration-300">
            <div className="bg-emerald-100 p-4 rounded-full">
              <CheckCircle2 size={48} />
            </div>
            <span className="text-accent-primary font-semibold">Berhasil Terhubung!</span>
          </div>
        )}
      </div>

      <div className="text-center space-y-2">
        <h3 className="font-sora font-semibold text-ink-primary">
          {status === 'connected' ? 'Perangkat Tertaut' : 'Pindai Kode QR'}
        </h3>
        <p className="text-sm text-ink-muted max-w-[250px]">
          {status === 'connected' 
            ? 'Menyiapkan dashboard Anda...' 
            : 'Buka WhatsApp di HP Anda > Perangkat Tertaut > Pindai kode ini.'}
        </p>
      </div>
    </div>
  );
}
