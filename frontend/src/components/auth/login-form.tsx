import { cn } from '@/lib/utils';
import { Input } from '@/components/ui/input';
import { Button } from '@/components/ui/button';
import { Checkbox } from '@/components/ui/checkbox';
import { Label } from '@/components/ui/label';
import {
  glassInput,
  glassLabel,
  glassCheckbox,
  primaryCTA,
} from '@/lib/glass-styles';

export const EmailLoginForm = () => {
  return (
    <form
      className="flex w-full max-w-[280px] flex-col space-y-4"
      onSubmit={(e) => e.preventDefault()} // TODO: Kratos submit
    >
      <div className="flex w-full flex-col gap-1.5">
        <Label htmlFor="email" className={glassLabel}>
          Email
        </Label>
        <Input
          type="email"
          id="email"
          placeholder="user@example.com"
          className={glassInput}
          autoComplete="email"
        />
      </div>

      <div className="flex w-full flex-col gap-1.5">
        <Label htmlFor="password" className={glassLabel}>
          Password
        </Label>
        <Input
          type="password"
          id="password"
          placeholder="••••••••"
          className={glassInput}
          autoComplete="current-password"
        />
      </div>

      <div className="flex items-center space-x-2">
        <Checkbox id="remember" className={glassCheckbox} />
        <Label htmlFor="remember" className={cn(glassLabel, 'cursor-pointer')}>
          Remember me
        </Label>
      </div>

      <Button type="submit" variant="ghost" className={primaryCTA}>
        Sign In
      </Button>
    </form>
  );
};
