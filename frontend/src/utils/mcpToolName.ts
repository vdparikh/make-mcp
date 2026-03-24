/**
 * MCP tool naming rules (aligned with backend mcpvalidate):
 * - Length 1–128 Unicode code points (typically ASCII for valid names)
 * - Allowed: A–Z, a–z, 0–9, underscore, hyphen, dot
 * - Case-sensitive uniqueness is enforced server-side
 */

const MCP_TOOL_NAME_RE = /^[A-Za-z0-9_.-]{1,128}$/;

export function mcpToolNameLength(s: string): number {
  return [...s].length;
}

/** Returns null if valid; otherwise a short user-facing error message. */
export function validateMcpToolName(name: string): string | null {
  const n = mcpToolNameLength(name);
  if (n < 1 || n > 128) {
    return 'Tool name must be 1–128 characters.';
  }
  if (!MCP_TOOL_NAME_RE.test(name)) {
    return 'Use only letters, digits, underscore (_), hyphen (-), and dot (.) — no spaces or other characters.';
  }
  return null;
}

/**
 * Maps disallowed characters to underscores (for suggested names from flows/templates).
 * Mirrors backend SanitizeToolName for consistency with OpenAPI import.
 */
export function sanitizeMcpToolName(s: string): string {
  let out = '';
  for (const ch of s) {
    const code = ch.codePointAt(0)!;
    if (
      (code >= 0x41 && code <= 0x5a) ||
      (code >= 0x61 && code <= 0x7a) ||
      (code >= 0x30 && code <= 0x39) ||
      ch === '_' ||
      ch === '-' ||
      ch === '.'
    ) {
      out += ch;
    } else {
      out += '_';
    }
  }
  while (out.includes('__')) {
    out = out.replace(/__/g, '_');
  }
  // Match strings.Trim(out, "._-") in backend SanitizeToolName
  out = out.replace(/^[._-]+|[._-]+$/g, '');
  if (out === '') {
    return 'tool';
  }
  const runes = [...out];
  if (runes.length > 128) {
    return runes.slice(0, 128).join('');
  }
  return out;
}
