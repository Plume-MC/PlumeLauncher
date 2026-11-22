import { useCallback, useEffect, useRef, useState } from 'react';
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
import { renderCapeSprite, renderSkinFigure } from '@/lib/skinRender';

// cacheBust appends a changing query to a http(s) URL so the browser
// refetches a texture Mojang just replaced instead of serving the old cached
// copy. Data URLs pass through untouched.
function cacheBust(url: string, nonce: number): string {
  if (!url || nonce === 0) return url;
  return `${url}${url.includes('?') ? '&' : '?'}plume=${nonce}`;
}

interface SkinCapeDialogProps {
  isOpen: boolean;
  onClose: () => void;
  accountUUID: string;
  accountName: string;
  /** Fired after every successful appearance write so parents can refresh their own skin displays. */
  onSkinChanged?: () => void;
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

export function SkinCapeDialog({ isOpen, onClose, accountUUID, accountName, onSkinChanged }: SkinCapeDialogProps) {
  const [appearance, setAppearance] = useState<Appearance | null>(null);
  const [busy, setBusy] = useState('');
  const [error, setError] = useState('');
  const [loadError, setLoadError] = useState('');
  const [model, setModel] = useState<ModelChoice>('CLASSIC');
  const [previewPNG, setPreviewPNG] = useState('');
  const [view, setView] = useState<'front' | 'back'>('front');
  const profileUrls = useRef(0);
  const fileInput = useRef<HTMLInputElement>(null);

  const loadProfile = useCallback(() => {
    setBusy('load');
    setLoadError('');
    void SkinService.GetProfileAppearance(accountUUID).then((result) => {
      setAppearance(result);
      setLoadError('');
      const skin = activeSkin(result?.profile ?? null);
      if (skin) setModel(skin.variant === 'SLIM' ? 'SLIM' : 'CLASSIC');
    }).catch((err) => {
      setLoadError(err instanceof Error ? err.message : 'Unable to load your Minecraft profile.');
    }).finally(() => setBusy(''));
  }, [accountUUID]);

  useEffect(() => {
    if (!isOpen) return;
    loadProfile();
    // Local closet entries survive profile failures; refresh them too.
    void SkinService.ListLocalSkins().then((skins) => {
      setAppearance((current) => current ? { ...current, localSkins: skins } : current);
    }).catch(() => undefined);
  }, [isOpen, accountUUID, loadProfile]);

  const apply = (result: Appearance) => {
    setAppearance(result);
    profileUrls.current += 1;
    setPreviewPNG('');
    const skin = activeSkin(result.profile);
    if (skin) setModel(skin.variant === 'SLIM' ? 'SLIM' : 'CLASSIC');
  };

  const runWrite = async (label: string, action: () => Promise<Appearance | null>) => {
    setBusy(label);
    setError('');
    try {
      const result = await action();
      if (result) apply(result);
      onSkinChanged?.();
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
  // Draft texture: the just-picked file wins, then the equipped skin. The
  // nonce makes Mojang URLs refetch after a write.
  const textureURL = previewPNG || (skin?.url ? cacheBust(skin.url, profileUrls.current) : '');

  return (
    <Dialog.Root open={isOpen} onOpenChange={(open) => { if (!open) onClose(); }}>
      <Dialog.Portal>
        <Dialog.Backdrop className="fixed inset-0 z-40 bg-black/50 backdrop-blur-sm transition-[opacity] duration-200 data-ending-style:opacity-0 data-starting-style:opacity-0" />
        <Dialog.Popup className="fixed left-1/2 top-1/2 z-50 flex max-h-[min(92vh,720px)] w-[min(860px,calc(100vw-1.5rem))] -translate-x-1/2 -translate-y-1/2 flex-col overflow-hidden rounded-xl bg-background shadow-xl outline-none transition-[opacity,transform] duration-200 ease-[cubic-bezier(0.16,1,0.3,1)] data-ending-style:scale-95 data-ending-style:opacity-0 data-starting-style:scale-95 data-starting-style:opacity-0">
          <div className="flex items-start justify-between border-b border-border px-5 py-4">
            <div>
              <Dialog.Title className="text-sm font-semibold">Skin &amp; Cape</Dialog.Title>
              <p className="text-xs text-muted-foreground">{accountName}</p>
            </div>
            <Button variant="ghost" size="icon-xs" onClick={onClose} aria-label="Close skin and cape editor"><IconX className="size-4" /></Button>
          </div>

          <div className="grid min-h-0 flex-1 overflow-auto md:grid-cols-[300px_1fr] md:overflow-hidden">
            {/* Hero: the character render, the one rich element in the modal */}
            <div className="relative flex flex-col border-b border-border bg-card/40 p-5 md:border-b-0 md:border-r">
              <div
                aria-hidden
                className="pointer-events-none absolute inset-x-6 top-8 h-56 rounded-full bg-primary/20 blur-3xl"
              />
              <div className="relative flex min-h-[340px] flex-1 items-center justify-center">
                {loadError ? (
                  <p className="max-w-[20ch] text-center text-xs text-muted-foreground">{loadError}</p>
                ) : busy === 'load' && !textureURL ? (
                  <div className="h-40 w-20 rounded bg-muted/40" aria-hidden />
                ) : textureURL ? (
                  <SkinFigure textureURL={textureURL} model={model === 'SLIM' ? 'slim' : 'classic'} view={view} />
                ) : (
                  <p className="max-w-[20ch] text-center text-xs text-muted-foreground">No custom skin yet. Upload one to get started.</p>
                )}
              </div>

              <div className="relative mt-4 flex flex-col gap-3">
                <div className="flex items-center justify-center rounded-lg border border-border bg-background/60 p-1" role="tablist" aria-label="Skin view">
                  {(['front', 'back'] as const).map((v) => (
                    <button
                      key={v}
                      type="button"
                      role="tab"
                      aria-selected={view === v}
                      onClick={() => setView(v)}
                      className={cn(
                        'flex-1 rounded-md px-3 py-1.5 text-xs font-medium capitalize transition-colors',
                        view === v ? 'bg-primary/15 text-foreground' : 'text-muted-foreground hover:text-foreground',
                      )}
                    >
                      {v}
                    </button>
                  ))}
                </div>

                <p className="text-center text-[11px] text-muted-foreground">
                  {model === 'SLIM' ? 'Slim (Alex) model' : 'Classic (Steve) model'}
                </p>

                {skin ? (
                  <Button
                    variant="secondary"
                    size="sm"
                    disabled={writeBusy}
                    onClick={() => void runWrite('reset', () => SkinService.ResetSkin(accountUUID))}
                  >
                    {busy === 'reset' ? 'Resetting...' : 'Reset to default skin'}
                  </Button>
                ) : null}
              </div>
            </div>

            {/* Controls: model, upload, capes, saved skins in one scroll column */}
            <div className="min-h-0 flex-1 space-y-5 overflow-auto p-5">
              <section>
                <h3 className="mb-2 text-xs font-medium text-muted-foreground">Model</h3>
                <div className="grid grid-cols-2 gap-2">
                  {(['CLASSIC', 'SLIM'] as ModelChoice[]).map((choice) => (
                    <button
                      key={choice}
                      type="button"
                      onClick={() => setModel(choice)}
                      className={cn(
                        'rounded-lg border px-3 py-2.5 text-xs transition-colors',
                        model === choice
                          ? 'border-primary bg-primary/10 text-foreground'
                          : 'border-border text-muted-foreground hover:bg-muted/40'
                      )}
                      aria-pressed={model === choice}
                    >
                      <span className="block font-medium">{choice === 'CLASSIC' ? 'Classic' : 'Slim'}</span>
                      <span className="text-[11px] text-muted-foreground">{choice === 'CLASSIC' ? 'Wide arms' : 'Slim arms'}</span>
                    </button>
                  ))}
                </div>
                <p className="mt-2 text-[11px] text-muted-foreground">Applied with your next skin upload.</p>
              </section>

              <section>
                <h3 className="mb-2 text-xs font-medium text-muted-foreground">Custom skin</h3>
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

              <section>
                <h3 className="mb-2 text-xs font-medium text-muted-foreground">Capes</h3>
                {capes.length === 0 ? (
                  <p className="text-[11px] text-muted-foreground">No capes owned yet.</p>
                ) : (
                  <div className="grid grid-cols-4 gap-2">
                    <button
                      type="button"
                      onClick={() => void runWrite('cape', () => SkinService.HideCape(accountUUID))}
                      className={cn(
                        'flex h-24 flex-col items-center justify-center gap-1 rounded-lg border text-[11px] transition-colors',
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
                        cape={{ ...cape, url: cacheBust(cape.url, profileUrls.current) }}
                        selected={equippedCape === cape.id}
                        busy={writeBusy}
                        onSelect={() => void runWrite('cape', () => SkinService.EquipCape(accountUUID, cape.id))}
                      />
                    ))}
                  </div>
                )}
              </section>

              <section>
                <h3 className="mb-2 text-xs font-medium text-muted-foreground">Saved skins</h3>
                {localSkins.length === 0 ? (
                  <p className="text-[11px] text-muted-foreground">Uploaded skins are saved here for reuse.</p>
                ) : (
                  <div className="space-y-2">
                    {localSkins.map((entry) => (
                      <LocalSkinRow
                        key={entry.textureKey}
                        entry={entry}
                        busy={writeBusy}
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

              {error ? <p role="alert" className="text-xs text-destructive">{error}</p> : null}
            </div>
          </div>
        </Dialog.Popup>
      </Dialog.Portal>
    </Dialog.Root>
  );
}

// SkinFigure renders one composed character view of the texture with a soft
// backdrop, so the hero panel always shows the player, never the raw atlas.
function SkinFigure({ textureURL, model, view }: { textureURL: string; model: 'classic' | 'slim'; view: 'front' | 'back' }) {
  const [rendered, setRendered] = useState('');

  useEffect(() => {
    let cancelled = false;
    void renderSkinFigure(textureURL, view, model, 8)
      .then((dataURL) => {
        if (!cancelled) setRendered(dataURL);
      })
      .catch(() => {
        if (!cancelled) setRendered('');
      });
    return () => {
      cancelled = true;
    };
  }, [textureURL, view, model]);

  if (!rendered) {
    return <div className="h-48 w-24 rounded bg-muted/40" aria-hidden />;
  }
  return <img src={rendered} alt={`Skin ${view} view`} className="[image-rendering:pixelated]" />;
}

// CapeTile renders the 10x16 cape sprite cropped from the texture atlas and
// scaled up, instead of the whole raw atlas.
function CapeTile({ cape, selected, busy, onSelect }: {
  cape: ProfileCape;
  selected: boolean;
  busy: boolean;
  onSelect: () => void;
}) {
  const [sprite, setSprite] = useState('');

  useEffect(() => {
    let cancelled = false;
    void renderCapeSprite(cape.url, 6)
      .then((dataURL) => {
        if (!cancelled) setSprite(dataURL);
      })
      .catch(() => {
        if (!cancelled) setSprite('');
      });
    return () => {
      cancelled = true;
    };
  }, [cape.url]);

  return (
    <button
      type="button"
      onClick={onSelect}
      disabled={busy}
      className={cn(
        'flex h-24 flex-col items-center justify-center gap-1 rounded-lg border p-2 transition-colors',
        selected
          ? 'border-primary bg-primary/10 text-foreground'
          : 'border-border text-muted-foreground hover:bg-muted/40'
      )}
      aria-pressed={selected}
    >
      {sprite ? (
        <span className="grid h-14 w-11 place-items-center rounded bg-black/25 p-1">
          <img src={sprite} alt="" aria-hidden className="max-h-12 [image-rendering:pixelated]" />
        </span>
      ) : (
        <span className="h-14 w-11 rounded bg-muted/40" aria-hidden />
      )}
      <span className="w-full truncate text-center text-[11px]">{cape.alias || 'Cape'}</span>
    </button>
  );
}

// LocalSkinRow is one compact closet entry: figure, variant label, and its
// two actions in a single row.
function LocalSkinRow({ entry, busy, onApply, onDelete }: {
  entry: ClosetSkin;
  busy: boolean;
  onApply: () => void;
  onDelete: () => void;
}) {
  const [figure, setFigure] = useState('');
  const [failed, setFailed] = useState(false);

  useEffect(() => {
    let cancelled = false;
    setFailed(false);
    void SkinService.LoadLocalSkinPNG(entry.textureKey).then((base64) => {
      if (cancelled) return;
      // The closet stores normalized 64x64 PNGs as base64.
      return renderSkinFigure(`data:image/png;base64,${base64}`, 'front', entry.variant === 'slim' ? 'slim' : 'classic', 3);
    }).then((rendered) => {
      if (cancelled) return;
      if (rendered) setFigure(rendered);
      else setFailed(true);
    }).catch(() => {
      if (!cancelled) setFailed(true);
    });
    return () => { cancelled = true; };
  }, [entry.textureKey, entry.variant]);

  return (
    <div className="flex items-center gap-3 rounded-lg border border-border bg-card/40 p-2.5">
      <div className="grid h-16 w-8 shrink-0 place-items-center">
        {figure ? (
          <img src={figure} alt="" aria-hidden className="[image-rendering:pixelated]" />
        ) : failed ? (
          <span className="text-[10px] text-muted-foreground">N/A</span>
        ) : (
          <div className="h-14 w-7 rounded bg-muted" aria-hidden />
        )}
      </div>
      <div className="min-w-0 flex-1">
        <p className="truncate text-xs font-medium text-foreground">{entry.variant === 'slim' ? 'Slim' : 'Classic'}</p>
        <p className="font-mono text-[10px] text-muted-foreground">{entry.textureKey.slice(0, 10)}</p>
      </div>
      <div className="flex shrink-0 gap-1.5">
        <Button variant="secondary" size="sm" disabled={busy} onClick={onApply}>Apply</Button>
        <Button variant="ghost" size="sm" className="text-destructive" disabled={busy} onClick={onDelete}>Delete</Button>
      </div>
    </div>
  );
}