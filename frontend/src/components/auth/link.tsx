import React from 'react';
import { cn } from '@/lib/utils';
import { tertiaryAction } from '@/lib/glass-styles';

interface LinkProps {
  onClick?: () => void;
  className?: string;
  children: React.ReactNode;
}

export const Link = ({ onClick, className, children }: LinkProps) => {
  return (
    <button onClick={onClick} className={cn(tertiaryAction, className)}>
      {children}
    </button>
  );
};
