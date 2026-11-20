// The Portuguese taxpayer number (NIF) a buyer may add at checkout for their
// purchase document. Optional: empty means "final consumer". The backend
// validates it again (services/taxid).
export const normalizeTaxId = (value: string): string => value.replace(/\s+/g, "").toUpperCase().replace(/^PT/, "");

const isValidNif = (nif: string): boolean => {
  if (!/^[1-9][0-9]{8}$/.test(nif)) return false;
  let sum = 0;
  for (let index = 0; index < 8; index += 1) sum += Number(nif[index]) * (9 - index);
  const check = 11 - (sum % 11);
  return Number(nif[8]) === (check >= 10 ? 0 : check);
};

/** True when the field is empty or holds a valid NIF. */
export const isValidOptionalTaxId = (value: string): boolean => {
  const nif = normalizeTaxId(value);
  return nif.length === 0 || isValidNif(nif);
};
