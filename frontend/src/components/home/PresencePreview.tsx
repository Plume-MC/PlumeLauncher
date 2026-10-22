import type { Instance, LauncherDefaults } from '../../../bindings/plumelauncher/internal/instances/models.js';
import { cn } from '@/lib/utils';
import plumeMark from '@/assets/plume-mark.webp';
import fabricMark from '@/assets/loaders/fabric.png';
import quiltMark from '@/assets/loaders/quilt.png';

interface PresencePreviewProps {
  instance?: Instance | null;
  settings: LauncherDefaults;
  className?: string;
}

// Mirrors loaderLabel in internal/presence/activity_builder.go.
function loaderLabel(loader: string): string {
  switch (loader) {
    case 'fabric':
      return 'Fabric';
    case 'quilt':
      return 'Quilt';
    case 'vanilla':
      return 'Vanilla';
    default:
      return '';
  }
}

// Mirrors loaderAssetKey: only Fabric and Quilt have uploaded art.
function loaderArt(loader: string): string | null {
  if (loader === 'fabric') return fabricMark;
  if (loader === 'quilt') return quiltMark;
  return null;
}

// Mirrors buildDetails: gated segments joined with an em dash, capped at 128.
function buildDetails(instance: Instance, settings: LauncherDefaults): string {
  const segments: string[] = [];
  if (settings.discordShowInstance) segments.push(instance.name);
  if (settings.discordShowVersion && instance.mcVersion) segments.push(`Minecraft ${instance.mcVersion}`);
  if (settings.discordShowLoader) {
    const label = loaderLabel(instance.loader);
    if (label) segments.push(label);
  }
  return segments.join(' — ').slice(0, 128);
}

export function PresencePreview({ instance, settings, className }: PresencePreviewProps) {
  const art = instance && settings.discordShowLoader ? loaderArt(instance.loader) : null;
  const details = instance ? buildDetails(instance, settings) : '';

  return (
    <div className={cn('rounded-lg border border-border bg-background/60 p-3', className)}>
      <p className="mb-2 text-[11px] font-medium uppercase tracking-wide text-muted-foreground">Preview</p>

      <div className="flex items-start gap-3">
        <div className="relative shrink-0">
          <img src={plumeMark} alt="" aria-hidden className="size-14 rounded-lg object-cover" />
          {art ? (
            <span className="absolute -bottom-1 -right-1 grid size-5 place-items-center rounded-full border border-border bg-background">
              <img src={art} alt="" aria-hidden className="size-3.5 object-contain" />
            </span>
          ) : null}
        </div>

        <div className="min-w-0 flex-1">
          <p className="truncate text-xs font-semibold text-foreground">Plume Launcher</p>
          {instance ? (
            <>
              {details ? <p className="truncate text-[11px] text-muted-foreground">{details}</p> : null}
              <p className="truncate text-[11px] text-muted-foreground">Playing with Plume</p>
              {settings.discordShowElapsed ? (
                <p className="mt-1 text-[11px] text-muted-foreground">⏱ elapsed time</p>
              ) : null}
            </>
          ) : (
            <p className="text-[11px] text-muted-foreground">Create an instance to preview activity.</p>
          )}
        </div>
      </div>

      {!settings.discordPresenceEnabled ? (
        <p className="mt-2 text-[11px] text-muted-foreground">Presence is off, so nothing is shown on Discord.</p>
      ) : null}
    </div>
  );
}
