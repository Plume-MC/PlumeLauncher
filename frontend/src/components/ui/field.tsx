import { Field as FieldPrimitive } from '@base-ui/react/field';
import type { ComponentProps } from 'react';

import { cn } from '@/lib/utils';

const Field = FieldPrimitive.Root;

function FieldGroup({ className, ...props }: ComponentProps<'div'>) {
  return <div className={cn('flex flex-col gap-3', className)} {...props} />;
}

function FieldLabel({ className, ...props }: ComponentProps<typeof FieldPrimitive.Label>) {
  return <FieldPrimitive.Label className={cn('text-xs font-medium text-foreground/90', className)} {...props} />;
}

function FieldDescription({ className, ...props }: ComponentProps<typeof FieldPrimitive.Description>) {
  return <FieldPrimitive.Description className={cn('text-xs text-muted-foreground', className)} {...props} />;
}

function FieldError({ className, ...props }: ComponentProps<typeof FieldPrimitive.Error>) {
  return <FieldPrimitive.Error className={cn('text-xs text-destructive', className)} {...props} />;
}

export { Field, FieldDescription, FieldError, FieldGroup, FieldLabel };
