import { StrictMode } from 'react';
import { createRoot } from 'react-dom/client';
import { ThemeProvider, ThemeColorUpdater } from './components/themes';
import './index.css';
import App from './App.tsx';

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <ThemeProvider defaultTheme="system" storageKey="vite-ui-theme">
      <ThemeColorUpdater />
      <App />
    </ThemeProvider>
  </StrictMode>
);
