import React from 'react';
import { cn } from '@/lib/utils';

interface SeparatorWithTextProps {
  text: string;
  className?: string;
}

export const SeparatorWithText: React.FC<SeparatorWithTextProps> = ({
  text,
  className,
}) => {
  return (
    <div className={cn('relative w-full max-w-[280px]', className)}>
      <div className="absolute inset-0 flex items-center">
        <span className="w-full border-t border-white/10" />
      </div>
      <div className="relative flex justify-center text-xs uppercase">
        <span
          className={cn(
            'px-2',
            'bg-white/10 dark:bg-black/10',
            'text-black/60 dark:text-white/60'
          )}
        >
          {text}
        </span>
      </div>
    </div>
  );
};
