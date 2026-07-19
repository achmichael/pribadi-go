import { Settings } from 'lucide-react';
import { EmptyState } from '../components/ui/EmptyState';
import { COPY } from '../lib/copy';

export default function SettingsPage() {
  return (
    <div className="space-y-6">
      <h1 className="font-sora text-2xl font-bold text-ink-primary">{COPY.MENU_SETTINGS}</h1>
      <EmptyState
        icon={Settings}
        title="Pengaturan"
        description="Halaman pengaturan akan tersedia pada tahap pengembangan selanjutnya."
      />
    </div>
  );
}
