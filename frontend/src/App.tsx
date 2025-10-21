import {
  AuthProvider,
  GitHubBanner,
  Refine,
  WelcomePage,
} from '@refinedev/core';
import { DevtoolsPanel, DevtoolsProvider } from '@refinedev/devtools';
import { RefineKbar, RefineKbarProvider } from '@refinedev/kbar';

import routerProvider, {
  DocumentTitleHandler,
  UnsavedChangesNotifier,
} from '@refinedev/react-router';
import dataProvider from '@refinedev/simple-rest';
import axios from 'axios';
import { BrowserRouter, Route, Routes } from 'react-router';
import './App.css';
import { Toaster } from './components/refine-ui/notification/toaster';
import { useNotificationProvider } from './components/refine-ui/notification/use-notification-provider';
import { ThemeProvider } from './components/refine-ui/theme/theme-provider';
import { CredentialResponse } from './interfaces/google';
import { parseJwt } from './utils/parse-jwt';

const axiosInstance = axios.create();
axiosInstance.interceptors.request.use((config) => {
  const token = localStorage.getItem('token');
  if (config.headers) {
    config.headers['Authorization'] = `Bearer ${token}`;
  }

  return config;
});

function App() {
  const authProvider: AuthProvider = {
    login: async ({ credential }: CredentialResponse) => {
      const profileObj = credential ? parseJwt(credential) : null;

      if (profileObj) {
        localStorage.setItem(
          'user',
          JSON.stringify({
            ...profileObj,
            avatar: profileObj.picture,
          })
        );

        localStorage.setItem('token', `${credential}`);

        return {
          success: true,
          redirectTo: '/',
        };
      }

      return {
        success: false,
      };
    },
    logout: async () => {
      const token = localStorage.getItem('token');

      if (token && typeof window !== 'undefined') {
        localStorage.removeItem('token');
        localStorage.removeItem('user');
        axios.defaults.headers.common = {};
        window.google?.accounts.id.revoke(token, () => {
          return {};
        });
      }

      return {
        success: true,
        redirectTo: '/login',
      };
    },
    onError: async (error) => {
      console.error(error);
      return { error };
    },
    check: async () => {
      const token = localStorage.getItem('token');

      if (token) {
        return {
          authenticated: true,
        };
      }

      return {
        authenticated: false,
        error: {
          message: 'Check failed',
          name: 'Token not found',
        },
        logout: true,
        redirectTo: '/login',
      };
    },
    getPermissions: async () => null,
    getIdentity: async () => {
      const user = localStorage.getItem('user');
      if (user) {
        return JSON.parse(user);
      }

      return null;
    },
  };

  return (
    <BrowserRouter>
      <GitHubBanner />
      <RefineKbarProvider>
        <ThemeProvider>
          <DevtoolsProvider>
            <Refine
              dataProvider={dataProvider('https://api.fake-rest.refine.dev')}
              notificationProvider={useNotificationProvider()}
              routerProvider={routerProvider}
              authProvider={authProvider}
              options={{
                syncWithLocation: true,
                warnWhenUnsavedChanges: true,
                projectId: 'NWRwq0-W5A3Rk-n1OMK0',
              }}
            >
              <Routes>
                <Route index element={<WelcomePage />} />
              </Routes>
              <Toaster />
              <RefineKbar />
              <UnsavedChangesNotifier />
              <DocumentTitleHandler />
            </Refine>
            <DevtoolsPanel />
          </DevtoolsProvider>
        </ThemeProvider>
      </RefineKbarProvider>
    </BrowserRouter>
  );
}

export default App;
