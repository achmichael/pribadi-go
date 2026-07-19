import React, { useState, useRef } from 'react';
import { UploadCloud, AlertCircle, Loader2, X } from 'lucide-react';

interface FileUploadProps {
  onUploadSuccess: (file: File) => void;
  maxSizeMB?: number;
  acceptedTypes?: string[]; // e.g. ['application/pdf', 'text/plain']
}

export function FileUpload({ 
  onUploadSuccess, 
  maxSizeMB = 5,
  acceptedTypes = ['application/pdf', 'text/plain']
}: FileUploadProps) {
  const [isDragging, setIsDragging] = useState(false);
  const [isUploading, setIsUploading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const fileInputRef = useRef<HTMLInputElement>(null);

  const maxSizeBytes = maxSizeMB * 1024 * 1024;

  const validateAndProcessFile = (file: File) => {
    setError(null);
    
    if (!acceptedTypes.includes(file.type) && !file.name.endsWith('.txt')) {
      setError('Hanya format .PDF dan .TXT yang didukung.');
      return;
    }

    if (file.size > maxSizeBytes) {
      setError(`Ukuran file terlalu besar. Maksimal ${maxSizeMB}MB.`);
      return;
    }

    setIsUploading(true);
    
    // Simulate upload delay
    setTimeout(() => {
      setIsUploading(false);
      onUploadSuccess(file);
    }, 2000);
  };

  const handleDragOver = (e: React.DragEvent) => {
    e.preventDefault();
    setIsDragging(true);
  };

  const handleDragLeave = (e: React.DragEvent) => {
    e.preventDefault();
    setIsDragging(false);
  };

  const handleDrop = (e: React.DragEvent) => {
    e.preventDefault();
    setIsDragging(false);
    
    const files = e.dataTransfer.files;
    if (files.length > 0) {
      validateAndProcessFile(files[0]);
    }
  };

  const handleFileSelect = (e: React.ChangeEvent<HTMLInputElement>) => {
    const files = e.target.files;
    if (files && files.length > 0) {
      validateAndProcessFile(files[0]);
    }
  };

  return (
    <div className="w-full">
      <div 
        onDragOver={handleDragOver}
        onDragLeave={handleDragLeave}
        onDrop={handleDrop}
        onClick={() => !isUploading && fileInputRef.current?.click()}
        className={`relative border-2 border-dashed rounded-2xl p-8 flex flex-col items-center justify-center text-center transition-all ${
          isUploading 
            ? 'border-ink-primary/10 bg-canvas cursor-default'
            : isDragging
              ? 'border-accent-primary bg-accent-primary/10/50'
              : 'border-ink-primary/10 bg-surface hover:bg-canvas hover:border-ink-primary/20 cursor-pointer'
        }`}
      >
        <input 
          type="file" 
          ref={fileInputRef} 
          className="hidden" 
          accept=".pdf,.txt"
          onChange={handleFileSelect}
          disabled={isUploading}
        />

        {isUploading ? (
          <div className="flex flex-col items-center text-accent-primary animate-in fade-in duration-300">
            <Loader2 size={40} className="animate-spin mb-4" />
            <span className="font-semibold text-ink-primary">Mengunggah...</span>
            <span className="text-sm text-ink-muted mt-1">Sedang dipelajari AI, butuh waktu beberapa saat.</span>
          </div>
        ) : (
          <>
            <div className={`p-4 rounded-full mb-4 transition-colors ${isDragging ? 'bg-emerald-100 text-accent-primary' : 'bg-brand-100 text-ink-muted'}`}>
              <UploadCloud size={32} />
            </div>
            <h3 className="font-sora font-semibold text-ink-primary mb-1">
              {isDragging ? 'Lepaskan file di sini' : 'Klik atau seret file ke area ini'}
            </h3>
            <p className="text-sm text-ink-muted">
              Format PDF atau TXT (Maks. {maxSizeMB}MB)
            </p>
          </>
        )}
      </div>

      {error && (
        <div className="mt-3 flex items-center gap-2 text-accent-danger bg-accent-danger/10 px-4 py-3 rounded-xl border border-rose-100 animate-in fade-in slide-in-from-top-2 duration-300">
          <AlertCircle size={18} />
          <span className="text-sm font-medium">{error}</span>
          <button 
            onClick={() => setError(null)}
            className="ml-auto text-rose-400 hover:text-rose-700"
          >
            <X size={16} />
          </button>
        </div>
      )}
    </div>
  );
}
