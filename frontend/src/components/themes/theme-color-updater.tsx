import { useEffect } from 'react';
import { useTheme } from './index';

export function ThemeColorUpdater() {
  const { theme } = useTheme();

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

      // Determine the actual theme (resolve "system" to actual theme)
      let currentTheme = theme;
      if (theme === 'system') {
        currentTheme = window.matchMedia('(prefers-color-scheme: dark)').matches
          ? 'dark'
          : 'light';
      }

      if (currentTheme === 'dark') {
        meta.content = '#0f172a'; // Dark slate color for dark theme
      } else {
        meta.content = '#f1f5f9'; // Light slate color for light theme
      }

      document.head.appendChild(meta);
    };

    updateThemeColor();

    // Listen for system theme changes
    const mediaQuery = window.matchMedia('(prefers-color-scheme: dark)');
    const handleSystemThemeChange = () => {
      if (theme === 'system') {
        updateThemeColor();
      }
    };

    mediaQuery.addEventListener('change', handleSystemThemeChange);

    // Listen for theme changes
    const observer = new MutationObserver(updateThemeColor);
    observer.observe(document.documentElement, {
      attributes: true,
      attributeFilter: ['class'],
    });

    return () => {
      mediaQuery.removeEventListener('change', handleSystemThemeChange);
      observer.disconnect();
    };
  }, [theme]);

  return null;
}
