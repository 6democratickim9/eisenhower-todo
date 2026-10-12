import type { SVGProps } from 'react';

const Base = ({children,...props}:SVGProps<SVGSVGElement>) => <svg viewBox="0 0 24 24" aria-hidden="true" {...props}>{children}</svg>;
export const HomeIcon=(p:SVGProps<SVGSVGElement>)=><Base {...p}><path d="M3.5 10.5 12 3.8l8.5 6.7v9.2h-6.1v-5.8H9.6v5.8H3.5z"/></Base>;
export const ListIcon=(p:SVGProps<SVGSVGElement>)=><Base {...p}><path d="M9 6h11M9 12h11M9 18h11M4 6h.01M4 12h.01M4 18h.01"/></Base>;
export const GridIcon=(p:SVGProps<SVGSVGElement>)=><Base {...p}><rect x="4" y="4" width="6" height="6" rx="1"/><rect x="14" y="4" width="6" height="6" rx="1"/><rect x="4" y="14" width="6" height="6" rx="1"/><rect x="14" y="14" width="6" height="6" rx="1"/></Base>;
export const CheckIcon=(p:SVGProps<SVGSVGElement>)=><Base {...p}><circle cx="12" cy="12" r="9"/><path d="m7.5 12 3 3 6-7"/></Base>;
export const CalendarIcon=(p:SVGProps<SVGSVGElement>)=><Base {...p}><rect x="3" y="5" width="18" height="16" rx="2"/><path d="M8 3v4M16 3v4M3 10h18"/></Base>;
export const TagIcon=(p:SVGProps<SVGSVGElement>)=><Base {...p}><path d="M20.4 13.6 13.6 20.4a2 2 0 0 1-2.8 0L3.6 13.2V4h9.2l7.6 7.6a1.4 1.4 0 0 1 0 2z"/><circle cx="8.2" cy="8.2" r="1.2"/></Base>;
export const ChartIcon=(p:SVGProps<SVGSVGElement>)=><Base {...p}><path d="M4 20V11h4v9M10 20V5h4v15M16 20v-7h4v7"/></Base>;

export const ArchiveIcon=(p:SVGProps<SVGSVGElement>)=><Base {...p}><path d="M4 7h16v13H4z"/><path d="M3 4h18v4H3zM9 12h6"/></Base>;
export const SettingsIcon=(p:SVGProps<SVGSVGElement>)=><Base {...p}><circle cx="12" cy="12" r="3"/><path d="M19.4 15a1.7 1.7 0 0 0 .3 1.9l.1.1-2.8 2.8-.1-.1a1.7 1.7 0 0 0-1.9-.3A1.7 1.7 0 0 0 14 21v.2h-4V21a1.7 1.7 0 0 0-1-1.6 1.7 1.7 0 0 0-1.9.3l-.1.1L4.2 17l.1-.1A1.7 1.7 0 0 0 4.6 15 1.7 1.7 0 0 0 3 14H2.8v-4H3a1.7 1.7 0 0 0 1.6-1 1.7 1.7 0 0 0-.3-1.9L4.2 7 7 4.2l.1.1a1.7 1.7 0 0 0 1.9.3A1.7 1.7 0 0 0 10 3V2.8h4V3a1.7 1.7 0 0 0 1 1.6 1.7 1.7 0 0 0 1.9-.3l.1-.1L19.8 7l-.1.1a1.7 1.7 0 0 0-.3 1.9 1.7 1.7 0 0 0 1.6 1h.2v4H21a1.7 1.7 0 0 0-1.6 1z"/></Base>;
export const SearchIcon=(p:SVGProps<SVGSVGElement>)=><Base {...p}><circle cx="10.5" cy="10.5" r="6.5"/><path d="m15.5 15.5 5 5"/></Base>;
export const PlusIcon=(p:SVGProps<SVGSVGElement>)=><Base {...p}><path d="M12 5v14M5 12h14"/></Base>;
