import React from 'react';
import { cn } from '@/lib/utils';
import { glassPanel } from '@/lib/glass-styles';

interface GlassPanelProps {
  children: React.ReactNode;
  className?: string;
}

export const GlassPanel: React.FC<GlassPanelProps> = ({
  children,
  className,
}) => {
  return <div className={cn(glassPanel, className)}>{children}</div>;
};
