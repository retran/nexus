import React from 'react';
import { Button } from '@/components/ui/button';
import { cn } from '@/lib/utils';
import { useTheme } from '@/components/themes';
import googleIcon from '@/assets/google.svg';
import appleIcon from '@/assets/apple.svg';

interface SocialLoginButtonProps {
  provider: 'google' | 'apple';
  onClick?: () => void;
  className?: string;
}

export const SocialLoginButton: React.FC<SocialLoginButtonProps> = ({
  provider,
  onClick,
  className,
}) => {
  const { theme } = useTheme();
  const iconSrc = provider === 'google' ? googleIcon : appleIcon;
  const label =
    provider === 'google' ? 'Sign in with Google' : 'Sign in with Apple';
  const isDark =
    theme === 'dark' ||
    (theme === 'system' &&
      window.matchMedia('(prefers-color-scheme: dark)').matches);
  const iconClass =
    'h-4 w-4 shrink-0 object-contain align-middle' +
    (provider === 'apple' ? ' -mt-0.5' : '');
  return (
    <Button
      variant="ghost"
      onClick={onClick}
      className={cn(
        'flex h-10 w-full max-w-[280px] items-center justify-center gap-2',
        'rounded-lg border border-white/10',
        'bg-white/20 dark:bg-black/20',
        'font-semibold text-black/90 dark:text-white/90',
        'hover:bg-white/30 hover:text-black/90 dark:hover:bg-black/30 dark:hover:text-white/90',
        'focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-white/30',
        className
      )}
    >
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
