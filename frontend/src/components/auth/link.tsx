import React from 'react';
import { cn } from '@/lib/utils';

interface LinkProps {
  onClick?: () => void;
  className?: string;
  children: React.ReactNode;
}

export const Link = ({ onClick, className, children }: LinkProps) => {
  return (
    <button
      onClick={onClick}
      className={cn(
        'text-sm text-black/60 dark:text-white/60',
        'hover:text-black/90 dark:hover:text-white/90',
        'focus-visible:rounded-sm focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-black/30 dark:focus-visible:ring-white/30',
        className
      )}
    >
      {children}
    </button>
  );
};
