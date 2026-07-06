import { useState } from 'react';
import { Trash2, Eye, FileText, CheckCircle2, Loader2, AlertCircle } from 'lucide-react';
import { ConfirmDialog } from '../components/ui/ConfirmDialog';
import { EmptyState } from '../components/ui/EmptyState';
import { FileUpload } from '../components/ui/FileUpload';

// --- Types & Mock Data ---
interface DocumentItem {
  id: string;
  name: string;
  uploadDate: string;
  status: 'completed' | 'processing' | 'failed';
}

const INITIAL_DOCS: DocumentItem[] = [
  { id: '1', name: 'Katalog_Produk_Q3.pdf', uploadDate: '12 Mei 2024, 14:30', status: 'completed' },
  { id: '2', name: 'SOP_Customer_Service.pdf', uploadDate: '10 Mei 2024, 09:15', status: 'completed' },
  { id: '3', name: 'Kebijakan_Pengembalian.txt', uploadDate: 'Hari ini, 08:00', status: 'processing' },
];

export default function Knowledge() {
  const [documents, setDocuments] = useState<DocumentItem[]>(INITIAL_DOCS);
  const [docToDelete, setDocToDelete] = useState<string | null>(null);

  const handleUploadSuccess = (file: File) => {
    // Mock adding new document
    const newDoc: DocumentItem = {
      id: Math.random().toString(),
      name: file.name,
      uploadDate: 'Baru saja',
      status: 'processing'
    };
    setDocuments([newDoc, ...documents]);
  };

  const handleDelete = () => {
    if (docToDelete) {
      setDocuments(documents.filter(d => d.id !== docToDelete));
      setDocToDelete(null);
    }
  };

  return (
    <div className="space-y-8 animate-in fade-in duration-500 pb-20 lg:pb-8">
      
      {/* Header */}
      <div>
        <h1 className="text-2xl font-bold text-brand-900 mb-2">Pengetahuan AI (Dokumen)</h1>
        <p className="text-brand-500">
          Unggah dokumen referensi agar AI bisa mempelajarinya dan menjawab pertanyaan berdasarkan data Anda.
        </p>
      </div>

      {/* Upload Zone */}
      <div className="bg-white border border-brand-200 rounded-2xl p-6 shadow-sm">
        <h2 className="text-lg font-bold text-brand-900 mb-4">Tambah Referensi Baru</h2>
        <FileUpload onUploadSuccess={handleUploadSuccess} />
      </div>

      {/* Document List */}
      <div className="bg-white border border-brand-200 rounded-2xl shadow-sm overflow-hidden">
        <div className="px-6 py-5 border-b border-brand-100 flex items-center justify-between bg-brand-50/50">
          <h2 className="text-lg font-bold text-brand-900">Daftar Dokumen</h2>
          <span className="bg-white border border-brand-200 text-brand-600 text-xs font-bold px-3 py-1 rounded-full">
            {documents.length} File
          </span>
        </div>
        
        {documents.length === 0 ? (
          <div className="py-12">
            <EmptyState 
              icon={FileText}
              title="Belum ada dokumen"
              description="Unggah PDF atau TXT di atas agar AI mulai belajar."
            />
          </div>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full text-left border-collapse">
              <thead>
                <tr className="bg-brand-50/30 text-brand-500 text-xs uppercase tracking-wider">
                  <th className="px-6 py-4 font-semibold">Nama File</th>
                  <th className="px-6 py-4 font-semibold">Tanggal Unggah</th>
                  <th className="px-6 py-4 font-semibold">Status</th>
                  <th className="px-6 py-4 font-semibold text-right">Aksi</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-brand-100">
                {documents.map((doc) => (
                  <tr key={doc.id} className="hover:bg-brand-50/50 transition-colors">
                    <td className="px-6 py-4">
                      <div className="flex items-center gap-3">
                        <div className="p-2 bg-brand-100 text-brand-500 rounded-lg shrink-0">
                          <FileText size={16} />
                        </div>
                        <span className="font-medium text-brand-900 truncate max-w-[200px] sm:max-w-[300px]">
                          {doc.name}
                        </span>
                      </div>
                    </td>
                    <td className="px-6 py-4 text-sm text-brand-500 whitespace-nowrap">
                      {doc.uploadDate}
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap">
                      {doc.status === 'completed' && (
                        <span className="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-md bg-emerald-50 text-emerald-700 text-xs font-semibold border border-emerald-100">
                          <CheckCircle2 size={14} /> Terpelajari
                        </span>
                      )}
                      {doc.status === 'processing' && (
                        <span className="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-md bg-amber-50 text-amber-700 text-xs font-semibold border border-amber-100">
                          <Loader2 size={14} className="animate-spin" /> Memproses
                        </span>
                      )}
                      {doc.status === 'failed' && (
                        <span className="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-md bg-rose-50 text-rose-700 text-xs font-semibold border border-rose-100">
                          <AlertCircle size={14} /> Gagal
                        </span>
                      )}
                    </td>
                    <td className="px-6 py-4 text-right whitespace-nowrap">
                      <div className="flex items-center justify-end gap-2">
                        <button 
                          title="Lihat Cuplikan"
                          className="p-2 text-brand-400 hover:text-brand-700 hover:bg-brand-100 rounded-lg transition-colors"
                        >
                          <Eye size={18} />
                        </button>
                        <button 
                          onClick={() => setDocToDelete(doc.id)}
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

      {/* Delete Confirmation */}
      <ConfirmDialog 
        isOpen={docToDelete !== null}
        onClose={() => setDocToDelete(null)}
        onConfirm={handleDelete}
        title="Hapus Dokumen?"
        message="Dokumen ini akan dihapus dari ingatan AI secara permanen. Anda yakin?"
        confirmLabel="Ya, Hapus"
      />
    </div>
  );
}
