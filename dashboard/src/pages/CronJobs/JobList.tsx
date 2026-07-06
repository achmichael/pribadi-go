import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { CalendarClock, Plus, Edit2, Trash2, Power, PowerOff } from 'lucide-react';
import { EmptyState } from '../../components/ui/EmptyState';
import { ConfirmDialog } from '../../components/ui/ConfirmDialog';

interface CronJob {
  id: string;
  name: string;
  job_type: string;
  frequency: string;
  schedule_time: string; // e.g. "09:00"
  target_whatsapp_jid: string;
  is_active: boolean;
}

const MOCK_JOBS: CronJob[] = [
  {
    id: '1',
    name: 'Follow-up Pembayaran',
    job_type: 'custom_message',
    frequency: 'Satu Kali',
    schedule_time: 'Besok, 09:00',
    target_whatsapp_jid: '08123456789',
    is_active: true,
  },
  {
    id: '2',
    name: 'Katalog Mingguan',
    job_type: 'broadcast',
    frequency: 'Mingguan (Senin)',
    schedule_time: '10:00',
    target_whatsapp_jid: 'Semua Pelanggan',
    is_active: false,
  }
];

export default function JobList() {
  const navigate = useNavigate();
  const [jobs, setJobs] = useState<CronJob[]>(MOCK_JOBS);
  const [jobToDelete, setJobToDelete] = useState<string | null>(null);

  const toggleActive = (id: string) => {
    setJobs(jobs.map(j => j.id === id ? { ...j, is_active: !j.is_active } : j));
  };

  const handleDelete = () => {
    if (jobToDelete) {
      setJobs(jobs.filter(j => j.id !== jobToDelete));
      setJobToDelete(null);
    }
  };

  return (
    <div className="space-y-6 animate-in fade-in duration-500 pb-20 lg:pb-8">
      
      {/* Header */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold text-brand-900 mb-2">Penjadwalan Tugas</h1>
          <p className="text-brand-500">
            Atur asisten AI untuk mengirim pesan proaktif pada waktu tertentu.
          </p>
        </div>
        <button
          onClick={() => navigate('/schedules/new')}
          className="flex items-center justify-center gap-2 bg-emerald-600 text-white px-5 py-2.5 rounded-xl font-medium hover:bg-emerald-700 transition-colors shadow-sm"
        >
          <Plus size={18} />
          <span>Buat Jadwal Baru</span>
        </button>
      </div>

      {/* List / Empty State */}
      <div className="bg-white border border-brand-200 rounded-2xl shadow-sm overflow-hidden">
        {jobs.length === 0 ? (
          <div className="py-16">
            <EmptyState 
              icon={CalendarClock}
              title="Belum ada Jadwal"
              description="Asisten AI belum memiliki tugas proaktif. Anda bisa menyetel pesan otomatis, misalnya untuk pengingat tagihan."
              action={{
                label: "Buat Jadwal Pertama",
                onClick: () => navigate('/schedules/new')
              }}
            />
          </div>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full text-left border-collapse">
              <thead>
                <tr className="bg-brand-50/50 text-brand-500 text-xs uppercase tracking-wider border-b border-brand-100">
                  <th className="px-6 py-4 font-semibold">Tugas / Pesan</th>
                  <th className="px-6 py-4 font-semibold">Target</th>
                  <th className="px-6 py-4 font-semibold">Waktu Eksekusi</th>
                  <th className="px-6 py-4 font-semibold">Status</th>
                  <th className="px-6 py-4 font-semibold text-right">Aksi</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-brand-100">
                {jobs.map((j) => (
                  <tr key={j.id} className="hover:bg-brand-50/50 transition-colors">
                    <td className="px-6 py-4">
                      <div className="font-semibold text-brand-900">{j.name}</div>
                      <div className="text-xs text-brand-500 mt-0.5">{j.job_type.replace('_', ' ')}</div>
                    </td>
                    <td className="px-6 py-4 text-sm font-medium text-brand-700">
                      {j.target_whatsapp_jid}
                    </td>
                    <td className="px-6 py-4">
                      <div className="text-sm font-semibold text-brand-900">{j.frequency}</div>
                      <div className="text-xs text-brand-500 mt-0.5">{j.schedule_time}</div>
                    </td>
                    <td className="px-6 py-4">
                      <button
                        onClick={() => toggleActive(j.id)}
                        className={`inline-flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs font-bold transition-colors ${
                          j.is_active 
                            ? 'bg-emerald-50 text-emerald-700 hover:bg-emerald-100' 
                            : 'bg-slate-100 text-slate-600 hover:bg-slate-200'
                        }`}
                      >
                        {j.is_active ? <Power size={14} /> : <PowerOff size={14} />}
                        {j.is_active ? 'Aktif' : 'Berhenti'}
                      </button>
                    </td>
                    <td className="px-6 py-4 text-right">
                      <div className="flex items-center justify-end gap-1">
                        <button 
                          title="Edit"
                          onClick={() => navigate(`/schedules/${j.id}/edit`)}
                          className="p-2 text-brand-400 hover:text-brand-700 hover:bg-brand-100 rounded-lg transition-colors"
                        >
                          <Edit2 size={18} />
                        </button>
                        <button 
                          onClick={() => setJobToDelete(j.id)}
                          title="Hapus"
                          className="p-2 text-brand-400 hover:text-rose-600 hover:bg-rose-50 rounded-lg transition-colors"
                        >
                          <Trash2 size={18} />
                        </button>
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>

      <ConfirmDialog 
        isOpen={jobToDelete !== null}
        onClose={() => setJobToDelete(null)}
        onConfirm={handleDelete}
        title="Hapus Jadwal?"
        message="Jadwal pengiriman pesan ini akan dibatalkan secara permanen. Anda yakin?"
        confirmLabel="Ya, Hapus"
      />
    </div>
  );
}
