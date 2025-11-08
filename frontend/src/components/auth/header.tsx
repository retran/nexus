import { cn } from '@/lib/utils';
import { House } from 'lucide-react';

interface HeaderProps {
  className?: string;
}

export const Header = ({ className }: HeaderProps) => {
  return (
    <div
      className={cn(
        'flex flex-col items-center justify-center space-y-3',
        className
      )}
    >
      <div className={cn('flex items-center justify-center')}>
        <House className="h-16 w-16 text-black/90 dark:text-white/90" />
      </div>

      <h1 className="text-2xl font-semibold text-black/90 dark:text-white/90">
        Nexus
      </h1>
    </div>
  );
};
