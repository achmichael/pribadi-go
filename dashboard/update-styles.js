import fs from 'fs';
import path from 'path';
import { fileURLToPath } from 'url';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

const pagesDir = path.join(__dirname, 'src', 'pages');
const compDir = path.join(__dirname, 'src', 'components');

function walkDir(dir, callback) {
  fs.readdirSync(dir).forEach(f => {
    let dirPath = path.join(dir, f);
    let isDirectory = fs.statSync(dirPath).isDirectory();
    isDirectory ? walkDir(dirPath, callback) : callback(dirPath);
  });
}

const replacements = [
  // Typography
  { regex: /text-brand-900/g, replacement: 'text-ink-primary' },
  { regex: /text-brand-800/g, replacement: 'text-ink-primary' },
  { regex: /text-brand-700/g, replacement: 'text-ink-primary' },
  { regex: /text-brand-600/g, replacement: 'text-ink-muted' },
  { regex: /text-brand-500/g, replacement: 'text-ink-muted' },
  { regex: /text-brand-400/g, replacement: 'text-ink-muted/70' },
  { regex: /text-brand-300/g, replacement: 'text-ink-muted/50' },
  
  // Backgrounds
  { regex: /bg-brand-50/g, replacement: 'bg-canvas' },
  { regex: /bg-white/g, replacement: 'bg-surface' },
  
  // Borders
  { regex: /border-brand-200/g, replacement: 'border-ink-primary/10' },
  { regex: /border-brand-300/g, replacement: 'border-ink-primary/20' },
  
  // Accents (Emerald -> accent-primary)
  { regex: /bg-emerald-500/g, replacement: 'bg-accent-primary' },
  { regex: /bg-emerald-600/g, replacement: 'bg-accent-primary' },
  { regex: /hover:bg-emerald-600/g, replacement: 'hover:bg-accent-primary/90' },
  { regex: /hover:bg-emerald-700/g, replacement: 'hover:bg-accent-primary/90' },
  { regex: /text-emerald-700/g, replacement: 'text-accent-primary' },
  { regex: /text-emerald-600/g, replacement: 'text-accent-primary' },
  { regex: /bg-emerald-50/g, replacement: 'bg-accent-primary/10' },
  { regex: /ring-emerald-500\/20/g, replacement: 'ring-accent-primary/15' },
  { regex: /border-emerald-500/g, replacement: 'border-accent-primary' },
  { regex: /border-emerald-300/g, replacement: 'border-accent-primary/50' },
  { regex: /hover:border-emerald-200/g, replacement: 'hover:border-accent-primary/30' },
  { regex: /hover:bg-emerald-50\/30/g, replacement: 'hover:bg-accent-primary/5' },
  { regex: /hover:bg-emerald-50\/50/g, replacement: 'hover:bg-accent-primary/10' },
  
  // Accents (Rose -> accent-danger)
  { regex: /text-rose-600/g, replacement: 'text-accent-danger' },
  { regex: /bg-rose-50/g, replacement: 'bg-accent-danger/10' },
  { regex: /hover:bg-rose-50/g, replacement: 'hover:bg-accent-danger/10' },
  
  // Accents (Amber -> accent-warm)
  { regex: /text-amber-700/g, replacement: 'text-accent-warm' },
  { regex: /bg-amber-100/g, replacement: 'bg-accent-warm/20' },
  { regex: /border-amber-200/g, replacement: 'border-accent-warm/30' },
];

function processFile(filePath) {
  if (!filePath.endsWith('.tsx') && !filePath.endsWith('.ts')) return;
  
  let content = fs.readFileSync(filePath, 'utf8');
  let originalContent = content;
  
  // Apply regex replacements
  replacements.forEach(({ regex, replacement }) => {
    content = content.replace(regex, replacement);
  });
  
  // Add font-sora to h1, h2, h3
  content = content.replace(/<(h[1-3])\s+className="([^"]+)"/g, (match, tag, classes) => {
    if (!classes.includes('font-sora')) {
      return `<${tag} className="font-sora ${classes}"`;
    }
    return match;
  });
  
  // Make cards rounded-[16px] and shadow-soft, remove borders if they had them
  content = content.replace(/className="([^"]*rounded-2xl[^"]*bg-surface[^"]*)"/g, (match, classes) => {
    let newClasses = classes
      .replace('rounded-2xl', 'rounded-[16px]')
      .replace('shadow-sm', 'shadow-soft')
      .replace('border-ink-primary/10', 'border-transparent hover:-translate-y-[2px] hover:shadow-md transition-all duration-200');
    return `className="${newClasses}"`;
  });

  content = content.replace(/className="([^"]*rounded-lg[^"]*bg-surface[^"]*)"/g, (match, classes) => {
    let newClasses = classes
      .replace('rounded-lg', 'rounded-[16px]')
      .replace('shadow-sm', 'shadow-soft')
      .replace('border ', 'border-transparent hover:-translate-y-[2px] hover:shadow-md transition-all duration-200 ');
    return `className="${newClasses}"`;
  });

  if (content !== originalContent) {
    fs.writeFileSync(filePath, content, 'utf8');
    console.log(`Updated ${filePath}`);
  }
}

walkDir(pagesDir, processFile);
walkDir(compDir, processFile);
