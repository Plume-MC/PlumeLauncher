import { Button as ButtonPrimitive } from "@base-ui/react/button"
import { motion } from "motion/react"
import { cva, type VariantProps } from "class-variance-authority"

import { cn } from "@/lib/utils"

const buttonVariants = cva(
  "group/button relative inline-flex shrink-0 items-center justify-center overflow-hidden rounded-lg border border-transparent bg-clip-padding text-sm font-medium whitespace-nowrap transition-[box-shadow,border-color] outline-none select-none focus-visible:ring-[3px] focus-visible:ring-ring/60 active:not-aria-[haspopup]:translate-y-px disabled:pointer-events-none disabled:opacity-50 aria-invalid:border-destructive aria-invalid:ring-3 aria-invalid:ring-destructive/20 dark:aria-invalid:border-destructive/50 dark:aria-invalid:ring-destructive/40 [&_svg]:pointer-events-none [&_svg]:shrink-0 [&_svg:not([class*='size-'])]:size-4",
  {
    variants: {
      variant: {
        default:
          "bg-primary text-primary-foreground shadow-[inset_0_1px_0_0_rgba(255,255,255,0.12),0_1px_2px_rgba(0,0,0,0.08)] hover:bg-primary hover:shadow-[inset_0_1px_0_0_rgba(255,255,255,0.16),0_0_0_1px_rgba(99,57,215,0.18),0_4px_18px_rgba(99,57,215,0.18)] dark:hover:shadow-[inset_0_1px_0_0_rgba(255,255,255,0.14),0_0_0_1px_rgba(99,57,215,0.25),0_6px_22px_rgba(99,57,215,0.22)]",
        outline:
          "border-border bg-background shadow-xs hover:border-primary/40 hover:bg-muted hover:text-foreground hover:shadow-sm aria-expanded:border-primary/40 aria-expanded:bg-muted",
        secondary:
          "bg-secondary text-secondary-foreground shadow-xs hover:bg-[color-mix(in_oklch,var(--secondary),var(--foreground)_6%)] hover:shadow-sm aria-expanded:bg-secondary aria-expanded:text-secondary-foreground",
        ghost:
          "hover:bg-muted hover:text-foreground aria-expanded:bg-muted aria-expanded:text-foreground dark:hover:bg-muted/50",
        destructive:
          "bg-destructive/10 text-destructive hover:bg-destructive/15 hover:shadow-[0_0_0_1px_rgba(239,68,68,0.12),0_4px_16px_rgba(239,68,68,0.1)] focus-visible:border-destructive/40 focus-visible:ring-destructive/20 dark:bg-destructive/20 dark:hover:bg-destructive/25 dark:focus-visible:ring-destructive/40",
        link: "text-primary underline-offset-4 hover:underline",
      },
      size: {
        default:
          "h-9 gap-1.5 px-3 in-data-[slot=button-group]:rounded-lg has-data-[icon=inline-end]:pr-2 has-data-[icon=inline-start]:pl-2",
        xs: "h-6 gap-1 rounded-md px-2 text-xs in-data-[slot=button-group]:rounded-md has-data-[icon=inline-end]:pr-1.5 has-data-[icon=inline-start]:pl-1.5 [&_svg:not([class*='size-'])]:size-3",
        sm: "h-8 gap-1 rounded-md px-2.5 in-data-[slot=button-group]:rounded-md has-data-[icon=inline-end]:pr-1.5 has-data-[icon=inline-start]:pl-1.5",
        lg: "h-10 gap-1.5 px-3.5 has-data-[icon=inline-end]:pr-2 has-data-[icon=inline-start]:pl-2",
        icon: "size-9",
        "icon-xs":
          "size-6 rounded-md in-data-[slot=button-group]:rounded-md [&_svg:not([class*='size-'])]:size-3",
        "icon-sm":
          "size-8 rounded-md in-data-[slot=button-group]:rounded-md",
        "icon-lg": "size-10",
      },
    },
    defaultVariants: {
      variant: "default",
      size: "default",
    },
  }
)

function Button({
  className,
  variant = "default",
  size = "default",
  ...props
}: ButtonPrimitive.Props & VariantProps<typeof buttonVariants>) {
  return (
    <ButtonPrimitive
      data-slot="button"
      className={cn(buttonVariants({ variant, size, className }))}
      {...props}
      render={(renderProps) => (
        <motion.span
          {...renderProps}
          whileHover={{ scale: 1.02 }}
          whileTap={{ scale: 0.98 }}
          transition={{ duration: 0.15, ease: [0.16, 1, 0.3, 1] }}
        />
      )}
    />
  )
}

export { Button, buttonVariants }
