import React from 'react';
import { cn } from '@/lib/utils';
import { Separator } from '@/components/ui/separator';
import { useTheme } from '@/components/themes';
import {
  glassTextSecondary,
  glassSeparator,
  glassSeparatorText,
} from '@/lib/glass-styles';

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
    <div className={cn(glassSeparator, className)}>
      <div className="absolute inset-0 flex items-center">
        <Separator
          className={cn(
            resolvedTheme === 'light' ? 'bg-black/10' : 'bg-white/10'
          )}
        />
      </div>
      <div className="relative flex justify-center text-xs uppercase">
        <span className={cn(glassSeparatorText, glassTextSecondary)}>
          {text}
        </span>
      </div>
    </div>
  );
};
