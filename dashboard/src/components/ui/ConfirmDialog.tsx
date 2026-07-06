import { Modal } from './Modal';
import { AlertTriangle } from 'lucide-react';

interface ConfirmDialogProps {
  isOpen: boolean;
  onClose: () => void;
  onConfirm: () => void;
  title: string;
  message: string;
  confirmLabel?: string;
  cancelLabel?: string;
  isDestructive?: boolean;
}

export function ConfirmDialog({
  isOpen,
  onClose,
  onConfirm,
  title,
  message,
  confirmLabel = 'Ya, Lanjutkan',
  cancelLabel = 'Batal',
  isDestructive = true
}: ConfirmDialogProps) {
  return (
    <Modal isOpen={isOpen} onClose={onClose} maxWidth="max-w-sm">
      <div className="flex flex-col items-center text-center space-y-4 pt-4">
        
        {/* Warning Icon */}
        <div className={`p-4 rounded-full ${isDestructive ? 'bg-rose-100 text-rose-600' : 'bg-amber-100 text-amber-600'}`}>
          <AlertTriangle size={32} />
        </div>
        
        {/* Texts */}
        <div className="space-y-2">
          <h3 className="text-xl font-bold text-brand-900">{title}</h3>
          <p className="text-brand-500 text-sm">{message}</p>
        </div>

        {/* Actions */}
        <div className="flex flex-col w-full gap-3 pt-4">
          <button
            onClick={() => {
              onConfirm();
              onClose();
            }}
            className={`w-full py-3 rounded-xl font-medium transition-colors ${
              isDestructive 
                ? 'bg-rose-600 hover:bg-rose-700 text-white' 
                : 'bg-emerald-600 hover:bg-emerald-700 text-white'
            }`}
          >
            {confirmLabel}
          </button>
          
          <button
            onClick={onClose}
            className="w-full py-3 rounded-xl font-medium text-brand-600 hover:bg-brand-50 transition-colors"
          >
            {cancelLabel}
          </button>
        </div>
      </div>
    </Modal>
  );
}
