import { Smartphone } from 'lucide-react';
import { EmptyState } from '../components/ui/EmptyState';
import { COPY } from '../lib/copy';

export default function Connections() {
  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold text-brand-900">{COPY.MENU_CONNECTIONS}</h1>
      <EmptyState
        icon={Smartphone}
        title="Belum Ada Koneksi"
        description="Hubungkan AI dengan WhatsApp atau Telegram agar bisa mulai digunakan."
      />
    </div>
  );
}
