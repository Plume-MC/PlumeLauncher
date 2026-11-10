import { useEffect, useRef, useState } from 'react';
import { IconUpload, IconX } from '@tabler/icons-react';
import { Dialog } from '@base-ui/react/dialog';
import { Button } from '@/components/ui/button';
import { toast } from '@/components/ui/toast';
import { SkinService } from '../../../bindings/plumelauncher/internal/services/index.js';
import type {
  Appearance,
  ClosetSkin,
} from '../../../bindings/plumelauncher/internal/services/models.js';
import type {
  ProfileCape,
  ProfileSkin,
  SkinVariant,
} from '../../../bindings/plumelauncher/internal/skins/models.js';
import { cn } from '@/lib/utils';

interface SkinCapeDialogProps {
  isOpen: boolean;
  onClose: () => void;
  accountUUID: string;
  accountName: string;
}

type ModelChoice = 'CLASSIC' | 'SLIM';

// SkinVariant enum values from the bindings.
const VARIANT_CLASSIC = 'classic' as SkinVariant;
const VARIANT_SLIM = 'slim' as SkinVariant;

// activeSkin picks the ACTIVE entry out of the profile's skin list.
function activeSkin(profile: Appearance['profile']): ProfileSkin | null {
  const skins = profile?.skins ?? [];
  return skins.find((skin) => skin.state === 'ACTIVE') ?? null;
}

function ownedCapes(profile: Appearance['profile']): ProfileCape[] {
  return profile?.capes ?? [];
}

// activeCapeID returns the equipped cape, or '' when none / hidden.
function activeCapeID(profile: Appearance['profile']): string {
  const capes = profile?.capes ?? [];
  return capes.find((cape) => cape.state === 'ACTIVE')?.id ?? '';
}

export function SkinCapeDialog({ isOpen, onClose, accountUUID, accountName }: SkinCapeDialogProps) {
  const [appearance, setAppearance] = useState<Appearance | null>(null);
  const [busy, setBusy] = useState('');
  const [error, setError] = useState('');
  const [model, setModel] = useState<ModelChoice>('CLASSIC');
  const [previewPNG, setPreviewPNG] = useState('');
  const fileInput = useRef<HTMLInputElement>(null);

  useEffect(() => {
    if (!isOpen) return;
    setBusy('load');
    setError('');
    void SkinService.GetProfileAppearance(accountUUID).then((result) => {
      setAppearance(result);
      const skin = activeSkin(result?.profile ?? null);
      if (skin) setModel(skin.variant === 'SLIM' ? 'SLIM' : 'CLASSIC');
    }).catch((err) => setError(err instanceof Error ? err.message : 'Unable to load your profile.')).finally(() => setBusy(''));
  }, [isOpen, accountUUID]);

  const apply = (result: Appearance) => {
    setAppearance(result);
    const skin = activeSkin(result.profile);
    if (skin) setModel(skin.variant === 'SLIM' ? 'SLIM' : 'CLASSIC');
  };

  const runWrite = async (label: string, action: () => Promise<Appearance | null>) => {
    setBusy(label);
    setError('');
    try {
      const result = await action();
      if (result) apply(result);
      toast.add({ type: 'success', title: 'Saved' });
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Something went wrong.');
    } finally {
      setBusy('');
    }
  };

  const onUpload = async (file: File) => {
    if (!file.name.toLowerCase().endsWith('.png')) {
      setError('Pick a PNG file.');
      return;
    }
    if (file.size > 1024 * 1024) {
      setError('That PNG is too large. Skin files are small; re-export the image.');
      return;
    }
    const buffer = new Uint8Array(await file.arrayBuffer());
    // Wails marshals []byte parameters as base64 strings.
    let binary = '';
    for (let i = 0; i < buffer.length; i += 0x8000) {
      binary += String.fromCharCode(...buffer.subarray(i, i + 0x8000));
    }
    const base64 = btoa(binary);
    // The uploaded texture doubles as the draft preview until Mojang responds.
    setPreviewPNG(`data:image/png;base64,${base64}`);
    await runWrite('upload', () => SkinService.UploadSkin(accountUUID, base64, model === 'SLIM' ? VARIANT_SLIM : VARIANT_CLASSIC));
  };

  const profile = appearance?.profile ?? null;
  const skin = activeSkin(profile);
  const capes = ownedCapes(profile);
  const equippedCape = activeCapeID(profile);
  const localSkins = appearance?.localSkins ?? [];
  const writeBusy = busy !== '' && busy !== 'load';

  return (
    <Dialog.Root open={isOpen} onOpenChange={(open) => { if (!open) onClose(); }}>
      <Dialog.Portal>
        <Dialog.Backdrop className="fixed inset-0 z-40 bg-black/50 backdrop-blur-sm transition-[opacity] duration-200 data-ending-style:opacity-0 data-starting-style:opacity-0" />
        <Dialog.Popup className="fixed left-1/2 top-1/2 z-50 flex max-h-[min(92vh,760px)] w-[min(880px,calc(100vw-1.5rem))] -translate-x-1/2 -translate-y-1/2 flex-col overflow-hidden rounded-xl bg-background shadow-xl outline-none transition-[opacity,transform] duration-200 ease-[cubic-bezier(0.16,1,0.3,1)] data-ending-style:scale-95 data-ending-style:opacity-0 data-starting-style:scale-95 data-starting-style:opacity-0">
          <div className="flex items-start justify-between border-b border-border px-5 py-4">
            <div>
              <Dialog.Title className="text-sm font-semibold">Skin &amp; Cape</Dialog.Title>
              <p className="text-xs text-muted-foreground">{accountName}</p>
            </div>
            <Button variant="ghost" size="icon-xs" onClick={onClose} aria-label="Close skin and cape editor"><IconX className="size-4" /></Button>
          </div>

          <div className="grid min-h-0 flex-1 gap-4 overflow-auto p-5 md:grid-cols-2">
            {/* Current skin */}
            <section className="rounded-lg border border-border bg-card/40 p-4">
              <h3 className="mb-2 text-xs font-medium uppercase tracking-wide text-muted-foreground">Current skin</h3>
              <div className="flex items-center justify-center rounded-lg bg-black/25 p-4">
                {previewPNG || skin?.url ? (
                  <img
                    src={previewPNG || skin?.url}
                    alt="Current skin"
                    className="h-64 [image-rendering:pixelated]"
                  />
                ) : (
                  <p className="text-xs text-muted-foreground">No custom skin yet.</p>
                )}
              </div>
              <p className="mt-2 text-[11px] text-muted-foreground">
                {model === 'SLIM' ? 'Slim (Alex) model' : 'Classic (Steve) model'}
              </p>
              {skin ? (
                <Button
                  variant="secondary"
                  size="sm"
                  className="mt-2 w-full"
                  disabled={writeBusy}
                  onClick={() => void runWrite('reset', () => SkinService.ResetSkin(accountUUID))}
                >
                  {busy === 'reset' ? 'Resetting...' : 'Reset to default skin'}
                </Button>
              ) : null}
            </section>

            {/* Model */}
            <section className="rounded-lg border border-border bg-card/40 p-4">
              <h3 className="mb-2 text-xs font-medium uppercase tracking-wide text-muted-foreground">Select your model</h3>
              <div className="grid grid-cols-2 gap-2">
                {(['CLASSIC', 'SLIM'] as ModelChoice[]).map((choice) => (
                  <button
                    key={choice}
                    type="button"
                    onClick={() => setModel(choice)}
                    className={cn(
                      'flex flex-col items-center gap-2 rounded-lg border p-3 text-xs transition-colors',
                      model === choice
                        ? 'border-primary bg-primary/10 text-foreground'
                        : 'border-border text-muted-foreground hover:bg-muted/40'
                    )}
                    aria-pressed={model === choice}
                  >
                    <span className="text-sm font-medium">{choice === 'CLASSIC' ? 'Classic' : 'Slim'}</span>
                    <span className="text-[11px] text-muted-foreground">{choice === 'CLASSIC' ? 'Wide arms' : 'Slim arms'}</span>
                  </button>
                ))}
              </div>
              <p className="mt-2 text-[11px] text-muted-foreground">Applied with your next skin upload.</p>
            </section>

            {/* Upload */}
            <section className="rounded-lg border border-border bg-card/40 p-4">
              <h3 className="mb-2 text-xs font-medium uppercase tracking-wide text-muted-foreground">Upload a custom skin</h3>
              <input
                ref={fileInput}
                type="file"
                accept="image/png"
                className="hidden"
                onChange={(event) => {
                  const file = event.target.files?.[0];
                  event.target.value = '';
                  if (file) void onUpload(file);
                }}
              />
              <Button
                size="sm"
                className="w-full"
                disabled={writeBusy}
                onClick={() => fileInput.current?.click()}
              >
                <IconUpload className="size-4" /> {busy === 'upload' ? 'Uploading...' : 'Select File'}
              </Button>
              <p className="mt-2 text-[11px] text-muted-foreground">
                PNG, 64×64 (or legacy 64×32). Applies to Minecraft: Java Edition.
              </p>
            </section>

            {/* Capes */}
            <section className="rounded-lg border border-border bg-card/40 p-4">
              <h3 className="mb-2 text-xs font-medium uppercase tracking-wide text-muted-foreground">Choose your cape</h3>
              {capes.length === 0 ? (
                <p className="text-[11px] text-muted-foreground">No capes owned yet.</p>
              ) : (
                <div className="grid grid-cols-4 gap-2">
                  <button
                    type="button"
                    onClick={() => void runWrite('cape', () => SkinService.HideCape(accountUUID))}
                    className={cn(
                      'flex h-20 flex-col items-center justify-center rounded-lg border text-[11px] transition-colors',
                      equippedCape === ''
                        ? 'border-primary bg-primary/10 text-foreground'
                        : 'border-border text-muted-foreground hover:bg-muted/40'
                    )}
                    aria-pressed={equippedCape === ''}
                  >
                    No Cape
                  </button>
                  {capes.map((cape) => (
                    <CapeTile
                      key={cape.id}
                      cape={cape}
                      selected={equippedCape === cape.id}
                      busy={writeBusy}
                      onSelect={() => void runWrite('cape', () => SkinService.EquipCape(accountUUID, cape.id))}
                    />
                  ))}
                </div>
              )}
            </section>

            {/* Local closet */}
            <section className="rounded-lg border border-border bg-card/40 p-4 md:col-span-2">
              <h3 className="mb-2 text-xs font-medium uppercase tracking-wide text-muted-foreground">Saved skins</h3>
              {localSkins.length === 0 ? (
                <p className="text-[11px] text-muted-foreground">Skins you upload are saved here for reuse.</p>
              ) : (
                <div className="grid grid-cols-[repeat(auto-fill,minmax(96px,1fr))] gap-2">
                  {localSkins.map((entry) => (
                    <LocalSkinTile
                      key={entry.textureKey}
                      entry={entry}
                      busy={writeBusy}
                      dataRoot={accountUUID}
                      onApply={() => {
                        void (async () => {
                          try {
                            const pngBase64 = await SkinService.LoadLocalSkinPNG(entry.textureKey);
                            await runWrite('upload', () => SkinService.UploadSkin(
                              accountUUID,
                              pngBase64,
                              entry.variant === 'slim' ? VARIANT_SLIM : VARIANT_CLASSIC,
                            ));
                          } catch (err) {
                            setError(err instanceof Error ? err.message : 'Unable to load that skin.');
                          }
                        })();
                      }}
                      onDelete={() => {
                        void (async () => {
                          try {
                            await SkinService.DeleteLocalSkin(entry.textureKey);
                            const refreshed = await SkinService.ListLocalSkins();
                            setAppearance((current) => current ? { ...current, localSkins: refreshed } : current);
                          } catch (err) {
                            setError(err instanceof Error ? err.message : 'Unable to delete that skin.');
                          }
                        })();
                      }}
                    />
                  ))}
                </div>
              )}
            </section>

            {error ? <p role="alert" className="text-xs text-destructive md:col-span-2">{error}</p> : null}
          </div>
        </Dialog.Popup>
        </Dialog.Portal>
      </Dialog.Root>
  );
}

function CapeTile({ cape, selected, busy, onSelect }: {
  cape: ProfileCape;
  selected: boolean;
  busy: boolean;
  onSelect: () => void;
}) {
  return (
    <button
      type="button"
      onClick={onSelect}
      disabled={busy}
      className={cn(
        'flex h-20 flex-col items-center justify-center gap-1 rounded-lg border p-2 transition-colors',
        selected
          ? 'border-primary bg-primary/10 text-foreground'
          : 'border-border text-muted-foreground hover:bg-muted/40'
      )}
      aria-pressed={selected}
    >
      <img src={cape.url} alt="" aria-hidden className="h-10 w-8 object-contain [image-rendering:pixelated]" />
      <span className="truncate text-[11px]">{cape.alias || 'Cape'}</span>
    </button>
  );
}

function LocalSkinTile({ entry, busy, dataRoot, onApply, onDelete }: {
  entry: ClosetSkin;
  busy: boolean;
  dataRoot: string;
  onApply: () => void;
  onDelete: () => void;
}) {
  const [png, setPNG] = useState('');
  useEffect(() => {
    let cancelled = false;
    void SkinService.LoadLocalSkinPNG(entry.textureKey).then((base64) => {
      if (!cancelled) setPNG(`data:image/png;base64,${base64}`);
    }).catch(() => undefined);
    return () => { cancelled = true; };
  }, [entry.textureKey]);

  return (
    <div className="flex flex-col items-center gap-1 rounded-lg border border-border p-2">
      {png ? <img src={png} alt="" aria-hidden className="h-16 [image-rendering:pixelated]" /> : <div className="h-16 w-8 animate-pulse bg-muted" />}
      <span className="text-[10px] text-muted-foreground">{entry.variant === 'slim' ? 'Slim' : 'Classic'}</span>
      <div className="flex gap-1">
        <Button variant="secondary" size="sm" disabled={busy} onClick={onApply}>Apply</Button>
        <Button variant="ghost" size="sm" className="text-destructive" disabled={busy} onClick={onDelete}>Delete</Button>
      </div>
      <span className="hidden">{dataRoot}</span>
    </div>
  );
}