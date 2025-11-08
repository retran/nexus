import React from 'react';
import { DynamicBackground } from '@/components/layout/dynamic-background';
import { ThemeToggle } from '@/components/layout/theme-toggle';
import { LanguageToggle } from '@/components/layout/language-toggle';
import { GlassPanel } from '@/components/auth/glass-panel';

import { SocialLoginButton } from '@/components/auth/social-login-button';
import { SeparatorWithText } from '@/components/auth/separator-with-text';
import { Link } from '@/components/auth/link';
import { EmailLoginForm } from '@/components/auth/login-form';
import { Header } from '@/components/auth/header';

export const LoginPageLayout = () => {
  const [showSocialLogin, setShowSocialLogin] = React.useState(true);

  return (
    <div className="relative flex h-screen w-screen items-center justify-center overflow-hidden">
      <DynamicBackground />
      <div className="absolute right-4 top-4 z-10 flex items-center space-x-2">
        <ThemeToggle />
        <LanguageToggle />
      </div>

      <GlassPanel>
        <Header className="pt-4" />
        <div className="flex flex-1 flex-col items-center justify-center space-y-4">
          {showSocialLogin ? (
            <>
              <SocialLoginButton
                provider="google"
                onClick={() => console.log('Google login')}
              />
              <SocialLoginButton
                provider="apple"
                onClick={() => console.log('Apple login')}
              />
              <SeparatorWithText text="" />
              <Link onClick={() => setShowSocialLogin(false)}>
                Sign in with Email
              </Link>
            </>
          ) : (
            <>
              <EmailLoginForm />
              <SeparatorWithText text="" />
              <Link onClick={() => setShowSocialLogin(true)}>Back</Link>
            </>
          )}
        </div>

        <div className="mt-auto pt-4 text-center text-xs text-black/60 dark:text-white/60">
          © 2025 Andrew Vasilyev. All rights reserved.
        </div>
      </GlassPanel>
    </div>
  );
};
