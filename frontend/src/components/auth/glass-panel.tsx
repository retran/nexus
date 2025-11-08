import React from 'react';
import { cn } from '@/lib/utils';

interface GlassPanelProps {
  children: React.ReactNode;
  className?: string;
}

export const GlassPanel: React.FC<GlassPanelProps> = ({
  children,
  className,
}) => {
  return (
    <div
      className={cn(
        'w-sm relative z-10 flex flex-col rounded-2xl',
        'bg-white/40 dark:bg-black/40',
        'border border-white/10',
        'shadow-xl backdrop-blur-xl',
        'md:w-sm md:ml-auto md:mr-[calc(60%-50vw)]',
        'h-[35em] p-8',
        className
      )}
    >
      {children}
    </div>
  );
};
