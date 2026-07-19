import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { Database, Plus, Edit2, Trash2, List } from 'lucide-react';
import { EmptyState } from '../../components/ui/EmptyState';
import { ConfirmDialog } from '../../components/ui/ConfirmDialog';

interface SchemaItem {
  id: string;
  name: string;
  description: string;
  fieldCount: number;
}

const MOCK_SCHEMAS: SchemaItem[] = [
  { id: '1', name: 'Order Penjualan', description: 'Format standar untuk menangkap pesanan dari pelanggan.', fieldCount: 5 },
  { id: '2', name: 'Data Lead', description: 'Formulir kontak untuk calon pelanggan baru.', fieldCount: 3 },
];

export default function SchemaList() {
  const navigate = useNavigate();
  const [schemas, setSchemas] = useState<SchemaItem[]>(MOCK_SCHEMAS);
  const [schemaToDelete, setSchemaToDelete] = useState<string | null>(null);

  const handleDelete = () => {
    if (schemaToDelete) {
      setSchemas(schemas.filter(s => s.id !== schemaToDelete));
      setSchemaToDelete(null);
    }
  };

  return (
    <div className="space-y-6 animate-in fade-in duration-500 pb-20 lg:pb-8">
      
      {/* Header */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
        <div>
          <h1 className="font-sora text-2xl font-bold text-ink-primary mb-2">Entitas Kustom</h1>
          <p className="text-ink-muted">
            Definisikan struktur data yang akan diekstraksi AI dari percakapan.
          </p>
        </div>
        <button
          onClick={() => navigate('/data-types/new')}
          className="flex items-center justify-center gap-2 bg-accent-primary text-white px-5 py-2.5 rounded-xl font-medium hover:bg-accent-primary/90 transition-colors shadow-sm"
        >
          <Plus size={18} />
          <span>Buat Entitas</span>
        </button>
      </div>

      {/* List / Empty State */}
      <div className="bg-surface border border-ink-primary/10 rounded-2xl shadow-sm overflow-hidden">
        {schemas.length === 0 ? (
          <div className="py-16">
            <EmptyState 
              icon={Database}
              title="Belum ada Entitas"
              description="Buat entitas kustom pertama Anda agar AI tahu format data apa yang harus dikumpulkan."
              action={{
                label: "Buat Entitas Sekarang",
                onClick: () => navigate('/data-types/new')
              }}
            />
          </div>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full text-left border-collapse">
              <thead>
                <tr className="bg-canvas/50 text-ink-muted text-xs uppercase tracking-wider border-b border-brand-100">
                  <th className="px-6 py-4 font-semibold">Nama Entitas</th>
                  <th className="px-6 py-4 font-semibold hidden sm:table-cell">Deskripsi</th>
                  <th className="px-6 py-4 font-semibold">Fields</th>
                  <th className="px-6 py-4 font-semibold text-right">Aksi</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-brand-100">
                {schemas.map((schema) => (
                  <tr key={schema.id} className="hover:bg-canvas/50 transition-colors">
                    <td className="px-6 py-4">
                      <div className="flex items-center gap-3">
                        <div className="p-2 bg-brand-100 text-ink-muted rounded-lg shrink-0">
                          <Database size={18} />
                        </div>
                        <span className="font-semibold text-ink-primary">{schema.name}</span>
                      </div>
                    </td>
                    <td className="px-6 py-4 text-sm text-ink-muted hidden sm:table-cell">
                      {schema.description}
                    </td>
                    <td className="px-6 py-4">
                      <span className="inline-flex items-center justify-center bg-brand-100 text-ink-primary text-xs font-bold px-2.5 py-1 rounded-md">
                        {schema.fieldCount}
                      </span>
                    </td>
                    <td className="px-6 py-4 text-right">
                      <div className="flex items-center justify-end gap-1">
                        <button 
                          title="Lihat Data"
                          onClick={() => navigate(`/data-types/${schema.id}/records`)}
                          className="p-2 text-ink-muted/70 hover:text-accent-primary hover:bg-accent-primary/10 rounded-lg transition-colors"
                        >
                          <List size={18} />
                        </button>
                        <button 
                          title="Edit"
                          onClick={() => navigate(`/data-types/${schema.id}/edit`)}
                          className="p-2 text-ink-muted/70 hover:text-ink-primary hover:bg-brand-100 rounded-lg transition-colors"
                        >
                          <Edit2 size={18} />
                        </button>
                        <button 
                          onClick={() => setSchemaToDelete(schema.id)}
                          title="Hapus"
                          className="p-2 text-ink-muted/70 hover:text-accent-danger hover:bg-accent-danger/10 rounded-lg transition-colors"
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
        isOpen={schemaToDelete !== null}
        onClose={() => setSchemaToDelete(null)}
        onConfirm={handleDelete}
        title="Hapus Entitas?"
        message="Entitas dan semua datanya yang telah dikumpulkan tidak dapat dikembalikan. Yakin ingin menghapus?"
        confirmLabel="Ya, Hapus"
      />
    </div>
  );
}
