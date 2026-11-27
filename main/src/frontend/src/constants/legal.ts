import { CONTACT_EMAIL } from "@/constants/contact";

/**
 * The legal identity shown in the Terms, Privacy, Cookies and Legal notice
 * pages (owner request, 2026-11-23). Values in [brackets] are not known to the
 * code base yet and must be filled in by the owner before going live; the
 * documents should also be reviewed by a lawyer.
 */
export const LEGAL_ENTITY = {
  name: "[Nome legal da empresa]",
  taxId: "[NIF/NIPC]",
  address: "[Morada da sede]",
  registry: "[Conservatória do Registo Comercial e número de matrícula]",
  shareCapital: "[Capital social]",
  email: CONTACT_EMAIL,
  privacyEmail: CONTACT_EMAIL,
  minimumAge: "16",
  // Date of the current version of the documents.
  updated: "2026-11-23",
  version: "1.0"
} as const;

// The Portuguese electronic complaints book and consumer dispute bodies.
export const COMPLAINTS_BOOK_URL = "https://www.livroreclamacoes.pt";
export const CNIACC_URL = "https://www.cniacc.pt";
export const CONSUMER_PORTAL_URL = "https://www.consumidor.gov.pt";
export const CNPD_URL = "https://www.cnpd.pt";

export const LEGAL_PATHS = {
  terms: "/legal/termos",
  privacy: "/legal/privacidade",
  cookies: "/legal/cookies",
  notice: "/legal/aviso-legal"
} as const;
