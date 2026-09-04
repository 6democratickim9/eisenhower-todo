/**
 * Build the browser URL for a Jira Issue Key.
 *
 * Backend responses normally include the canonical `url`.  The Base URL
 * fallback keeps manually-added / not-yet-synced tickets clickable too.
 */
export function jiraIssueUrl(issueKey: string, explicitUrl?: string, baseUrl?: string): string | undefined {
  const direct = explicitUrl?.trim();
  if (direct) return direct;

  const key = issueKey.trim().toUpperCase();
  if (!key) return undefined;

  const base = normalizeJiraBaseUrl(baseUrl || '');
  return base ? `${base}/browse/${encodeURIComponent(key)}` : undefined;
}

export function normalizeJiraBaseUrl(raw: string): string {
  let value = raw.trim().replace(/\/+$/, '');
  if (!value) return '';

  // Keep the same forgiving behavior as the Python backend when a user
  // pastes a ticket URL or REST API URL into Jira Base URL.
  value = value.replace(/\/browse\/[^/?#]+(?:[?#].*)?$/i, '');
  value = value.replace(/\/browse$/i, '');
  value = value.replace(/\/rest\/api\/.*$/i, '');
  return value.replace(/[?#].*$/, '').replace(/\/+$/, '');
}
