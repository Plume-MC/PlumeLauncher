// 2D skin and cape rendering from raw texture URLs. Everything runs on a
// canvas element: no WebGL, no image libraries. Keeping these as utilities
// rather than components makes both preview types share one implementation.

// drawRegion renders the part of source defined by sx, sy, sw, sh into a
// scaled canvas and resolves with its data URL. Pixelated scaling is applied
// via imageSmoothingEnabled=false so texels stay sharp.
export function drawRegion(
  source: HTMLImageElement | HTMLCanvasElement,
  sx: number,
  sy: number,
  sw: number,
  sh: number,
  scale: number,
): string {
  const canvas = document.createElement('canvas');
  canvas.width = sw * scale;
  canvas.height = sh * scale;
  const ctx = canvas.getContext('2d');
  if (!ctx) return '';
  ctx.imageSmoothingEnabled = false;
  ctx.drawImage(source, sx, sy, sw, sh, 0, 0, canvas.width, canvas.height);
  return canvas.toDataURL('image/png');
}

// loadImage resolves once the texture at src is drawable.
export function loadImage(src: string): Promise<HTMLImageElement> {
  return new Promise((resolve, reject) => {
    const image = new Image();
    image.crossOrigin = 'anonymous';
    image.onload = () => resolve(image);
    image.onerror = () => reject(new Error('texture failed to load'));
    image.src = src;
  });
}

// Atlas boxes for the classic model: every limb face of the modern 64x64
// layout. Per minecraft.wiki/w/Skin, the classic figure is 16px wide with
// 4px arms, and a slim texture is the same atlas with the arms losing 25%
// width: only three of each arm's four columns are painted, the rightmost
// column is empty and ignored.
const SKIN_BOXES = {
  head: { front: { sx: 8, sy: 8, sw: 8, sh: 8 }, back: { sx: 24, sy: 8, sw: 8, sh: 8 } },
  body: { front: { sx: 20, sy: 20, sw: 8, sh: 12 }, back: { sx: 32, sy: 20, sw: 8, sh: 12 } },
  rightArm: { front: { sx: 44, sy: 20, sw: 4, sh: 12 }, back: { sx: 52, sy: 20, sw: 4, sh: 12 } },
  leftArm: { front: { sx: 36, sy: 52, sw: 4, sh: 12 }, back: { sx: 44, sy: 52, sw: 4, sh: 12 } },
  rightLeg: { front: { sx: 4, sy: 20, sw: 4, sh: 12 }, back: { sx: 12, sy: 20, sw: 4, sh: 12 } },
  leftLeg: { front: { sx: 20, sy: 52, sw: 4, sh: 12 }, back: { sx: 28, sy: 52, sw: 4, sh: 12 } },
} as const;

// Figure footprint per model, per minecraft.wiki: classic is 16px wide with
// 4px arms; slim is 14px wide with 3px arms. Slim centers on the same axis,
// so the body sits one pixel left of where it does on classic.
type ModelGeometry = {
  figureW: number;
  armWidth: number;
  bodyX: number;
  headX: number;
};

const MODEL_GEOMETRY: Record<'classic' | 'slim', ModelGeometry> = {
  classic: { figureW: 16, armWidth: 4, bodyX: 4, headX: 4 },
  slim: { figureW: 14, armWidth: 3, bodyX: 3, headX: 3 },
};

type PartKey = keyof typeof SKIN_BOXES;

type Placement = { sx: number; sy: number; sw: number; sh: number; dx: number; dy: number; dw: number };

// buildPlacements maps every limb from the atlas onto the figure for one
// model and view. Arms always hug the body: no gaps, no floating limbs.
function buildPlacements(model: ModelGeometry, view: 'front' | 'back'): Placement[] {
  const isFront = view === 'front';
  const figureW = model.figureW;
  const arm = model.armWidth;

  // Body/head sit centered: (figureW - 8) / 2 above the legs.
  const bodyX = Math.round((figureW - 8) / 2);
  const headX = bodyX;
  // Arms attach directly to the body's sides. Front view puts the right arm
  // on the viewer's left; back view flips it.
  const rightArmX = isFront ? bodyX - arm : bodyX + 8;
  const leftArmX = isFront ? bodyX + 8 : bodyX - arm;
  const legRightX = bodyX;
  const legLeftX = bodyX + 4;

  const entries: Array<[PartKey, Placement]> = [
    ['head', { sx: isFront ? 8 : 24, sy: 8, sw: 8, sh: 8, dx: headX, dy: 0, dw: 8 }],
    ['body', { sx: isFront ? 20 : 32, sy: 20, sw: 8, sh: 12, dx: bodyX, dy: 8, dw: 8 }],
    ['rightArm', {
      sx: isFront ? 44 : 52, sy: 20, sw: arm, sh: 12,
      dx: rightArmX, dy: 8, dw: arm,
    }],
    ['leftArm', {
      sx: isFront ? 36 : 44, sy: 52, sw: arm, sh: 12,
      dx: leftArmX, dy: 8, dw: arm,
    }],
    ['rightLeg', { sx: isFront ? 4 : 12, sy: 20, sw: 4, sh: 12, dx: legRightX, dy: 20, dw: 4 }],
    ['leftLeg', { sx: isFront ? 20 : 28, sy: 52, sw: 4, sh: 12, dx: legLeftX, dy: 20, dw: 4 }],
  ];

  // Order back-to-front per view so overlapping parts stack like the game.
  const order = isFront
    ? ['rightArm', 'rightLeg', 'body', 'leftLeg', 'head', 'leftArm']
    : ['leftArm', 'leftLeg', 'body', 'rightLeg', 'head', 'rightArm'];

  return order.map((key) => entries.find(([k]) => k === key)![1]);
}

// renderSkinFigure composes the visible parts of a skin atlas into a clean
// front- or back-facing character render, like the ely.by skin page.
//
// model decides both the figure footprint and which arm columns are sampled:
// slim samples three painted columns per arm face (the fourth is empty on
// slim textures) and renders a 14px-wide figure, classic samples all four and
// renders 16px. Arms always touch the body, matching the real proportions.
export async function renderSkinFigure(
  textureURL: string,
  view: 'front' | 'back',
  model: 'classic' | 'slim',
  scale = 6,
): Promise<string> {
  const image = await loadImage(textureURL);
  const geometry = MODEL_GEOMETRY[model];

  const canvas = document.createElement('canvas');
  canvas.width = geometry.figureW * scale;
  canvas.height = 32 * scale;
  const ctx = canvas.getContext('2d');
  if (!ctx) return '';
  ctx.imageSmoothingEnabled = false;

  for (const part of buildPlacements(geometry, view)) {
    ctx.drawImage(
      image,
      part.sx, part.sy, part.sw, part.sh,
      part.dx * scale, part.dy * scale, part.dw * scale, part.sh * scale,
    );
  }

  return canvas.toDataURL('image/png');
}

// The cape texture is a 10x16 sprite living at (1, 0) of a 64x32 atlas.
export const CAPE_REGION = { sx: 1, sy: 0, sw: 10, sh: 16 } as const;

// renderCapeSprite crops the 10x16 cape sprite and scales it up.
export async function renderCapeSprite(
  textureURL: string,
  scale = 6,
): Promise<string> {
  const image = await loadImage(textureURL);
  return drawRegion(image, CAPE_REGION.sx, CAPE_REGION.sy, CAPE_REGION.sw, CAPE_REGION.sh, scale);
}