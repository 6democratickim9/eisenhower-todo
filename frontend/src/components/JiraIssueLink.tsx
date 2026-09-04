import type { KeyboardEvent, MouseEvent } from 'react';
import { jiraIssueUrl } from '../utils/jira';

export function JiraIssueLink({
  issueKey,
  url,
  baseUrl,
  className = '',
}: {
  issueKey: string;
  url?: string;
  baseUrl?: string;
  className?: string;
}) {
  const href = jiraIssueUrl(issueKey, url, baseUrl);
  if (!href) return <span className={className}>{issueKey}</span>;

  const stop = (event: MouseEvent<HTMLElement>) => event.stopPropagation();
  const stopKey = (event: KeyboardEvent<HTMLAnchorElement>) => event.stopPropagation();
  return (
    <a
      className={`jira-issue-link ${className}`.trim()}
      href={href}
      target="_blank"
      rel="noopener noreferrer"
      draggable={false}
      title={`Open ${issueKey} in Jira`}
      onClick={stop}
      onMouseDown={stop}
      onKeyDown={stopKey}
    >
      <span>{issueKey}</span><span className="jira-external-mark" aria-hidden="true">↗</span>
    </a>
  );
}
