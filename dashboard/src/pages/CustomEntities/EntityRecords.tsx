import { useState, useEffect } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { ArrowLeft, Download, MessageCircle, FileSpreadsheet } from 'lucide-react';
import { EmptyState } from '../../components/ui/EmptyState';
import { Modal } from '../../components/ui/Modal';

// --- MOCK DATA ---
const MOCK_SCHEMA = {
  title: 'Order Penjualan',
  properties: {
    namaPelanggan: { title: 'Nama Pelanggan', type: 'string' },
    produk: { title: 'Produk (SKU)', type: 'string' },
    jumlah: { title: 'Jumlah', type: 'number' },
    alamat: { title: 'Alamat Pengiriman', type: 'string' }
  }
};

const MOCK_RECORDS = [
  {
    id: 'rec-101',
    created_at: '2024-05-12T10:30:00Z',
    data: {
      namaPelanggan: 'Budi Santoso',
      produk: 'TSHIRT-BLK-L',
      jumlah: 2,
      alamat: 'Jl. Merdeka No. 45, Jakarta'
    },
    source_chat: 'Halo min, saya mau pesan kaos hitam ukuran L 2 pcs, kirim ke Jl. Merdeka No. 45 Jakarta ya.'
  },
  {
    id: 'rec-102',
    created_at: '2024-05-12T11:15:00Z',
    data: {
      namaPelanggan: 'Siti Aminah',
      produk: 'JACKET-BLU-M',
      jumlah: 1,
      alamat: 'Komp. Mawar Blok B/12, Bandung'
    },
    source_chat: 'Pesen jaket biru size M 1 biji dong, alamatnya di Komp. Mawar Blok B/12 Bandung. Bisa COD?'
  }
];

export default function EntityRecords() {
  const { id } = useParams<{ id: string }>(); // Entity ID
  const navigate = useNavigate();
  
  const [loading, setLoading] = useState(true);
  const [selectedChat, setSelectedChat] = useState<string | null>(null);

  useEffect(() => {
    // Simulate fetching data based on ID
    const timer = setTimeout(() => {
      setLoading(false);
    }, 800);
    return () => clearTimeout(timer);
  }, [id]);

  const handleExportCSV = () => {
    if (MOCK_RECORDS.length === 0) return;

    // Build headers from schema
    const keys = Object.keys(MOCK_SCHEMA.properties);
    const headers = ['ID', 'Waktu Ekstraksi', ...keys.map(k => (MOCK_SCHEMA.properties as any)[k].title)];
    
    // Build rows
    const rows = MOCK_RECORDS.map(record => {
      const rowData = keys.map(k => `"${(record.data as any)[k] || ''}"`);
      return [`"${record.id}"`, `"${new Date(record.created_at).toLocaleString()}"`, ...rowData].join(',');
    });

    const csvContent = "data:text/csv;charset=utf-8," + [headers.join(','), ...rows].join('\n');
    const encodedUri = encodeURI(csvContent);
    const link = document.createElement("a");
    link.setAttribute("href", encodedUri);
    link.setAttribute("download", `${MOCK_SCHEMA.title}_export.csv`);
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
  };

  const schemaKeys = Object.keys(MOCK_SCHEMA.properties);

  return (
    <div className="space-y-6 animate-in fade-in duration-500 pb-20 lg:pb-8">
      
      {/* Header */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
        <div className="flex items-center gap-3">
          <button 
            onClick={() => navigate('/data-types')}
            className="p-2 text-ink-muted/70 hover:text-ink-primary bg-surface border border-ink-primary/10 rounded-xl hover:bg-canvas transition-colors shadow-sm"
          >
            <ArrowLeft size={20} />
          </button>
          <div>
            <h1 className="font-sora text-2xl font-bold text-ink-primary">Data {MOCK_SCHEMA.title}</h1>
            <p className="text-sm text-ink-muted">Data ini diekstraksi secara otomatis oleh AI dari percakapan.</p>
          </div>
        </div>
        
        <button
          onClick={handleExportCSV}
          disabled={loading || MOCK_RECORDS.length === 0}
          className="flex items-center justify-center gap-2 bg-surface text-ink-primary border border-ink-primary/10 px-5 py-2.5 rounded-xl font-medium hover:bg-canvas transition-colors disabled:opacity-50 disabled:cursor-not-allowed shadow-sm"
        >
          <Download size={18} />
          <span>Export CSV</span>
        </button>
      </div>

      {/* Content */}
      <div className="bg-surface border border-ink-primary/10 rounded-2xl shadow-sm overflow-hidden">
        {loading ? (
          <div className="py-24 text-center text-ink-muted/70">
            <div className="animate-pulse flex flex-col items-center">
              <div className="w-12 h-12 bg-brand-100 rounded-xl mb-4"></div>
              <div className="h-4 bg-brand-100 rounded w-48 mb-2"></div>
              <div className="h-3 bg-canvas rounded w-32"></div>
            </div>
          </div>
        ) : MOCK_RECORDS.length === 0 ? (
          <div className="py-16">
            <EmptyState 
              icon={FileSpreadsheet}
              title="Belum ada Data"
              description="AI belum mengekstrak data apapun untuk entitas ini. Data akan muncul otomatis saat ada percakapan yang relevan."
            />
          </div>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full text-left border-collapse">
              <thead>
                <tr className="bg-canvas/50 text-ink-muted text-xs uppercase tracking-wider border-b border-brand-100">
                  <th className="px-6 py-4 font-semibold whitespace-nowrap">Waktu</th>
                  {/* Dynamic Columns from Schema */}
                  {schemaKeys.map((key) => (
                    <th key={key} className="px-6 py-4 font-semibold whitespace-nowrap">
                      {(MOCK_SCHEMA.properties as any)[key].title}
                    </th>
                  ))}
                  <th className="px-6 py-4 font-semibold text-right">Aksi</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-brand-100">
                {MOCK_RECORDS.map((record) => (
                  <tr key={record.id} className="hover:bg-canvas/50 transition-colors">
                    <td className="px-6 py-4 text-sm text-ink-muted whitespace-nowrap">
                      {new Date(record.created_at).toLocaleString('id-ID', {
                        day: '2-digit', month: 'short', hour: '2-digit', minute: '2-digit'
                      })}
                    </td>
                    
                    {/* Dynamic Data Cells */}
                    {schemaKeys.map((key) => (
                      <td key={key} className="px-6 py-4 text-ink-primary max-w-[200px] truncate">
                        {(record.data as any)[key] || '-'}
                      </td>
                    ))}

                    <td className="px-6 py-4 text-right">
                      <button 
                        onClick={() => setSelectedChat(record.source_chat)}
                        className="inline-flex items-center gap-1.5 px-3 py-1.5 bg-brand-100 text-ink-primary hover:bg-brand-200 hover:text-ink-primary rounded-lg text-sm font-medium transition-colors"
                      >
                        <MessageCircle size={16} />
                        <span className="hidden sm:inline">Lihat Chat</span>
                      </button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>

      {/* Chat Traceback Modal */}
      <Modal
        isOpen={selectedChat !== null}
        onClose={() => setSelectedChat(null)}
        title="Cuplikan Sumber Chat"
      >
        <div className="mt-2 space-y-4">
          <p className="text-sm text-ink-muted">
            Berikut adalah cuplikan pesan mentah dari pelanggan yang diekstraksi oleh AI menjadi format terstruktur:
          </p>
          
          <div className="bg-[#E5DDD5] p-4 rounded-xl max-w-sm">
            <div className="bg-surface rounded-lg p-3 shadow-sm relative text-sm text-gray-800">
              {selectedChat}
              <div className="absolute top-0 -left-2 w-0 h-0 border-t-[10px] border-t-white border-l-[12px] border-l-transparent"></div>
            </div>
          </div>

          <div className="flex justify-end pt-4 border-t border-brand-100">
            <button 
              onClick={() => setSelectedChat(null)}
              className="bg-brand-100 text-ink-primary px-5 py-2 rounded-lg font-medium hover:bg-brand-200 transition-colors"
            >
              Tutup
            </button>
          </div>
        </div>
      </Modal>

    </div>
  );
}
