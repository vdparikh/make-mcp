/**
 * Helpers to convert backend WebAuthn options (JSON with base64url) to browser PublicKeyCredential API format.
 */

/** Decode base64 or base64url string to ArrayBuffer. */
function base64ToBuffer(input: string): ArrayBuffer {
  const base64 = input.includes('+') || input.includes('/') ? input : input.replace(/-/g, '+').replace(/_/g, '/');
  const pad = base64.length % 4;
  const padded = pad ? base64 + '='.repeat(4 - pad) : base64;
  const binary = atob(padded);
  const bytes = new Uint8Array(binary.length);
  for (let i = 0; i < binary.length; i++) bytes[i] = binary.charCodeAt(i);
  return bytes.buffer;
}

function bufferToBase64url(buffer: ArrayBuffer): string {
  const bytes = new Uint8Array(buffer);
  let binary = '';
  for (let i = 0; i < bytes.length; i++) binary += String.fromCharCode(bytes[i]);
  return btoa(binary).replace(/\+/g, '-').replace(/\//g, '_').replace(/=+$/, '');
}

export interface CredentialCreationOptionsFromServer {
  rp: { name: string; id?: string };
  user: { id: string; name: string; displayName: string };
  challenge: string;
  pubKeyCredParams: { type: string; alg: number }[];
  timeout?: number;
  attestation?: string;
  authenticatorSelection?: Record<string, unknown>;
}

export function toCreationOptions(
  opts: CredentialCreationOptionsFromServer
): PublicKeyCredentialCreationOptions {
  return {
    rp: opts.rp,
    user: {
      id: base64ToBuffer(opts.user.id),
      name: opts.user.name,
      displayName: opts.user.displayName,
    },
    challenge: base64ToBuffer(opts.challenge),
    pubKeyCredParams: (opts.pubKeyCredParams ?? [{ type: 'public-key', alg: -7 }]) as PublicKeyCredentialParameters[],
    timeout: opts.timeout ?? 60000,
    attestation: (opts.attestation as AttestationConveyancePreference) ?? 'none',
    authenticatorSelection: opts.authenticatorSelection as AuthenticatorSelectionCriteria | undefined,
  };
}

export interface CredentialRequestOptionsFromServer {
  challenge: string;
  timeout?: number;
  rpId?: string;
  allowCredentials?: { type: string; id: string; transports?: string[] }[];
  userVerification?: string;
}

export function toRequestOptions(
  opts: CredentialRequestOptionsFromServer
): PublicKeyCredentialRequestOptions {
  return {
    challenge: base64ToBuffer(opts.challenge),
    timeout: opts.timeout ?? 60000,
    rpId: opts.rpId,
    allowCredentials: opts.allowCredentials?.map((c) => ({
      type: 'public-key' as PublicKeyCredentialType,
      id: base64ToBuffer(c.id),
      transports: c.transports as AuthenticatorTransport[] | undefined,
    })),
    userVerification: (opts.userVerification as UserVerificationRequirement) ?? 'preferred',
  };
}

export function credentialCreationResponseToJSON(cred: PublicKeyCredential): {
  id: string;
  rawId: string;
  type: string;
  response: {
    clientDataJSON: string;
    attestationObject: string;
    transports?: string[];
  };
} {
  const response = cred.response as AuthenticatorAttestationResponse;
  return {
    id: cred.id,
    rawId: bufferToBase64url(cred.rawId),
    type: cred.type,
    response: {
      clientDataJSON: bufferToBase64url(response.clientDataJSON),
      attestationObject: bufferToBase64url(response.attestationObject),
      transports: response.getTransports?.() ?? undefined,
    },
  };
}

export function credentialAssertionResponseToJSON(cred: PublicKeyCredential): {
  id: string;
  rawId: string;
  type: string;
  response: {
    clientDataJSON: string;
    authenticatorData: string;
    signature: string;
    userHandle: string | null;
  };
} {
  const response = cred.response as AuthenticatorAssertionResponse;
  return {
    id: cred.id,
    rawId: bufferToBase64url(cred.rawId),
    type: cred.type,
    response: {
      clientDataJSON: bufferToBase64url(response.clientDataJSON),
      authenticatorData: bufferToBase64url(response.authenticatorData),
      signature: bufferToBase64url(response.signature),
      userHandle: response.userHandle ? bufferToBase64url(response.userHandle) : null,
    },
  };
}
