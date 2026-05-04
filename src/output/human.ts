import boxen from 'boxen';
import pc from 'picocolors';
import { useColor } from './mode.ts';

const c = (fn: (s: string) => string) => (s: string) => (useColor() ? fn(s) : s);

export const ui = {
  bold: c(pc.bold),
  dim: c(pc.dim),
  red: c(pc.red),
  green: c(pc.green),
  yellow: c(pc.yellow),
  cyan: c(pc.cyan),
  blue: c(pc.blue),
  magenta: c(pc.magenta),
  gray: c(pc.gray),
  check: () => (useColor() ? pc.green('✓') : '✓'),
  cross: () => (useColor() ? pc.red('✗') : '✗'),
  arrow: () => (useColor() ? pc.cyan('→') : '→'),
};

export const banner = (message: string, title?: string): string =>
  boxen(message, {
    padding: 1,
    margin: 0,
    borderStyle: 'round',
    title,
    titleAlignment: 'left',
  });

export const truncate = (s: string, max: number): string => {
  if (s.length <= max) return s;
  return `${s.slice(0, Math.max(0, max - 1))}…`;
};

export interface Column<T> {
  header: string;
  width?: number;
  render: (row: T) => string;
}

export const renderTable = <T>(rows: T[], columns: Column<T>[]): string => {
  const widths = columns.map((col) => {
    const headerLen = col.header.length;
    const maxBody = rows.reduce((m, r) => Math.max(m, col.render(r).length), 0);
    return col.width ?? Math.max(headerLen, maxBody);
  });
  const fmt = (parts: string[]): string =>
    parts.map((part, i) => part.padEnd(widths[i] ?? 0, ' ')).join('  ');
  const header = fmt(columns.map((c) => ui.bold(c.header)));
  const rule = fmt(widths.map((w) => '─'.repeat(w)));
  const body = rows.map((r) =>
    fmt(columns.map((col) => truncate(col.render(r), widths[columns.indexOf(col)] ?? 0))),
  );
  return [header, rule, ...body].join('\n');
};
