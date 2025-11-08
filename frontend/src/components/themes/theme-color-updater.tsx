import { useEffect } from 'react';
import { useTheme } from './index';

export function ThemeColorUpdater() {
  const { resolvedTheme } = useTheme();

  useEffect(() => {
    const updateThemeColor = () => {
      // Remove existing theme-color meta tags
      const existingMetaTags = document.querySelectorAll(
        'meta[name="theme-color"]'
      );
      existingMetaTags.forEach((tag) => tag.remove());

      // Add new theme-color meta tag
      const meta = document.createElement('meta');
      meta.name = 'theme-color';

      if (resolvedTheme === 'dark') {
        meta.content = '#0f172a'; // Dark slate color for dark theme
      } else {
        meta.content = '#f1f5f9'; // Light slate color for light theme
      }

      document.head.appendChild(meta);
    };

    updateThemeColor();
  }, [resolvedTheme]);

  return null;
}
