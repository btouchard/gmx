// GMX language mode for CodeMirror 6
import { LRLanguage, LanguageSupport } from 'https://esm.sh/@codemirror/language@6.12.1';
import { styleTags, tags as t } from 'https://esm.sh/@lezer/highlight@1.2.3';
import { parser } from 'https://esm.sh/@lezer/javascript@1.4.21';

// GMX keywords
const gmxKeywords = [
  'model', 'service', 'func', 'let', 'const', 'try', 'render', 'error',
  'import', 'from', 'if', 'else', 'return', 'provider', 'as'
];

// GMX types
const gmxTypes = [
  'uuid', 'string', 'int', 'float', 'bool', 'datetime', 'error'
];

// GMX annotations
const gmxAnnotations = [
  '@pk', '@default', '@min', '@max', '@email', '@unique', '@scoped',
  '@env', '@relation', '@references'
];

// Create a simple GMX language extension
const gmxHighlight = styleTags({
  'model service func let const try render error import from if else return provider as': t.keyword,
  'uuid string int float bool datetime': t.typeName,
  'Number': t.number,
  'String': t.string,
  'LineComment BlockComment': t.lineComment,
  'Identifier': t.variableName,
  'FunctionDeclaration/Identifier': t.function(t.variableName),
  '( )': t.paren,
  '{ }': t.brace,
  '[ ]': t.squareBracket,
  ': , ;': t.separator,
});

// Build a basic GMX language
const gmxLanguageBase = LRLanguage.define({
  name: 'gmx',
  parser: parser.configure({
    props: [gmxHighlight]
  }),
  languageData: {
    commentTokens: { line: '//', block: { open: '/*', close: '*/' } }
  }
});

// Create language support with highlighting for sections
export function gmxLanguage() {
  return new LanguageSupport(gmxLanguageBase, [
    // Additional extensions can be added here
  ]);
}
