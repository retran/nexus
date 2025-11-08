'use client';

import * as React from 'react';
import { Languages, Check } from 'lucide-react';
import { useTheme } from '@/components/themes';
import { cn } from '@/lib/utils';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';

const languages = [
  { code: 'en', label: 'English' },
  { code: 'ru', label: 'Русский' },
];

export const LanguageToggle: React.FC = () => {
  const [mounted, setMounted] = React.useState(false);
  const [language, setLanguage] = React.useState('en');
  const { theme } = useTheme();

  const getCurrentTheme = () => {
    if (theme === 'system') {
      return window.matchMedia('(prefers-color-scheme: dark)').matches
        ? 'dark'
        : 'light';
    }
    return theme;
  };

  const currentTheme = getCurrentTheme();

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
        <Languages className="h-4 w-4 text-black/90 dark:text-white/90" />
      </div>
    );
  }

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
        <Languages className="h-4 w-4 text-black/90 dark:text-white/90" />
        <span className="sr-only">Change language</span>
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
        {languages.map((lang) => (
          <DropdownMenuItem
            key={lang.code}
            onClick={() => setLanguage(lang.code)}
            className={cn(
              'flex cursor-pointer items-center gap-2 rounded-md px-3 py-2',
              'hover:bg-white/20 dark:hover:bg-black/20',
              'focus:bg-white/20 dark:focus:bg-black/20'
            )}
          >
            <span
              className={cn(
                language === lang.code
                  ? 'text-black/90 dark:text-white/90'
                  : 'text-black/60 dark:text-white/60'
              )}
            >
              {lang.label}
            </span>
            {language === lang.code && (
              <Check className="ml-auto h-4 w-4 text-black/90 dark:text-white/90" />
            )}
          </DropdownMenuItem>
        ))}
      </DropdownMenuContent>
    </DropdownMenu>
  );
};
