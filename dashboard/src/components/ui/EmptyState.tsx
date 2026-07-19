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
    <div className="flex flex-col items-center justify-center text-center p-8 md:p-12 border-2 border-dashed border-ink-primary/10 rounded-[16px] bg-surface">
      <div className="w-16 h-16 bg-canvas text-ink-muted rounded-2xl flex items-center justify-center mb-6 shadow-soft">
        <Icon size={32} strokeWidth={1.5} />
      </div>
      
      <h3 className="text-lg font-sora font-semibold text-ink-primary mb-2">
        {title}
      </h3>
      
      <p className="text-ink-muted max-w-sm mb-8">
        {description}
      </p>
      
      {action && (
        <button
          onClick={action.onClick}
          className="inline-flex items-center gap-2 bg-accent-primary text-white px-5 py-2.5 rounded-lg font-medium shadow-sm hover:-translate-y-[2px] hover:shadow-soft active:scale-95 transition-all duration-150"
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
