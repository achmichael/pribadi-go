import { useState } from 'react';
import { MessageCircle, Smartphone, Settings2, Code, Link2 } from 'lucide-react';
import { Modal } from '../components/ui/Modal';
import { ConfirmDialog } from '../components/ui/ConfirmDialog';
import { QRScanner } from '../components/ui/QRScanner';

export default function Connections() {
  const [advancedMode, setAdvancedMode] = useState(false);
  
  // Mock States
  const [waConnected, setWaConnected] = useState(true);
  const [tgConnected, setTgConnected] = useState(false);
  
  // Modal States
  const [showQR, setShowQR] = useState(false);
  const [showConfirm, setShowConfirm] = useState<'whatsapp' | 'telegram' | null>(null);

  const handleDisconnect = () => {
    if (showConfirm === 'whatsapp') setWaConnected(false);
    if (showConfirm === 'telegram') setTgConnected(false);
    setShowConfirm(null);
  };

  const handleQRConnected = () => {
    setWaConnected(true);
    setShowQR(false);
  };

  return (
    <div className="space-y-6 animate-in fade-in duration-500 pb-20 lg:pb-8">
      
      {/* Header */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
        <div>
          <h1 className="font-sora text-2xl font-bold text-ink-primary mb-2">Koneksi Platform</h1>
          <p className="text-ink-muted">Kelola platform pesan tempat asisten AI Anda beroperasi.</p>
        </div>
        
        {/* Advanced Mode Toggle */}
        <label className="flex items-center gap-3 cursor-pointer bg-surface border border-ink-primary/10 px-4 py-2.5 rounded-xl hover:bg-canvas transition-colors">
          <Settings2 size={18} className="text-ink-muted" />
          <span className="text-sm font-medium text-ink-primary">Mode Lanjutan</span>
          <div className="relative">
            <input 
              type="checkbox" 
              className="sr-only" 
              checked={advancedMode}
              onChange={(e) => setAdvancedMode(e.target.checked)}
            />
            <div className={`block w-10 h-6 rounded-full transition-colors ${advancedMode ? 'bg-accent-primary' : 'bg-brand-200'}`}></div>
            <div className={`absolute left-1 top-1 bg-surface w-4 h-4 rounded-full transition-transform ${advancedMode ? 'translate-x-4' : ''}`}></div>
          </div>
        </label>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        
        {/* WhatsApp Card */}
        <div className="bg-surface border border-ink-primary/10 rounded-2xl p-6 shadow-sm">
          <div className="flex items-start justify-between mb-6">
            <div className="flex items-center gap-4">
              <div className={`p-3 rounded-xl ${waConnected ? 'bg-emerald-100 text-accent-primary' : 'bg-brand-100 text-ink-muted/70'}`}>
                <MessageCircle size={32} />
              </div>
              <div>
                <h2 className="font-sora text-lg font-bold text-ink-primary">WhatsApp</h2>
                <div className="flex items-center gap-2 mt-1">
                  <span className="relative flex h-2.5 w-2.5">
                    {waConnected && <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-emerald-400 opacity-75"></span>}
                    <span className={`relative inline-flex rounded-full h-2.5 w-2.5 ${waConnected ? 'bg-accent-primary' : 'bg-accent-danger/100'}`}></span>
                  </span>
                  <span className="text-sm font-medium text-ink-muted">
                    {waConnected ? 'Terhubung' : 'Terputus'}
                  </span>
                </div>
              </div>
            </div>
          </div>

          <div className="space-y-4">
            <div className="bg-canvas rounded-xl p-4 border border-brand-100">
              <span className="text-xs font-semibold text-ink-muted/70 uppercase tracking-wider">Identitas Terhubung</span>
              <p className="font-medium text-ink-primary mt-1">{waConnected ? '+62 812 3456 7890' : 'Belum disetel'}</p>
            </div>

            {waConnected ? (
              <button 
                onClick={() => setShowConfirm('whatsapp')}
                className="w-full py-2.5 rounded-xl border border-rose-200 text-accent-danger font-medium hover:bg-accent-danger/10 transition-colors"
              >
                Putus Koneksi
              </button>
            ) : (
              <button 
                onClick={() => setShowQR(true)}
                className="w-full py-2.5 rounded-xl bg-accent-primary text-white font-medium hover:bg-accent-primary/90 transition-colors shadow-sm"
              >
                Hubungkan WhatsApp
              </button>
            )}

            {advancedMode && (
              <div className="mt-6 pt-4 border-t border-brand-100 space-y-3">
                <div className="flex items-center gap-2 text-ink-muted mb-2">
                  <Code size={16} />
                  <span className="text-sm font-semibold">Detail Teknis</span>
                </div>
                <div className="grid grid-cols-[100px_1fr] gap-2 text-xs font-mono bg-brand-900 text-brand-100 p-4 rounded-xl overflow-hidden">
                  <span className="text-ink-muted/70">Session ID:</span>
                  <span className="truncate">{waConnected ? 'sess_wa_098f6bcd46' : 'null'}</span>
                  <span className="text-ink-muted/70">Sync:</span>
                  <span>{waConnected ? '2024-05-12 14:02:11' : '-'}</span>
                  <span className="text-ink-muted/70">Webhook:</span>
                  <span className="truncate">https://api.domain.com/webhook/wa</span>
                </div>
              </div>
            )}
          </div>
        </div>

        {/* Telegram Card */}
        <div className="bg-surface border border-ink-primary/10 rounded-2xl p-6 shadow-sm">
          <div className="flex items-start justify-between mb-6">
            <div className="flex items-center gap-4">
              <div className={`p-3 rounded-xl ${tgConnected ? 'bg-emerald-100 text-accent-primary' : 'bg-brand-100 text-ink-muted/70'}`}>
                <Smartphone size={32} />
              </div>
              <div>
                <h2 className="font-sora text-lg font-bold text-ink-primary">Telegram</h2>
                <div className="flex items-center gap-2 mt-1">
                  <span className="relative flex h-2.5 w-2.5">
                    {tgConnected && <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-emerald-400 opacity-75"></span>}
                    <span className={`relative inline-flex rounded-full h-2.5 w-2.5 ${tgConnected ? 'bg-accent-primary' : 'bg-accent-danger/100'}`}></span>
                  </span>
                  <span className="text-sm font-medium text-ink-muted">
                    {tgConnected ? 'Terhubung' : 'Terputus'}
                  </span>
                </div>
              </div>
            </div>
          </div>

          <div className="space-y-4">
            <div className="bg-canvas rounded-xl p-4 border border-brand-100">
              <span className="text-xs font-semibold text-ink-muted/70 uppercase tracking-wider">Identitas Terhubung</span>
              <p className="font-medium text-ink-primary mt-1">{tgConnected ? '@AsistenBisnisBot' : 'Belum disetel'}</p>
            </div>

            {tgConnected ? (
              <button 
                onClick={() => setShowConfirm('telegram')}
                className="w-full py-2.5 rounded-xl border border-rose-200 text-accent-danger font-medium hover:bg-accent-danger/10 transition-colors"
              >
                Putus Koneksi
              </button>
            ) : (
              <div className="space-y-3">
                <input 
                  type="text" 
                  placeholder="Masukkan Token Bot Telegram..."
                  className="w-full bg-canvas border border-ink-primary/10 rounded-xl px-4 py-2.5 text-sm outline-none focus:ring-2 focus:ring-accent-primary/15 focus:border-accent-primary transition-all"
                />
                <button 
                  onClick={() => setTgConnected(true)}
                  className="w-full py-2.5 rounded-xl border border-ink-primary/10 text-ink-primary font-medium hover:bg-canvas transition-colors flex items-center justify-center gap-2"
                >
                  <Link2 size={18} />
                  <span>Simpan Token</span>
                </button>
              </div>
            )}

            {advancedMode && (
              <div className="mt-6 pt-4 border-t border-brand-100 space-y-3">
                <div className="flex items-center gap-2 text-ink-muted mb-2">
                  <Code size={16} />
                  <span className="text-sm font-semibold">Detail Teknis</span>
                </div>
                <div className="grid grid-cols-[100px_1fr] gap-2 text-xs font-mono bg-brand-900 text-brand-100 p-4 rounded-xl overflow-hidden">
                  <span className="text-ink-muted/70">Bot ID:</span>
                  <span className="truncate">{tgConnected ? '6123456789' : 'null'}</span>
                  <span className="text-ink-muted/70">Sync:</span>
                  <span>{tgConnected ? '2024-05-12 14:02:11' : '-'}</span>
                  <span className="text-ink-muted/70">Webhook:</span>
                  <span className="truncate">https://api.domain.com/webhook/tg</span>
                </div>
              </div>
            )}
          </div>
        </div>

      </div>

      {/* QR Scanner Modal */}
      <Modal isOpen={showQR} onClose={() => setShowQR(false)} title="Hubungkan WhatsApp">
        <div className="py-4">
          <QRScanner onConnected={handleQRConnected} />
        </div>
      </Modal>

      {/* Confirmation Dialog */}
      <ConfirmDialog 
        isOpen={showConfirm !== null}
        onClose={() => setShowConfirm(null)}
        onConfirm={handleDisconnect}
        title="Putus Koneksi?"
        message={`Apakah Anda yakin ingin memutus koneksi ${showConfirm === 'whatsapp' ? 'WhatsApp' : 'Telegram'}? Asisten AI tidak akan bisa merespons pesan pelanggan di platform ini hingga Anda menghubungkannya kembali.`}
        confirmLabel="Ya, Putuskan Koneksi"
      />

    </div>
  );
}
