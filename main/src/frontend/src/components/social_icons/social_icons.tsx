import type { JSX, SVGProps } from "react";

export const SOCIAL_PLATFORMS = ["instagram", "tiktok", "youtube", "x", "facebook", "strava"] as const;
export type SocialPlatform = (typeof SOCIAL_PLATFORMS)[number];

export const SOCIAL_NAMES: Record<SocialPlatform, string> = {
  instagram: "Instagram",
  tiktok: "TikTok",
  youtube: "YouTube",
  x: "X",
  facebook: "Facebook",
  strava: "Strava"
};

// Simple line glyphs in the style of the app's icons (the installed icon set
// has no brand marks).
const GLYPHS: Record<SocialPlatform, JSX.Element> = {
  instagram: (
    <>
      <rect x="3" y="3" width="18" height="18" rx="5" />
      <circle cx="12" cy="12" r="4" />
      <circle cx="17.5" cy="6.5" r="0.9" fill="currentColor" stroke="none" />
    </>
  ),
  tiktok: (
    <>
      <path d="M14 3v11.5a3.5 3.5 0 1 1-3.5-3.5" />
      <path d="M14 3c.6 2.6 2.4 4.3 5 4.6" />
    </>
  ),
  youtube: (
    <>
      <rect x="2" y="5" width="20" height="14" rx="4" />
      <path d="M10 9.2v5.6l4.8-2.8z" fill="currentColor" />
    </>
  ),
  x: <path d="M4.5 4l15 16M19.5 4l-15 16" />,
  facebook: <path d="M15 3h-2.5A3.5 3.5 0 0 0 9 6.5V10H6.5v3.5H9V21h3.5v-7.5H15l.5-3.5h-3V7a1 1 0 0 1 1-1H15z" />,
  strava: (
    <>
      <path d="M10 3 4.5 14h3.6L10 10.2 11.9 14h3.6z" />
      <path d="m13.6 15.6 1.8 3.6 1.8-3.6" />
    </>
  )
};

export const SocialIcon = ({ platform, ...props }: { platform: SocialPlatform } & SVGProps<SVGSVGElement>) => (
  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={1.8} strokeLinecap="round" strokeLinejoin="round" aria-hidden="true" {...props}>
    {GLYPHS[platform]}
  </svg>
);
