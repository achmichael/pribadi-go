import { BookOpen } from 'lucide-react';
import { EmptyState } from '../components/ui/EmptyState';
import { COPY } from '../lib/copy';

export default function Knowledge() {
  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold text-brand-900">{COPY.MENU_KNOWLEDGE}</h1>
      <EmptyState
        icon={BookOpen}
        title="Pengetahuan AI Masih Kosong"
        description="Upload dokumen PDF atau teks untuk mengajarkan AI tentang bisnis Anda."
      />
    </div>
  );
}
