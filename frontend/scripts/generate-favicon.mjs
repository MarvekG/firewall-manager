import { mkdirSync, writeFileSync } from 'node:fs';
import { dirname, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';

const root = resolve(dirname(fileURLToPath(import.meta.url)), '..');
const output = resolve(root, 'public/favicon.svg');

const svg = `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 64 64" role="img" aria-label="Firewall Manager">
  <defs>
    <linearGradient id="bg" x1="8" y1="6" x2="56" y2="58" gradientUnits="userSpaceOnUse">
      <stop offset="0" stop-color="#22d3ee"/>
      <stop offset="1" stop-color="#2563eb"/>
    </linearGradient>
    <linearGradient id="flame" x1="25" y1="18" x2="42" y2="48" gradientUnits="userSpaceOnUse">
      <stop offset="0" stop-color="#fde68a"/>
      <stop offset="0.55" stop-color="#fb923c"/>
      <stop offset="1" stop-color="#ef4444"/>
    </linearGradient>
  </defs>
  <rect width="64" height="64" rx="16" fill="#020617"/>
  <path d="M32 7l20 8v14c0 13.5-7.7 23.1-20 28-12.3-4.9-20-14.5-20-28V15l20-8z" fill="url(#bg)"/>
  <path d="M18 25h28M18 34h28M24 16v31M34 14v38M44 20v23" fill="none" stroke="#e0f2fe" stroke-width="3" stroke-linecap="round" opacity="0.75"/>
  <path d="M37 47c5.5-3.1 7.3-8.8 4.2-13.9-.9 2.7-2.6 4.3-4.9 5.1 1.2-5.4-.9-10.5-5.9-15.2.1 5.6-3.1 8.5-5.8 11.6-3 3.5-2.7 8.7 1.2 11.7 3.2 2.5 7.8 2.7 11.2.7z" fill="url(#flame)" stroke="#fff7ed" stroke-width="2" stroke-linejoin="round"/>
</svg>
`;

mkdirSync(dirname(output), { recursive: true });
writeFileSync(output, svg);
console.log(`Generated ${output}`);
