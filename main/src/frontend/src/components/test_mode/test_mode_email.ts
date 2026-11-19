const TEST_EMAIL_PATTERN = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;

// Test purchases are real purchases for the persona, but personas have
// non-deliverable @ryze.local addresses. Checkout therefore asks for a real
// mailbox before starting the purchase. The backend validates it again.
export const isValidTestEmail = (value: string): boolean => {
  const address = value.trim();
  return address.length > 0 && address.length <= 255 && TEST_EMAIL_PATTERN.test(address);
};
