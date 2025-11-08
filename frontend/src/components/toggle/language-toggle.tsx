'use client';

import * as React from 'react';
import { Languages } from 'lucide-react';
import { ToggleDropdown } from '@/components/toggle/toggle-dropdown';

const languages = [
  { code: 'en', label: 'English' },
  { code: 'ru', label: 'Русский' },
];

export const LanguageToggle: React.FC = () => {
  const [language, setLanguage] = React.useState('en');

  const triggerChildren = (
    <Languages className="h-4 w-4 text-black/90 dark:text-white/90" />
  );

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
