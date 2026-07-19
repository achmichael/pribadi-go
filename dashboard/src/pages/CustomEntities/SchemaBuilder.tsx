import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import Form from '@rjsf/core';
import validator from '@rjsf/validator-ajv8';
import { 
  Plus, 
  Trash2, 
  ArrowLeft,
  Save,
  LayoutTemplate
} from 'lucide-react';

interface SchemaField {
  id: string;
  name: string;
  type: 'string' | 'number' | 'boolean';
  title: string;
  required: boolean;
}

export default function SchemaBuilder() {
  const navigate = useNavigate();
  
  // Basic Schema Info
  const [entityName, setEntityName] = useState('');
  const [entityDesc, setEntityDesc] = useState('');
  
  // Field Builder State
  const [fields, setFields] = useState<SchemaField[]>([
    { id: '1', name: 'namaLengkap', title: 'Nama Lengkap', type: 'string', required: true }
  ]);

  // Derived JSON Schema for Preview
  const getJsonSchema = () => {
    const properties: any = {};
    const required: string[] = [];

    fields.forEach(f => {
      properties[f.name] = {
        type: f.type,
        title: f.title
      };
      if (f.required) required.push(f.name);
    });

    return {
      title: entityName || 'Pratinjau Formulir',
      description: entityDesc || 'Isi formulir ini untuk melihat pratinjau.',
      type: 'object',
      required,
      properties
    };
  };

  const addField = () => {
    const newId = Math.random().toString();
    setFields([
      ...fields, 
      { id: newId, name: `field_${newId.slice(2,6)}`, title: 'Label Baru', type: 'string', required: false }
    ]);
  };

  const updateField = (id: string, updates: Partial<SchemaField>) => {
    setFields(fields.map(f => f.id === id ? { ...f, ...updates } : f));
  };

  const removeField = (id: string) => {
    setFields(fields.filter(f => f.id !== id));
  };

  return (
    <div className="space-y-6 animate-in fade-in duration-500 pb-24 lg:pb-8">
      
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
            <h1 className="font-sora text-2xl font-bold text-ink-primary">Buat Entitas Baru</h1>
            <p className="text-sm text-ink-muted">Rancang struktur data yang akan ditangkap oleh AI.</p>
          </div>
        </div>
        
        <button
          onClick={() => navigate('/data-types')} // Mock save
          disabled={!entityName || fields.length === 0}
          className="flex items-center justify-center gap-2 bg-accent-primary text-white px-6 py-2.5 rounded-xl font-medium hover:bg-accent-primary/90 transition-colors disabled:opacity-50 disabled:cursor-not-allowed shadow-sm"
        >
          <Save size={18} />
          <span>Simpan Skema</span>
        </button>
      </div>

      {/* Split Screen Layout */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6 items-start">
        
        {/* LEFT PANEL: Builder */}
        <div className="space-y-6">
          
          {/* Info Card */}
          <div className="bg-surface border border-ink-primary/10 rounded-2xl p-6 shadow-sm">
            <h2 className="font-sora font-bold text-ink-primary mb-4">Informasi Dasar</h2>
            <div className="space-y-4">
              <div>
                <label className="block text-sm font-semibold text-ink-primary mb-1.5 ml-1">Nama Entitas</label>
                <input
                  type="text"
                  value={entityName}
                  onChange={(e) => setEntityName(e.target.value)}
                  className="w-full bg-canvas border border-ink-primary/10 rounded-xl px-4 py-2 text-ink-primary focus:bg-surface focus:ring-2 focus:ring-accent-primary/15 focus:border-accent-primary outline-none transition-all"
                  placeholder="Misal: Order Penjualan"
                />
              </div>
              <div>
                <label className="block text-sm font-semibold text-ink-primary mb-1.5 ml-1">Deskripsi (Untuk konteks AI)</label>
                <textarea
                  value={entityDesc}
                  onChange={(e) => setEntityDesc(e.target.value)}
                  rows={2}
                  className="w-full bg-canvas border border-ink-primary/10 rounded-xl px-4 py-2 text-ink-primary focus:bg-surface focus:ring-2 focus:ring-accent-primary/15 focus:border-accent-primary outline-none transition-all resize-y"
                  placeholder="Formulir ini digunakan untuk..."
                />
              </div>
            </div>
          </div>

          {/* Fields Card */}
          <div className="bg-surface border border-ink-primary/10 rounded-2xl p-6 shadow-sm">
            <div className="flex items-center justify-between mb-4">
              <h2 className="font-sora font-bold text-ink-primary">Struktur Field</h2>
              <button 
                onClick={addField}
                className="flex items-center gap-1.5 text-sm font-semibold text-accent-primary hover:text-accent-primary transition-colors"
              >
                <Plus size={16} /> Tambah Field
              </button>
            </div>

            <div className="space-y-4">
              {fields.map((field) => (
                <div key={field.id} className="relative bg-canvas border border-ink-primary/10 p-4 rounded-xl space-y-3">
                  <div className="absolute top-4 right-4">
                    <button 
                      onClick={() => removeField(field.id)}
                      className="text-ink-muted/70 hover:text-rose-500 transition-colors"
                    >
                      <Trash2 size={16} />
                    </button>
                  </div>
                  
                  <div className="grid grid-cols-2 gap-3 pr-8">
                    <div>
                      <label className="block text-xs font-semibold text-ink-primary mb-1 ml-1">Nama Field (JSON Key)</label>
                      <input
                        type="text"
                        value={field.name}
                        onChange={(e) => updateField(field.id, { name: e.target.value.replace(/\s+/g, '') })}
                        className="w-full bg-surface border border-ink-primary/10 rounded-lg px-3 py-1.5 text-sm text-ink-primary outline-none focus:border-accent-primary"
                      />
                    </div>
                    <div>
                      <label className="block text-xs font-semibold text-ink-primary mb-1 ml-1">Tipe Data</label>
                      <select
                        value={field.type}
                        onChange={(e) => updateField(field.id, { type: e.target.value as any })}
                        className="w-full bg-surface border border-ink-primary/10 rounded-lg px-3 py-1.5 text-sm text-ink-primary outline-none focus:border-accent-primary"
                      >
                        <option value="string">Teks (String)</option>
                        <option value="number">Angka (Number)</option>
                        <option value="boolean">Ya/Tidak (Boolean)</option>
                      </select>
                    </div>
                  </div>
                  
                  <div>
                    <label className="block text-xs font-semibold text-ink-primary mb-1 ml-1">Label (Ditampilkan di UI)</label>
                    <input
                      type="text"
                      value={field.title}
                      onChange={(e) => updateField(field.id, { title: e.target.value })}
                      className="w-full bg-surface border border-ink-primary/10 rounded-lg px-3 py-1.5 text-sm text-ink-primary outline-none focus:border-accent-primary"
                    />
                  </div>

                  <label className="flex items-center gap-2 cursor-pointer mt-2 w-max">
                    <input 
                      type="checkbox" 
                      checked={field.required}
                      onChange={(e) => updateField(field.id, { required: e.target.checked })}
                      className="rounded text-accent-primary focus:ring-accent-primary/15"
                    />
                    <span className="text-xs font-semibold text-ink-primary">Wajib diisi</span>
                  </label>
                </div>
              ))}
            </div>
          </div>
        </div>

        {/* RIGHT PANEL: Live Preview */}
        <div className="bg-surface border border-ink-primary/10 rounded-2xl overflow-hidden shadow-sm sticky top-24">
          <div className="bg-brand-900 text-white px-5 py-3 flex items-center gap-2">
            <LayoutTemplate size={18} className="text-emerald-400" />
            <h2 className="font-sora font-semibold text-sm">Live Preview</h2>
          </div>
          <div className="p-6 overflow-y-auto max-h-[600px] bg-canvas/30">
            {fields.length > 0 ? (
              <div className="rjsf-preview-wrapper bg-surface p-6 rounded-xl border border-brand-100 shadow-sm">
                <Form 
                  schema={getJsonSchema() as any} 
                  validator={validator} 
                  uiSchema={{
                    "ui:submitButtonOptions": { norender: true } // Hide default submit button
                  }}
                  className="space-y-4"
                />
              </div>
            ) : (
              <div className="text-center py-12 text-ink-muted/70">
                <LayoutTemplate size={48} className="mx-auto mb-3 opacity-20" />
                <p className="text-sm">Tambahkan field untuk melihat pratinjau.</p>
              </div>
            )}
          </div>
        </div>

      </div>
    </div>
  );
}
