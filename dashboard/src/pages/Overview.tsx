import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { 
  MessageCircle, 
  Smartphone, 
  BookOpen, 
  Clock, 
  AlertTriangle,
  ArrowRight,
  Bot
} from 'lucide-react';


// --- Types & Mock Data ---
interface StatCardProps {
  title: string;
  value: string;
  status: 'connected' | 'disconnected' | 'info';
  icon: any;
  onClick: () => void;
  subtext?: string;
}

const MOCK_ACTIVITIES = [
  { id: 1, type: 'message', text: 'AI menjawab pertanyaan seputar jam buka toko ke pelanggan di WhatsApp.', time: '5 mnt lalu' },
  { id: 2, type: 'message', text: 'AI merespons keluhan Budi terkait pesanan telat di Telegram.', time: '12 mnt lalu' },
  { id: 3, type: 'schedule', text: 'Rangkuman aktivitas harian berhasil dikirim ke nomor Anda.', time: '1 jam lalu' },
  { id: 4, type: 'system', text: 'Koneksi WhatsApp sempat terputus, namun berhasil dihubungkan kembali otomatis.', time: '2 jam lalu' },
  { id: 5, type: 'knowledge', text: 'Dokumen "Katalog_Produk_Q3.pdf" berhasil dipelajari AI.', time: 'Kemarin' },
];

// --- Subcomponents ---
function StatCard({ title, value, status, icon: Icon, onClick, subtext }: StatCardProps) {
  return (
    <div 
      onClick={onClick}
      className="bg-white border border-brand-200 rounded-2xl p-5 cursor-pointer hover:border-emerald-300 hover:shadow-md transition-all group"
    >
      <div className="flex items-start justify-between mb-4">
        <div className={`p-3 rounded-xl ${
          status === 'connected' ? 'bg-emerald-100 text-emerald-600' :
          status === 'disconnected' ? 'bg-rose-100 text-rose-600' :
          'bg-brand-100 text-brand-600'
        }`}>
          <Icon size={24} />
        </div>
        <div className="text-brand-300 group-hover:text-emerald-500 transition-colors">
          <ArrowRight size={20} />
        </div>
      </div>
      <div>
        <h3 className="text-brand-500 text-sm font-medium mb-1">{title}</h3>
        <div className="flex items-baseline gap-2">
          <span className="text-2xl font-bold text-brand-900">{value}</span>
          {subtext && <span className="text-sm font-medium text-brand-500">{subtext}</span>}
        </div>
      </div>
    </div>
  );
}

// --- Main Component ---
export default function Overview() {
  const navigate = useNavigate();
  const [hasWarning] = useState(true); // Mock warning state

  return (
    <div className="space-y-8 animate-in fade-in duration-500 pb-20 lg:pb-8">
      
      {/* Header */}
      <div>
        <h1 className="text-2xl font-bold text-brand-900 mb-2">Selamat Datang, Admin 👋</h1>
        <p className="text-brand-500">Berikut adalah ringkasan status asisten AI Anda hari ini.</p>
      </div>

      {/* Warning Banner */}
      {hasWarning && (
        <div className="bg-amber-50 border border-amber-200 rounded-xl p-4 flex flex-col sm:flex-row items-start sm:items-center justify-between gap-4">
          <div className="flex items-start gap-3">
            <div className="text-amber-500 mt-0.5">
              <AlertTriangle size={20} />
            </div>
            <div>
              <h4 className="font-semibold text-amber-900">Telegram Belum Terhubung</h4>
              <p className="text-sm text-amber-700 mt-1">
                Anda belum mengonfigurasi Bot Telegram. Asisten AI tidak dapat merespons pesan dari Telegram.
              </p>
            </div>
          </div>
          <button 
            onClick={() => navigate('/connections')}
            className="shrink-0 bg-white border border-amber-200 text-amber-700 px-4 py-2 rounded-lg text-sm font-medium hover:bg-amber-100 transition-colors"
          >
            Hubungkan Sekarang
          </button>
        </div>
      )}

      {/* Stats Grid */}
      <div className="grid grid-cols-2 lg:grid-cols-4 gap-4">
        <StatCard 
          title="Status WhatsApp" 
          value="Aktif" 
          status="connected" 
          icon={MessageCircle} 
          onClick={() => navigate('/connections')} 
        />
        <StatCard 
          title="Status Telegram" 
          value="Terputus" 
          status="disconnected" 
          icon={Smartphone} 
          onClick={() => navigate('/connections')} 
        />
        <StatCard 
          title="Pengetahuan AI" 
          value="12" 
          subtext="Dokumen"
          status="info" 
          icon={BookOpen} 
          onClick={() => navigate('/knowledge')} 
        />
        <StatCard 
          title="Jadwal Otomatis" 
          value="3" 
          subtext="Aktif"
          status="info" 
          icon={Clock} 
          onClick={() => navigate('/schedules')} 
        />
      </div>

      {/* Activity Feed */}
      <div className="bg-white border border-brand-200 rounded-2xl overflow-hidden">
        <div className="px-6 py-5 border-b border-brand-100">
          <h2 className="text-lg font-bold text-brand-900">Aktivitas Terkini</h2>
        </div>
        <div className="divide-y divide-brand-100">
          {MOCK_ACTIVITIES.map((activity) => (
            <div key={activity.id} className="p-6 flex items-start gap-4 hover:bg-brand-50/50 transition-colors">
              <div className="shrink-0 mt-1">
                <div className="w-10 h-10 rounded-full bg-brand-100 text-brand-500 flex items-center justify-center">
                  <Bot size={20} />
                </div>
              </div>
              <div className="flex-1 min-w-0">
                <p className="text-brand-900 text-sm leading-relaxed">
                  {activity.text}
                </p>
                <p className="text-xs font-medium text-brand-400 mt-2">
                  {activity.time}
                </p>
              </div>
            </div>
          ))}
        </div>
        <div className="p-4 bg-brand-50 border-t border-brand-100 text-center">
          <button 
            onClick={() => navigate('/activity')}
            className="text-sm font-semibold text-emerald-600 hover:text-emerald-700 transition-colors"
          >
            Lihat Semua Aktivitas
          </button>
        </div>
      </div>

    </div>
  );
}
