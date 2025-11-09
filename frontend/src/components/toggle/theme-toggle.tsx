'use client';

import * as React from 'react';
import { Sun, Moon, Monitor } from 'lucide-react';
import { useTheme } from '@/components/themes';
import { cn } from '@/lib/utils';
import { ToggleDropdown } from '@/components/toggle/toggle-dropdown';
import { glassText } from '@/lib/glass-styles';

const themeOptions = [
  { code: 'light', label: 'Light', icon: Sun },
  { code: 'dark', label: 'Dark', icon: Moon },
  { code: 'system', label: 'System', icon: Monitor },
];

export const ThemeToggle: React.FC = () => {
  const { theme, setTheme } = useTheme();

  const triggerChildren = (
    <>
      <Sun
        className={cn(
          'h-4 w-4',
          glassText,
          theme === 'light' ? 'rotate-0 scale-100' : '-rotate-90 scale-0'
        )}
      />
      <Moon
        className={cn(
          'absolute h-4 w-4',
          glassText,
          theme === 'dark' ? 'rotate-0 scale-100' : 'rotate-90 scale-0'
        )}
      />
      <Monitor
        className={cn(
          'absolute h-4 w-4',
          glassText,
          theme === 'system' ? 'rotate-0 scale-100' : 'rotate-90 scale-0'
        )}
      />
    </>
  );

  return (
    <ToggleDropdown
      options={themeOptions}
      current={theme}
      setCurrent={setTheme as (code: string) => void}
      triggerChildren={triggerChildren}
      placeholderIcon={Sun}
    />
  );
};
