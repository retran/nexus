'use client';

import * as React from 'react';
import { Check } from 'lucide-react';
import { cn } from '@/lib/utils';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';
import {
  glassDropdown,
  glassDropdownItem,
  glassText,
  glassToggleTrigger,
} from '@/lib/glass-styles';

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

  React.useEffect(() => {
    setMounted(true);
  }, []);

  if (!mounted) {
    return (
      <div className={cn(glassToggleTrigger, 'opacity-50')}>
        <PlaceholderIcon className={cn('h-4 w-4', glassText)} />
      </div>
    );
  }

  return (
    <DropdownMenu>
      <DropdownMenuTrigger className={glassToggleTrigger}>
        {triggerChildren}
        <span className="sr-only">Toggle</span>
      </DropdownMenuTrigger>
      <DropdownMenuContent
        align="end"
        sideOffset={8}
        className={cn('min-w-[140px] p-1', glassDropdown)}
      >
        {options.map((option) => (
          <DropdownMenuItem
            key={option.code}
            onClick={() => setCurrent(option.code)}
            className={cn(
              'flex cursor-pointer items-center gap-2 rounded-md px-3 py-2',
              glassDropdownItem
            )}
          >
            {option.icon && (
              <option.icon
                className={cn(
                  'h-4 w-4',
                  current === option.code
                    ? glassText
                    : 'text-black/60 dark:text-white/60'
                )}
              />
            )}
            <span
              className={cn(
                current === option.code
                  ? glassText
                  : 'text-black/60 dark:text-white/60'
              )}
            >
              {option.label}
            </span>
            {current === option.code && (
              <Check className={cn('ml-auto h-4 w-4', glassText)} />
            )}
          </DropdownMenuItem>
        ))}
      </DropdownMenuContent>
    </DropdownMenu>
  );
};
