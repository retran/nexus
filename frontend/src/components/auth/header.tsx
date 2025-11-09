import { cn } from '@/lib/utils';
import {
  glassText,
  glassHeader,
  glassHeaderIcon,
  glassHeaderTitle,
} from '@/lib/glass-styles';
import { House } from 'lucide-react';

interface HeaderProps {
  className?: string;
}

export const Header = ({ className }: HeaderProps) => {
  return (
    <div className={cn(glassHeader, className)}>
      <div className={cn('flex items-center justify-center')}>
        <House className={cn(glassHeaderIcon, glassText)} />
      </div>

      <h1 className={cn(glassHeaderTitle, glassText)}>Nexus</h1>
    </div>
  );
};
