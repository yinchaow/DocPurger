import React from 'react';

interface IconProps {
  className?: string;
}

/**
 * Unified Fluent Design 2 brand logo for 文档清匣 (DocPurger) —
 * matches the 512x512 native build/appicon.png pixel-for-pixel in design language.
 */
export const AppLogo: React.FC<IconProps> = ({ className = 'w-9 h-9' }) => (
  <svg viewBox="0 0 64 64" fill="none" xmlns="http://www.w3.org/2000/svg" className={className}>
    <defs>
      <linearGradient id="dp-bg" x1="6" y1="4" x2="58" y2="60" gradientUnits="userSpaceOnUse">
        <stop offset="0%" stopColor="#36B4F8" />
        <stop offset="52%" stopColor="#0078D4" />
        <stop offset="100%" stopColor="#004E98" />
      </linearGradient>
      <linearGradient id="dp-doc" x1="16" y1="14" x2="40" y2="48" gradientUnits="userSpaceOnUse">
        <stop offset="0%" stopColor="#FFFFFF" />
        <stop offset="100%" stopColor="#ECF4FF" />
      </linearGradient>
      <linearGradient id="dp-badge" x1="34" y1="32" x2="52" y2="52" gradientUnits="userSpaceOnUse">
        <stop offset="0%" stopColor="#185AB4" />
        <stop offset="100%" stopColor="#0A2E66" />
      </linearGradient>
    </defs>
    {/* Fluent squircle base */}
    <rect x="3" y="3" width="58" height="58" rx="15" fill="url(#dp-bg)" />
    <rect x="3.75" y="3.75" width="56.5" height="56.5" rx="14.25" stroke="white" strokeOpacity="0.32" strokeWidth="1.5" />
    {/* Back frosted sheet */}
    <rect x="21" y="13" width="24" height="30" rx="4.5" fill="#E0F2FF" fillOpacity="0.45" />
    {/* Front pearl-white document */}
    <rect x="15" y="16" width="26" height="33" rx="4.5" fill="url(#dp-doc)" />
    {/* Document lines + warm amber keyword highlight */}
    <rect x="20" y="22" width="13" height="2.6" rx="1.3" fill="#0078D4" fillOpacity="0.88" />
    <rect x="20" y="28" width="6.5" height="2.4" rx="1.2" fill="#94A3B8" fillOpacity="0.65" />
    <rect x="28" y="27.2" width="8.5" height="4" rx="2" fill="#FBBF24" />
    <rect x="20" y="34" width="15" height="2.4" rx="1.2" fill="#94A3B8" fillOpacity="0.58" />
    <rect x="20" y="39.5" width="10" height="2.4" rx="1.2" fill="#94A3B8" fillOpacity="0.45" />
    {/* Crisp sparkle symbolizing purification */}
    <path d="M47 14L48.1 17.4L51.5 18.5L48.1 19.6L47 23L45.9 19.6L42.5 18.5L45.9 17.4L47 14Z" fill="white" fillOpacity="0.92" />
    {/* Floating ocean-acrylic loupe badge */}
    <circle cx="42.5" cy="42" r="10.5" fill="url(#dp-badge)" stroke="#D2F2FF" strokeWidth="1.4" />
    <circle cx="41.2" cy="40.8" r="3.8" fill="#7DD3FC" fillOpacity="0.28" stroke="white" strokeWidth="1.8" />
    <line x1="44.2" y1="43.8" x2="47.6" y2="47.2" stroke="#7DD3FC" strokeWidth="2.1" strokeLinecap="round" />
  </svg>
);

/**
 * Crisp Windows 11 Fluent-style layered colorful document type icons
 * matching the Fluent Mica preview aesthetic.
 */
export const FluentFileIcon: React.FC<{ category: string; className?: string }> = ({
  category,
  className = 'w-8 h-8',
}) => {
  switch (category) {
    case 'word':
      return (
        <svg viewBox="0 0 36 36" fill="none" className={className}>
          <rect x="7" y="4" width="23" height="28" rx="4" fill="#2B7CD3" />
          <path d="M16 11h10M16 16h10M16 21h10M16 26h7" stroke="white" strokeOpacity="0.6" strokeWidth="1.8" strokeLinecap="round" />
          <rect x="3" y="10" width="16" height="16" rx="3.2" fill="#185ABD" />
          <text x="11" y="21.5" textAnchor="middle" fill="white" fontSize="10.5" fontWeight="700" fontFamily="Segoe UI, sans-serif">W</text>
        </svg>
      );
    case 'excel':
      return (
        <svg viewBox="0 0 36 36" fill="none" className={className}>
          <rect x="7" y="4" width="23" height="28" rx="4" fill="#21A366" />
          <path d="M18 10v16M24 10v16M15 15h12M15 21h12" stroke="white" strokeOpacity="0.45" strokeWidth="1.4" />
          <rect x="3" y="10" width="16" height="16" rx="3.2" fill="#107C41" />
          <text x="11" y="21.5" textAnchor="middle" fill="white" fontSize="10.5" fontWeight="700" fontFamily="Segoe UI, sans-serif">X</text>
        </svg>
      );
    case 'ppt':
      return (
        <svg viewBox="0 0 36 36" fill="none" className={className}>
          <rect x="7" y="4" width="23" height="28" rx="4" fill="#E36544" />
          <circle cx="22" cy="18" r="5" fill="white" fillOpacity="0.42" />
          <path d="M22 13a5 5 0 0 1 5 5h-5v-5z" fill="white" fillOpacity="0.82" />
          <rect x="3" y="10" width="16" height="16" rx="3.2" fill="#C43E1C" />
          <text x="11" y="21.5" textAnchor="middle" fill="white" fontSize="10.5" fontWeight="700" fontFamily="Segoe UI, sans-serif">P</text>
        </svg>
      );
    case 'pdf':
      return (
        <svg viewBox="0 0 36 36" fill="none" className={className}>
          <path d="M8 4h14l7 7v21a2.5 2.5 0 0 1-2.5 2.5h-18A2.5 2.5 0 0 1 6 32V6.5A2.5 2.5 0 0 1 8.5 4z" fill="#E53935" />
          <path d="M22 4v5.5A1.5 1.5 0 0 0 23.5 11H29l-7-7z" fill="#FFCDD2" />
          <rect x="9.5" y="18.5" width="16" height="9" rx="2" fill="white" fillOpacity="0.92" />
          <text x="17.5" y="25.2" textAnchor="middle" fill="#C62828" fontSize="6.8" fontWeight="800" fontFamily="Segoe UI, sans-serif">PDF</text>
        </svg>
      );
    case 'ofd':
      return (
        <svg viewBox="0 0 36 36" fill="none" className={className}>
          <path d="M8 4h14l7 7v21a2.5 2.5 0 0 1-2.5 2.5h-18A2.5 2.5 0 0 1 6 32V6.5A2.5 2.5 0 0 1 8.5 4z" fill="#0F9D9A" />
          <path d="M22 4v5.5A1.5 1.5 0 0 0 23.5 11H29l-7-7z" fill="#B2DFDB" />
          <rect x="9.5" y="18.5" width="16" height="9" rx="2" fill="white" fillOpacity="0.92" />
          <text x="17.5" y="25.2" textAnchor="middle" fill="#00695C" fontSize="6.8" fontWeight="800" fontFamily="Segoe UI, sans-serif">OFD</text>
        </svg>
      );
    case 'archive':
      return (
        <svg viewBox="0 0 36 36" fill="none" className={className}>
          <path d="M5 9.5A2.5 2.5 0 0 1 7.5 7h6.6l2.4 2.5H28.5A2.5 2.5 0 0 1 31 12v16.5A2.5 2.5 0 0 1 28.5 31h-21A2.5 2.5 0 0 1 5 28.5v-19z" fill="#F2B83B" />
          <path d="M5 13h26v15.5A2.5 2.5 0 0 1 28.5 31h-21A2.5 2.5 0 0 1 5 28.5V13z" fill="#F6C851" />
          <rect x="16" y="12" width="4" height="19" fill="#B45309" fillOpacity="0.28" />
          <rect x="16.5" y="14" width="3" height="2.2" rx="0.6" fill="white" />
          <rect x="16.5" y="18" width="3" height="2.2" rx="0.6" fill="white" />
          <rect x="16.5" y="22" width="3" height="2.2" rx="0.6" fill="white" />
        </svg>
      );
    case 'text':
      return (
        <svg viewBox="0 0 36 36" fill="none" className={className}>
          <path d="M8 4h14l7 7v21a2.5 2.5 0 0 1-2.5 2.5h-18A2.5 2.5 0 0 1 6 32V6.5A2.5 2.5 0 0 1 8.5 4z" fill="#64748B" />
          <path d="M22 4v5.5A1.5 1.5 0 0 0 23.5 11H29l-7-7z" fill="#CBD5E1" />
          <path d="M11 16h13M11 21h13M11 26h9" stroke="white" strokeOpacity="0.88" strokeWidth="2" strokeLinecap="round" />
        </svg>
      );
    default:
      return (
        <svg viewBox="0 0 36 36" fill="none" className={className}>
          <path d="M8 4h14l7 7v21a2.5 2.5 0 0 1-2.5 2.5h-18A2.5 2.5 0 0 1 6 32V6.5A2.5 2.5 0 0 1 8.5 4z" fill="#94A3B8" />
          <path d="M22 4v5.5A1.5 1.5 0 0 0 23.5 11H29l-7-7z" fill="#E2E8F0" />
        </svg>
      );
  }
};
