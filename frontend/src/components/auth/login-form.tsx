import { cn } from '@/lib/utils';
import { Input } from '@/components/ui/input';
import { Button } from '@/components/ui/button';
import { Checkbox } from '@/components/ui/checkbox';
import { Label } from '@/components/ui/label';

export const EmailLoginForm = () => {
  const glassInputStyle = cn(
    'h-10 rounded-lg border border-white/10',
    'bg-white/20 dark:bg-black/20',
    'text-black/90 placeholder:text-black/60 dark:text-white/90 dark:placeholder:text-white/60',
    'focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-black/30 dark:focus-visible:ring-white/30'
  );

  const glassLabelStyle = cn('text-sm text-black/90 dark:text-white/90');

  const glassCheckboxStyle = cn(
    'border-white/10 bg-white/20 dark:bg-black/20',
    'data-[state=checked]:border-white/30 data-[state=checked]:bg-white/30',
    'focus-visible:ring-1 focus-visible:ring-black/30 dark:focus-visible:ring-white/30'
  );

  const glassButtonStyle = cn(
    'flex h-10 w-full items-center justify-center gap-2',
    'rounded-lg border border-white/10',
    'bg-white/20 dark:bg-black/20',
    'font-semibold text-black/90 dark:text-white/90',
    'hover:bg-white/30 hover:text-black dark:hover:bg-black/30 dark:hover:text-white/90',
    'focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-black/30 dark:focus-visible:ring-white/30'
  );

  return (
    <form
      className="flex w-full max-w-[280px] flex-col space-y-4"
      onSubmit={(e) => e.preventDefault()} // TODO: Kratos submit
    >
      <div className="flex w-full flex-col gap-1.5">
        <Label htmlFor="email" className={glassLabelStyle}>
          Email
        </Label>
        <Input
          type="email"
          id="email"
          placeholder="user@example.com"
          className={glassInputStyle}
          autoComplete="email"
        />
      </div>

      <div className="flex w-full flex-col gap-1.5">
        <Label htmlFor="password" className={glassLabelStyle}>
          Password
        </Label>
        <Input
          type="password"
          id="password"
          placeholder="••••••••"
          className={glassInputStyle}
          autoComplete="current-password"
        />
      </div>

      <div className="flex items-center space-x-2">
        <Checkbox id="remember" className={glassCheckboxStyle} />
        <Label
          htmlFor="remember"
          className={cn(glassLabelStyle, 'cursor-pointer')}
        >
          Remember me
        </Label>
      </div>

      <Button type="submit" variant="ghost" className={glassButtonStyle}>
        Sign In
      </Button>
    </form>
  );
};
