import type { ReactNode } from 'react';
import type { LucideIcon } from 'lucide-react';

interface EmptyStateProps {
  icon: LucideIcon;
  title: string;
  description: string;
  action?: {
    label: string;
    onClick: () => void;
    icon?: LucideIcon;
  };
  children?: ReactNode;
}

export function EmptyState({ 
  icon: Icon, 
  title, 
  description, 
  action,
  children 
}: EmptyStateProps) {
  return (
    <div className="flex flex-col items-center justify-center text-center p-8 md:p-12 border-2 border-dashed border-brand-200 rounded-xl bg-white/50">
      <div className="w-16 h-16 bg-brand-100 text-brand-500 rounded-2xl flex items-center justify-center mb-6 shadow-sm">
        <Icon size={32} />
      </div>
      
      <h3 className="text-lg font-semibold text-brand-900 mb-2">
        {title}
      </h3>
      
      <p className="text-brand-500 max-w-sm mb-8">
        {description}
      </p>
      
      {action && (
        <button
          onClick={action.onClick}
          className="inline-flex items-center gap-2 bg-emerald-600 hover:bg-emerald-700 text-white px-5 py-2.5 rounded-lg font-medium transition-colors shadow-sm active:scale-95"
        >
          {action.icon && <action.icon size={18} />}
          {action.label}
        </button>
      )}

      {children && (
        <div className="mt-8 w-full max-w-md">
          {children}
        </div>
      )}
    </div>
  );
}
