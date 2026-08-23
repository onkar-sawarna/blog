// Shiki writes its background and colors as inline styles on <pre>, which beat
// anything in global.css. So the palette has to live here, not there.
import type { ThemeRegistration } from 'shiki';

const fg = '#E8EFEA';
const bg = '#12261F';

export const codeTheme: ThemeRegistration = {
  name: 'onkar',
  type: 'dark',
  colors: {
    'editor.foreground': fg,
    'editor.background': bg,
  },
  settings: [
    { scope: ['comment', 'punctuation.definition.comment'], settings: { foreground: '#7E9A8C', fontStyle: 'italic' } },
    { scope: ['string', 'string.quoted', 'constant.character'], settings: { foreground: '#A8CBB4' } },
    { scope: ['constant.numeric', 'constant.language', 'constant.other'], settings: { foreground: '#C4A24A' } },
    { scope: ['keyword', 'storage', 'storage.type', 'keyword.operator.new'], settings: { foreground: '#E07A4A' } },
    { scope: ['entity.name.function', 'support.function', 'meta.function-call'], settings: { foreground: '#8FB9A0' } },
    { scope: ['entity.name.type', 'support.type', 'support.class', 'entity.name.class'], settings: { foreground: '#B5D4C4' } },
    { scope: ['variable', 'variable.parameter', 'meta.definition.variable'], settings: { foreground: fg } },
    { scope: ['punctuation', 'meta.brace', 'keyword.operator'], settings: { foreground: '#8AA396' } },
    { scope: ['entity.name.tag', 'support.type.property-name'], settings: { foreground: '#8FB9A0' } },
    { scope: ['markup.inserted'], settings: { foreground: '#8FB9A0' } },
    { scope: ['markup.deleted'], settings: { foreground: '#E07A4A' } },
  ],
};
