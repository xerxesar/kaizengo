export interface Pos {
  line: number;
  character: number;
}

export function offsetToPosition(text: string, offset: number): Pos {
  const cap = Math.max(0, Math.min(offset, text.length));
  let line = 0;
  let lastNl = -1;
  for (let i = 0; i < cap; i++) {
    if (text.charCodeAt(i) === 10) {
      line++;
      lastNl = i;
    }
  }
  return { line, character: cap - (lastNl + 1) };
}

export function positionToOffset(text: string, line: number, character: number): number {
  let current = 0;
  let i = 0;
  while (current < line && i < text.length) {
    if (text.charCodeAt(i) === 10) {
      current++;
    }
    i++;
  }
  return Math.min(i + character, text.length);
}
