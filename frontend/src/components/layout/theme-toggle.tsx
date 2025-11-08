'use client';

import * as React from 'react';
import { useTheme } from '@/components/themes';
import { Sun, Moon, Monitor, Check } from 'lucide-react';
import { cn } from '@/lib/utils';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';

export const ThemeToggle: React.FC = () => {
  const [mounted, setMounted] = React.useState(false);
  const { theme, setTheme } = useTheme();

  React.useEffect(() => {
    setMounted(true);
  }, []);

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
        <Sun className="h-4 w-4 text-black/90 dark:text-white/90" />
      </div>
    );
  }

  const getCurrentTheme = () => {
    if (theme === 'system') {
      return window.matchMedia('(prefers-color-scheme: dark)').matches
        ? 'dark'
        : 'light';
    }
    return theme;
  };

  const currentTheme = getCurrentTheme();
  const isLightTheme = currentTheme === 'light';

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
        <Sun
          className={cn(
            'h-4 w-4 text-black/90 dark:text-white/90',
            theme === 'light' ? 'rotate-0 scale-100' : '-rotate-90 scale-0'
          )}
        />
        <Moon
          className={cn(
            'absolute h-4 w-4 text-black/90 dark:text-white/90',
            theme === 'dark' ? 'rotate-0 scale-100' : 'rotate-90 scale-0'
          )}
        />
        <Monitor
          className={cn(
            'absolute h-4 w-4 text-black/90 dark:text-white/90',
            theme === 'system' ? 'rotate-0 scale-100' : 'rotate-90 scale-0'
          )}
        />
        <span className="sr-only">Toggle theme</span>
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
        <DropdownMenuItem
          onClick={() => setTheme('light')}
          className={cn(
            'flex cursor-pointer items-center gap-2 rounded-md px-3 py-2',
            'hover:bg-white/20 dark:hover:bg-black/20',
            'focus:bg-white/20 dark:focus:bg-black/20'
          )}
        >
          <Sun
            className={cn(
              'h-4 w-4',
              theme === 'light'
                ? 'text-black/90 dark:text-white/90'
                : 'text-black/60 dark:text-white/60'
            )}
          />
          <span
            className={cn(
              theme === 'light'
                ? 'text-black/90 dark:text-white/90'
                : 'text-black/60 dark:text-white/60'
            )}
          >
            Light
          </span>
          {theme === 'light' && (
            <Check className="ml-auto h-4 w-4 text-black/90 dark:text-white/90" />
          )}
        </DropdownMenuItem>
        <DropdownMenuItem
          onClick={() => setTheme('dark')}
          className={cn(
            'flex cursor-pointer items-center gap-2 rounded-md px-3 py-2',
            'hover:bg-white/20 dark:hover:bg-black/20',
            'focus:bg-white/20 dark:focus:bg-black/20'
          )}
        >
          <Moon
            className={cn(
              'h-4 w-4',
              theme === 'dark'
                ? 'text-black/90 dark:text-white/90'
                : 'text-black/60 dark:text-white/60'
            )}
          />
          <span
            className={cn(
              theme === 'dark'
                ? 'text-black/90 dark:text-white/90'
                : 'text-black/60 dark:text-white/60'
            )}
          >
            Dark
          </span>
          {theme === 'dark' && (
            <Check className="ml-auto h-4 w-4 text-black dark:text-white" />
          )}
        </DropdownMenuItem>
        <DropdownMenuItem
          onClick={() => setTheme('system')}
          className={cn(
            'flex cursor-pointer items-center gap-2 rounded-md px-3 py-2',
            'hover:bg-white/20 dark:hover:bg-black/20',
            'focus:bg-white/20 dark:focus:bg-black/20'
          )}
        >
          <Monitor
            className={cn(
              'h-4 w-4',
              theme === 'system'
                ? 'text-black/90 dark:text-white/90'
                : 'text-black/60 dark:text-white/60'
            )}
          />
          <span
            className={cn(
              theme === 'system'
                ? 'text-black/90 dark:text-white/90'
                : 'text-black/60 dark:text-white/60'
            )}
          >
            System
          </span>
          {theme === 'system' && (
            <Check className="ml-auto h-4 w-4 text-black/90 dark:text-white/90" />
          )}
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  );
};
