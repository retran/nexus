import { useContext } from 'react';

import { ThemeProviderContext } from './theme-context';

export function useTheme() {
  const context = useContext(ThemeProviderContext);

  if (context === undefined) {
    console.error('useTheme must be used within a ThemeProvider');
  }

  return context;
}
