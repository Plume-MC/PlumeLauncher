import { useDeferredValue, useEffect, useRef, useState } from "react";
import { ArrowDown, Download, Terminal, Trash2, X } from "lucide-react";
import { motion } from "motion/react";
import { Button } from "@/components/ui/button";
import { useMotionPreference } from "@/components/motion";
import { HomeService } from "../../../bindings/plumelauncher/internal/services/index.js";
import type { DownloadProgressEvent } from "../../../bindings/plumelauncher/internal/services/models.js";

export interface ActivityLogLine {
  level: string;
  message: string;
}

interface ActivityPanelProps {
  isOpen: boolean;
  onClose: () => void;
  onClearConsole: () => void;
  consoleLines: ActivityLogLine[];
  download: DownloadProgressEvent | null;
}

const activeDownloadStates = new Set(["downloading", "repairing"]);

function formatBytes(bytes: number) {
  return `${Math.round(bytes / 1024 / 1024)} MB`;
}

function formatETA(seconds: number) {
  if (seconds < 60) return `${Math.ceil(seconds)}s left`;
  return `${Math.ceil(seconds / 60)}m left`;
}

function logTone(level: string) {
  if (level === "error") return "text-destructive";
  if (level === "warn" || level === "warning") return "text-amber-400";
  return "text-muted-foreground";
}

export function ActivityPanel({ isOpen, onClose, onClearConsole, consoleLines, download }: ActivityPanelProps) {
  const { reduced } = useMotionPreference();
  const [cancelling, setCancelling] = useState(false);
  const [following, setFollowing] = useState(true);
  const deferredConsoleLines = useDeferredValue(consoleLines);
  const consoleRef = useRef<HTMLDivElement>(null);
  const showOperation = !!download && (activeDownloadStates.has(download.status) || download.status === "failed");

  useEffect(() => {
    if (!isOpen || !following || !consoleRef.current) return;
    consoleRef.current.scrollTop = consoleRef.current.scrollHeight;
  }, [deferredConsoleLines.length, following, isOpen]);

  useEffect(() => {
    if (!isOpen) return;
    const closeOnEscape = (event: globalThis.KeyboardEvent) => {
      if (event.key === "Escape") onClose();
    };
    window.addEventListener("keydown", closeOnEscape);
    return () => window.removeEventListener("keydown", closeOnEscape);
  }, [isOpen, onClose]);

  if (!isOpen) return null;

  const cancelDownload = async () => {
    if (!download?.instanceId) return;
    setCancelling(true);
    try {
      await HomeService.CancelInstance(download.instanceId);
    } finally {
      setCancelling(false);
    }
  };

  const maxProgress = download?.totalBytes || download?.totalFiles || 1;
  const currentProgress = download?.totalBytes ? download.byteProgress : (download?.fileProgress ?? 0);

  return (
    <motion.aside
      className="fixed bottom-4 right-4 z-50 flex h-[min(440px,calc(100dvh-5rem))] w-[min(500px,calc(100vw-2rem))] flex-col overflow-hidden rounded-xl border border-border bg-background shadow-xl"
      data-slot="activity-panel"
      initial={reduced ? false : { opacity: 0, y: 12, scale: 0.98 }}
      animate={{ opacity: 1, y: 0, scale: 1 }}
      transition={{ duration: reduced ? 0.01 : 0.2, ease: [0.16, 1, 0.3, 1] }}
    >
      <header className="flex items-center justify-between border-b border-border px-4 py-3">
        <div className="flex items-center gap-2">
          <Terminal className="size-4 text-muted-foreground" />
          <h2 className="text-sm font-semibold">Activity</h2>
        </div>
        <Button variant="ghost" size="icon-xs" onClick={onClose} aria-label="Close activity panel">
          <X className="size-3.5" />
        </Button>
      </header>

      {showOperation ? (
        <section className="space-y-3 border-b border-border px-4 py-3" aria-live="polite">
          <div className="flex items-center justify-between gap-3 text-xs">
            <span className="flex items-center gap-2 font-medium">
              <Download className="size-3.5" />
              {download.status === "repairing" ? "Repairing files" : "Installing files"}
            </span>
            {activeDownloadStates.has(download.status) ? (
              <Button variant="destructive" size="sm" disabled={cancelling} onClick={() => void cancelDownload()}>
                {cancelling ? "Cancelling..." : "Cancel"}
              </Button>
            ) : null}
          </div>
          <progress className="h-1.5 w-full accent-primary" value={currentProgress} max={maxProgress} aria-label="Download progress" />
          <div className="flex flex-wrap gap-x-2 text-xs text-muted-foreground">
            <span>
              {download.fileProgress}/{download.totalFiles} files
            </span>
            {download.totalBytes > 0 ? (
              <span>
                {formatBytes(download.byteProgress)} / {formatBytes(download.totalBytes)}
              </span>
            ) : null}
            {download.speed > 0 ? <span>{Math.round(download.speed / 1024)} KB/s</span> : null}
            {download.eta > 0 ? <span>{formatETA(download.eta)}</span> : null}
          </div>
          {download.error ? (
            <p role="alert" className="text-xs text-destructive">
              {download.error}
            </p>
          ) : null}
        </section>
      ) : null}

      <section className="flex min-h-0 flex-1 flex-col">
        <div className="flex items-center justify-between border-b border-border px-4 py-2">
          <span className="text-xs font-medium text-muted-foreground">Console</span>
          <div className="flex items-center gap-1">
            {!following ? (
              <Button variant="ghost" size="icon-xs" onClick={() => setFollowing(true)} aria-label="Follow latest output" title="Follow latest output">
                <ArrowDown className="size-3.5" />
              </Button>
            ) : null}
            <Button variant="ghost" size="icon-xs" onClick={onClearConsole} disabled={!consoleLines.length} aria-label="Clear console">
              <Trash2 className="size-3.5" />
            </Button>
          </div>
        </div>
        <div
          ref={consoleRef}
          className="min-h-0 flex-1 overflow-y-auto px-4 py-3 font-mono text-[11px] leading-5"
          onScroll={() => {
            const element = consoleRef.current;
            if (element) setFollowing(element.scrollHeight - element.scrollTop - element.clientHeight < 24);
          }}
        >
          {deferredConsoleLines.length ? (
            deferredConsoleLines.map((line, index) => (
              <p key={`${index}:${line.message}`} className={`break-words [content-visibility:auto] ${logTone(line.level)}`}>
                {line.message}
              </p>
            ))
          ) : (
            <p className="text-muted-foreground/60">No console output.</p>
          )}
        </div>
      </section>
    </motion.aside>
  );
}
