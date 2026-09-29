const JAVA_RANGES: Record<number, string> = {
  25: 'For Minecraft 26 and newer',
  21: 'For Minecraft 1.20.5 and newer',
  17: 'For Minecraft 1.17 to 1.20.4',
  8: 'For Minecraft 1.16 and older',
};

export function javaRangeLabel(major: number): string {
  return JAVA_RANGES[major] ?? `Java ${major}`;
}
