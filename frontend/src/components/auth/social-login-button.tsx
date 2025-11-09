import React from 'react';
import { Button } from '@/components/ui/button';
import { secondaryCTA } from '@/lib/glass-styles';
import googleIcon from '@/assets/google.svg';
import appleIcon from '@/assets/apple.svg';
import { useTheme } from '@/components/themes';

interface SocialLoginButtonProps {
  provider: 'google' | 'apple';
  onClick?: () => void;
}

export const SocialLoginButton: React.FC<SocialLoginButtonProps> = ({
  provider,
  onClick,
}) => {
  const { resolvedTheme } = useTheme();
  const iconSrc = provider === 'google' ? googleIcon : appleIcon;
  const label =
    provider === 'google' ? 'Sign in with Google' : 'Sign in with Apple';
  const isDark = resolvedTheme === 'dark';
  const iconClass =
    'h-4 w-4 shrink-0 object-contain align-middle' +
    (provider === 'apple' ? ' -mt-0.5' : '');
  return (
    <Button variant="ghost" onClick={onClick} className={secondaryCTA}>
      <img
        src={iconSrc}
        alt={`${provider} icon`}
        className={iconClass}
        style={{
          filter: provider === 'apple' && isDark ? 'invert(1)' : 'none',
        }}
      />
      <span>{label}</span>
    </Button>
  );
};
