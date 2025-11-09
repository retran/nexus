'use client';

import * as React from 'react';
import { Languages } from 'lucide-react';
import { ToggleDropdown } from '@/components/toggle/toggle-dropdown';
import { cn } from '@/lib/utils';
import { glassText } from '@/lib/glass-styles';

const languages = [
  { code: 'en', label: 'English' },
  { code: 'ru', label: 'Русский' },
];

export const LanguageToggle: React.FC = () => {
  const [language, setLanguage] = React.useState('en');

  const triggerChildren = <Languages className={cn('h-4 w-4', glassText)} />;

  return (
    <ToggleDropdown
      options={languages}
      current={language}
      setCurrent={setLanguage}
      triggerChildren={triggerChildren}
      placeholderIcon={Languages}
    />
  );
};
