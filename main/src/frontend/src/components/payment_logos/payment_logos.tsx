// Payment marks drawn inline (no external images), used to show which methods
// and card brands the checkout accepts. Decorative: the method names are
// always written next to them.

export const VisaLogo = () => (
  <svg viewBox="0 0 64 40" aria-hidden="true" focusable="false">
    <rect width="64" height="40" rx="6" fill="#ffffff" />
    <text x="32" y="27" textAnchor="middle" fontFamily="Arial, Helvetica, sans-serif" fontSize="17" fontStyle="italic" fontWeight="900" fill="#1a1f71" letterSpacing="0.5">
      VISA
    </text>
  </svg>
);

export const MastercardLogo = () => (
  <svg viewBox="0 0 64 40" aria-hidden="true" focusable="false">
    <rect width="64" height="40" rx="6" fill="#ffffff" />
    <circle cx="26" cy="20" r="11" fill="#eb001b" />
    <circle cx="38" cy="20" r="11" fill="#f79e1b" />
    <path d="M32 10.8a11 11 0 0 1 0 18.4a11 11 0 0 1 0-18.4z" fill="#ff5f00" />
  </svg>
);

export const MaestroLogo = () => (
  <svg viewBox="0 0 64 40" aria-hidden="true" focusable="false">
    <rect width="64" height="40" rx="6" fill="#ffffff" />
    <circle cx="26" cy="20" r="11" fill="#0099df" />
    <circle cx="38" cy="20" r="11" fill="#e7001b" />
    <path d="M32 10.8a11 11 0 0 1 0 18.4a11 11 0 0 1 0-18.4z" fill="#6c6bbd" />
  </svg>
);

export const AmexLogo = () => (
  <svg viewBox="0 0 64 40" aria-hidden="true" focusable="false">
    <rect width="64" height="40" rx="6" fill="#2e77bc" />
    <text x="32" y="25" textAnchor="middle" fontFamily="Arial, Helvetica, sans-serif" fontSize="12.5" fontWeight="900" fill="#ffffff" letterSpacing="0.8">
      AMEX
    </text>
  </svg>
);

export const MbWayLogo = () => (
  <svg viewBox="0 0 96 40" aria-hidden="true" focusable="false">
    <rect width="96" height="40" rx="6" fill="#ffffff" />
    <rect x="8" y="8" width="30" height="24" rx="4" fill="#1d1d1b" />
    <text x="23" y="25" textAnchor="middle" fontFamily="Arial, Helvetica, sans-serif" fontSize="13" fontWeight="900" fill="#ffffff">
      MB
    </text>
    <text x="65" y="26" textAnchor="middle" fontFamily="Arial, Helvetica, sans-serif" fontSize="15" fontWeight="900" fill="#d60e1f" letterSpacing="0.5">
      WAY
    </text>
  </svg>
);

export const PayPalLogo = () => (
  <svg viewBox="0 0 96 40" aria-hidden="true" focusable="false">
    <rect width="96" height="40" rx="6" fill="#ffffff" />
    <path d="M14 30 L18 10 H25 C30 10 32 13 31 17 C30 22 26 24 22 24 H19.5 L18.3 30 Z" fill="#003087" />
    <path d="M18.5 32 L22 15 H28 C32 15 33.5 17.5 33 20.5 C32 25 28.5 27 25 27 H23 L22 32 Z" fill="#009cde" opacity="0.9" />
    <text x="40" y="26" fontFamily="Arial, Helvetica, sans-serif" fontSize="15" fontStyle="italic" fontWeight="900" fill="#003087">
      Pay
    </text>
    <text x="65" y="26" fontFamily="Arial, Helvetica, sans-serif" fontSize="15" fontStyle="italic" fontWeight="900" fill="#009cde">
      Pal
    </text>
  </svg>
);

export const StripeLogo = () => (
  <svg viewBox="0 0 60 25" aria-hidden="true" focusable="false">
    <text x="0" y="19" fontFamily="Arial, Helvetica, sans-serif" fontSize="21" fontWeight="900" fill="#ffffff" letterSpacing="-0.8">
      stripe
    </text>
  </svg>
);

export const CARD_LOGOS = [VisaLogo, MastercardLogo, AmexLogo, MaestroLogo];
