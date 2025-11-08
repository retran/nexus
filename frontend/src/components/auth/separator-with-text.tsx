import React from 'react';
import { cn } from '@/lib/utils';
import { Separator } from '@/components/ui/separator';
import { useTheme } from '@/components/themes';

interface SeparatorWithTextProps {
  text: string;
  className?: string;
}

export const SeparatorWithText: React.FC<SeparatorWithTextProps> = ({
  text,
  className,
}) => {
  const { resolvedTheme } = useTheme();

  return (
    <div className={cn('relative w-full max-w-[280px]', className)}>
      <div className="absolute inset-0 flex items-center">
        <Separator
          className={cn(
            resolvedTheme === 'light' ? 'bg-black/10' : 'bg-white/10'
          )}
        />
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
