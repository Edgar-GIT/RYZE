import type { ReactNode } from "react";
import { Link } from "react-router-dom";

import { joinClassNames } from "@utils/class_names";

import ui from "./public_ui.module.css";

type PublicActionVariant = "primary" | "ink" | "ghostOnDark" | "ghostOnLight" | "quiet";

interface PublicActionBaseProps {
  children: ReactNode;
  variant?: PublicActionVariant;
  icon?: ReactNode;
  className?: string;
}

interface PublicActionRouteProps extends PublicActionBaseProps {
  to: string;
  href?: never;
  type?: never;
  disabled?: never;
  onClick?: never;
}

interface PublicActionAnchorProps extends PublicActionBaseProps {
  href: string;
  to?: never;
  type?: never;
  disabled?: never;
  onClick?: never;
}

interface PublicActionButtonProps extends PublicActionBaseProps {
  to?: never;
  href?: never;
  type?: "button" | "submit";
  disabled?: boolean;
  onClick?: () => void;
}

type PublicActionProps = PublicActionRouteProps | PublicActionAnchorProps | PublicActionButtonProps;

// PublicAction is the public website's only button/link primitive. It is kept
// separate from the shared Button so redesigned pages never restyle the
// protected Services, Exercise Library or Home program-highlight buttons.
export const PublicAction = ({
  children,
  variant = "primary",
  to,
  href,
  type = "button",
  disabled = false,
  icon,
  className,
  onClick
}: PublicActionProps) => {
  const actionClassName = joinClassNames(ui.action, ui[variant], className);
  const content = (
    <>
      <span>{children}</span>
      {icon ? <span aria-hidden="true">{icon}</span> : null}
    </>
  );

  if (to) {
    return (
      <Link className={actionClassName} to={to}>
        {content}
      </Link>
    );
  }

  if (href) {
    const isExternal = href.startsWith("http");

    return (
      <a
        className={actionClassName}
        href={href}
        target={isExternal ? "_blank" : undefined}
        rel={isExternal ? "noreferrer" : undefined}
      >
        {content}
        {isExternal ? <span className={ui.srOnly}> (opens in a new tab)</span> : null}
      </a>
    );
  }

  return (
    <button className={actionClassName} type={type} disabled={disabled} onClick={onClick}>
      {content}
    </button>
  );
};

// BrandStop renders the cyan full stop of the official slogan lockup.
export const BrandStop = () => <span className={ui.stop}>.</span>;

// A seamless wave: quadratic arcs that alternate above and below the baseline
// (each T mirrors the previous control point), drawn over twice the visible
// width so translating it by half loops without a seam (every period divides
// the 1440-unit half width).
const WAVE_WIDTH = 2880;
const WAVE_HEIGHT = 300;

const wavePath = (period: number, amplitude: number, baseline: number) => {
  const half = period / 2;
  let d = `M0 ${baseline} Q ${half / 2} ${baseline - amplitude * 2} ${half} ${baseline}`;
  for (let x = half * 2; x <= WAVE_WIDTH; x += half) {
    d += ` T ${x} ${baseline}`;
  }
  return `${d} V ${WAVE_HEIGHT} H 0 Z`;
};

const WAVE_LAYERS = [
  { className: ui.waveBack, d: wavePath(720, 26, 120), fill: "rgba(255, 255, 255, 0.28)" },
  { className: ui.waveMid, d: wavePath(480, 22, 170), fill: "rgba(4, 129, 220, 0.32)" },
  { className: ui.waveFront, d: wavePath(1440, 18, 220), fill: "rgba(7, 42, 119, 0.4)" }
] as const;

// Waves is the decorative motif of the light-blue panels and the auth page.
export const Waves = ({ className }: { className?: string }) => (
  <div className={joinClassNames(ui.waves, className)} aria-hidden="true">
    <svg viewBox={`0 0 ${WAVE_WIDTH} ${WAVE_HEIGHT}`} preserveAspectRatio="none" focusable="false">
      {WAVE_LAYERS.map((layer) => (
        <path key={layer.className} className={layer.className} d={layer.d} fill={layer.fill} />
      ))}
    </svg>
  </div>
);
