// Lightweight JWT payload decoder (no signature verification)
export function decodeJwtPayload(token: string): any | null {
  if (!token || token.split('.').length < 2) return null;
  try {
    const payloadPart = token.split('.')[1]
      .replace(/-/g, '+')
      .replace(/_/g, '/');
    // Pad base64 if necessary
    const pad = payloadPart.length % 4;
    const b64 = pad ? payloadPart + '='.repeat(4 - pad) : payloadPart;
    const json = atob(b64);
    return JSON.parse(json);
  } catch (e) {
    console.warn('decodeJwtPayload failed', e);
    return null;
  }
}
