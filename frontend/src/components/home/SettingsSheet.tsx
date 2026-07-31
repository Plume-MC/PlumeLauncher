import { useState } from 'react';
import { X } from 'lucide-react';
import { Button } from '@/components/ui/button';

type SettingsTab = 'general' | 'java' | 'data' | 'about';

interface SettingsSheetProps {
  isOpen: boolean;
  onClose: () => void;
}

export function SettingsSheet({ isOpen, onClose }: SettingsSheetProps) {
  const [activeTab, setActiveTab] = useState<SettingsTab>('general');

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      {/* Backdrop */}
      <div className="absolute inset-0 bg-black/40" onClick={onClose} />

      {/* Sheet */}
      <div className="relative flex h-[520px] w-[560px] max-w-[90vw] flex-col rounded-lg border border-border bg-background shadow-xl">
        {/* Header */}
        <div className="flex items-center justify-between border-b border-border px-5 py-4">
          <h2 className="text-sm font-semibold">Settings</h2>
          <Button variant="ghost" size="icon-xs" onClick={onClose} aria-label="Close settings">
            <X className="size-4" />
          </Button>
        </div>

        {/* Tabs */}
        <div className="flex border-b border-border px-5">
          {(['general', 'java', 'data', 'about'] as const).map((tab) => (
            <button
              key={tab}
              className={`px-3 py-2.5 text-xs font-medium capitalize transition-colors ${
                activeTab === tab
                  ? 'border-b-2 border-primary text-foreground'
                  : 'text-muted-foreground hover:text-foreground'
              }`}
              onClick={() => setActiveTab(tab)}
            >
              {tab}
            </button>
          ))}
        </div>

        {/* Content */}
        <div className="flex-1 overflow-auto px-5 py-4 text-sm text-muted-foreground">
          {activeTab === 'general' && (
            <div className="space-y-4">
              <p className="text-xs">General settings (theme, window mode, resolution, RAM defaults).</p>
            </div>
          )}
          {activeTab === 'java' && (
            <div className="space-y-4">
              <p className="text-xs">Java scan list, default path, JVM args, wrapper command.</p>
            </div>
          )}
          {activeTab === 'data' && (
            <div className="space-y-4">
              <p className="text-xs">Data root location, open logs folder.</p>
            </div>
          )}
          {activeTab === 'about' && (
            <div className="space-y-4">
              <p className="text-xs">Plume Launcher v1.0.0 · Windows + Linux</p>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
