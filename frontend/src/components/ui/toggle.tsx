import * as TogglePrimitive from '@radix-ui/react-toggle';
import * as React from 'react';

import { cn } from '@/lib/utils';
import { toggleVariants, type ToggleVariants } from './toggle-variants';

function Toggle({
  className,
  variant,
  size,
  ...props
}: React.ComponentProps<typeof TogglePrimitive.Root> & ToggleVariants) {
  return (
    <TogglePrimitive.Root
      data-slot="toggle"
      className={cn(toggleVariants({ variant, size, className }))}
      {...props}
    />
  );
}

export { Toggle };
