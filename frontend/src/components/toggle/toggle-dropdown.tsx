'use client';

import * as React from 'react';
import { Check } from 'lucide-react';
import { useTheme } from '@/components/themes';
import { cn } from '@/lib/utils';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';

interface Option {
  code: string;
  label: string;
  icon?: React.ComponentType<{ className?: string }>;
}

interface ToggleDropdownProps {
  options: Option[];
  current: string;
  setCurrent: (code: string) => void;
  triggerChildren: React.ReactNode;
  placeholderIcon: React.ComponentType<{ className?: string }>;
}

export const ToggleDropdown: React.FC<ToggleDropdownProps> = ({
  options,
  current,
  setCurrent,
  triggerChildren,
  placeholderIcon: PlaceholderIcon,
}) => {
  const [mounted, setMounted] = React.useState(false);
  const { resolvedTheme } = useTheme();

  React.useEffect(() => {
    setMounted(true);
  }, []);

  const isLightTheme = resolvedTheme === 'light';

  if (!mounted) {
    return (
      <div
        className={cn(
          'h-9 w-9 rounded-lg',
          'border border-white/10',
          'bg-white/40 dark:bg-black/40',
          'backdrop-blur-xl',
          'flex items-center justify-center',
          'opacity-50'
        )}
      >
        <PlaceholderIcon className="h-4 w-4 text-black/90 dark:text-white/90" />
      </div>
    );
  }

  return (
    <DropdownMenu>
      <DropdownMenuTrigger
        className={cn(
          'h-9 w-9 rounded-lg',
          'border border-white/10',
          'flex items-center justify-center',
          'focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-black/30 dark:focus-visible:ring-white/30',
          'relative overflow-hidden'
        )}
        style={{
          backgroundColor: isLightTheme
            ? 'rgba(255, 255, 255, 0.4)'
            : 'rgba(0, 0, 0, 0.4)',
          backdropFilter: 'blur(16px)',
          WebkitBackdropFilter: 'blur(16px)',
        }}
        onMouseEnter={(e) => {
          e.currentTarget.style.backgroundColor = isLightTheme
            ? 'rgba(255, 255, 255, 0.5)'
            : 'rgba(0, 0, 0, 0.5)';
        }}
        onMouseLeave={(e) => {
          e.currentTarget.style.backgroundColor = isLightTheme
            ? 'rgba(255, 255, 255, 0.4)'
            : 'rgba(0, 0, 0, 0.4)';
        }}
      >
        {triggerChildren}
        <span className="sr-only">Toggle</span>
      </DropdownMenuTrigger>
      <DropdownMenuContent
        align="end"
        sideOffset={8}
        className={cn(
          'min-w-[140px] rounded-lg p-1',
          'border border-white/10',
          'bg-white/40 dark:bg-black/40',
          'backdrop-blur-xl',
          'shadow-xl'
        )}
      >
        {options.map((option) => (
          <DropdownMenuItem
            key={option.code}
            onClick={() => setCurrent(option.code)}
            className={cn(
              'flex cursor-pointer items-center gap-2 rounded-md px-3 py-2',
              'hover:bg-white/20 dark:hover:bg-black/20',
              'focus:bg-white/20 dark:focus:bg-black/20'
            )}
          >
            {option.icon && (
              <option.icon
                className={cn(
                  'h-4 w-4',
                  current === option.code
                    ? 'text-black/90 dark:text-white/90'
                    : 'text-black/60 dark:text-white/60'
                )}
              />
            )}
            <span
              className={cn(
                current === option.code
                  ? 'text-black/90 dark:text-white/90'
                  : 'text-black/60 dark:text-white/60'
              )}
            >
              {option.label}
            </span>
            {current === option.code && (
              <Check className="ml-auto h-4 w-4 text-black/90 dark:text-white/90" />
            )}
          </DropdownMenuItem>
        ))}
      </DropdownMenuContent>
    </DropdownMenu>
  );
};
