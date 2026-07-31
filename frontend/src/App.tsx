import './index.css'
import { useState, useEffect, useRef } from 'react'
import { Events, WML } from "@wailsio/runtime";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Card, CardContent, CardDescription, CardFooter, CardHeader, CardTitle } from "@/components/ui/card";
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";
import { Separator } from "@/components/ui/separator";
import { toast } from "@/components/ui/toast";
import { ArrowRight, Clock, ExternalLink } from "lucide-react";

const wailsVersion = "v3.0.0-alpha2.117";

function App() {
  const [name, setName] = useState('');
  const [time, setTime] = useState('Listening for Time event...');
  const titleNameRef = useRef<HTMLSpanElement | null>(null);

  const swapTitleName = (name: string) => {
    const el = titleNameRef.current;
    if (!el) return;
    const current = el.querySelector('.title-name-text:not(.is-outgoing)');
    if (!current || current.textContent === name) return;
    const incoming = document.createElement('span');
    incoming.className = 'title-name-text is-entering';
    incoming.textContent = name;
    current.classList.add('is-outgoing');
    el.appendChild(incoming);
    void incoming.offsetWidth;
    incoming.classList.remove('is-entering');
    current.classList.add('is-leaving');
    current.addEventListener('transitionend', () => current.remove(), { once: true });
  };

  const doGreet = () => {
    const n = name || 'anonymous';
    swapTitleName(n);
  };

  useEffect(() => {
    document.documentElement.classList.add('dark');
    Events.On('time', (timeValue: any) => {
      const full = timeValue.data;
      const compact = (full.match(/\d{1,2}:\d{2}:\d{2}/) || [full])[0];
      setTime(window.matchMedia('(max-width: 640px)').matches ? compact : full);
    });
    WML.Reload();
  }, []);

  return (
    <div className="relative flex min-h-[100dvh] flex-col overflow-hidden bg-zinc-950 text-white">
      <div className="pointer-events-none fixed inset-0 z-0 bg-cover bg-center bg-no-repeat bg-desktop" />
      <div className="pointer-events-none fixed inset-0 z-0 bg-gradient-to-r from-zinc-950/95 via-zinc-950/70 to-zinc-950/25" />
      <div className="pointer-events-none fixed inset-0 z-0 bg-gradient-to-t from-zinc-950/80 via-transparent to-zinc-950/30" />

      <main className="relative z-10 flex flex-1 items-center px-5 py-12 sm:px-10 lg:px-16">
        <Card className="w-full max-w-xl border-white/15 bg-zinc-950/55 py-0 text-white shadow-2xl shadow-black/45 backdrop-blur-2xl [--wails-draggable:no-drag]">
          <CardHeader className="gap-5 border-b border-white/10 pb-6">
            <div className="flex items-center gap-3">
              <a href="https://v3.wails.io" target="_blank" rel="noopener" aria-label="Wails website">
                <Avatar className="size-12 rounded-lg">
                  <AvatarImage src="/wails.png" alt="Wails logo" />
                  <AvatarFallback>W</AvatarFallback>
                </Avatar>
              </a>
              <Separator orientation="vertical" className="h-8 bg-white/15" />
              <a href="https://reactjs.org" target="_blank" rel="noopener" aria-label="React">
                <Avatar className="size-10 rounded-lg">
                  <AvatarImage src="/react.svg" alt="React logo" />
                  <AvatarFallback>R</AvatarFallback>
                </Avatar>
              </a>
            </div>
            <div className="flex flex-col gap-3">
              <CardTitle className="text-4xl font-extrabold tracking-tight sm:text-5xl">
                <span className="bg-gradient-to-r from-rose-400 to-pink-500 bg-clip-text text-transparent">Wails +</span>{' '}
                <span className="relative inline-block" ref={titleNameRef}>
                  <span className="title-name-text inline-block transition-all duration-500 ease-out">React</span>
                </span>
              </CardTitle>
              <CardDescription className="max-w-sm text-base leading-relaxed text-white/65">
                Build beautiful cross-platform apps with Go and React.
              </CardDescription>
            </div>
          </CardHeader>

          <CardContent className="pt-6">
            <div className="flex flex-col gap-3 sm:flex-row">
              <Input
                value={name}
                onChange={(e) => setName(e.target.value)}
                placeholder="Your name"
                autoComplete="off"
                aria-label="input"
                className="h-11 bg-white/5 border-white/15 text-white placeholder:text-white/40 focus-visible:border-rose-400/70 focus-visible:ring-rose-400/25"
              />
              <Button
                onClick={doGreet}
                aria-label="greet-btn"
                size="lg"
                className="bg-rose-500 text-white hover:bg-rose-400"
              >
                Greet
                <ArrowRight data-icon="inline-end" />
              </Button>
            </div>
          </CardContent>

          <CardFooter className="border-t border-white/10 pt-4 text-xs text-white/45">
            Your greeting will be handled by the Go service.
          </CardFooter>
        </Card>
      </main>

      <Separator className="relative z-10 bg-white/10" />

      <footer className="relative z-10 grid grid-cols-[1fr_auto_1fr] items-center gap-4 bg-zinc-950/35 px-5 py-4 text-xs text-white/60 backdrop-blur-md sm:px-10 lg:px-16 [--wails-draggable:no-drag]">
        <span className="flex items-center gap-2 whitespace-nowrap">{wailsVersion}</span>
        <span className="flex items-center justify-self-center gap-2 whitespace-nowrap">
          <Clock className="size-4 opacity-80" />
          <span>{time}</span>
        </span>
        <a
          href="https://v3.wails.io"
          target="_blank"
          rel="noopener"
          className="flex items-center justify-self-end gap-1 text-white/60 no-underline transition-colors hover:text-white whitespace-nowrap"
          aria-label="Wails documentation"
        >
          Docs
          <ExternalLink className="size-4" />
        </a>
      </footer>

      <style>{`
        .title-name-text.is-outgoing { position: absolute; left: 0; top: 0; }
        .title-name-text.is-entering { opacity: 0; transform: translateY(0.3em); }
        .title-name-text.is-leaving { opacity: 0; transform: translateY(-0.3em); }
        body { --wails-draggable: drag; }
        input { -webkit-user-select: text; user-select: text; }
        .bg-desktop { background-image: url("/bg-desktop.jpg"); }
        @media (max-width: 640px) {
          .bg-desktop { background-image: url("/bg-mobile.jpg"); }
        }
      `}</style>
    </div>
  )
}

export default App
