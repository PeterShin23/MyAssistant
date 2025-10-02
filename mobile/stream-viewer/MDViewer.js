import React, { memo } from 'react';
import { ScrollView, View, Text, useColorScheme, StyleSheet, Platform, Linking } from 'react-native';
import Markdown, { MarkdownIt } from '@ronradtke/react-native-markdown-display';
import SyntaxHighlighter from 'react-native-syntax-highlighter';

// ---------- load styles from react-syntax-highlighter (NOT the RN wrapper) ----------
let atomOneDark, atomOneLight;
(function loadStyles() {
  try {
    // Most toolchains
    ({ atomOneDark, atomOneLight } = require('react-syntax-highlighter/styles/hljs'));
  } catch {
    try {
      // ESM path
      ({ atomOneDark, atomOneLight } = require('react-syntax-highlighter/dist/esm/styles/hljs'));
    } catch {
      try {
        // CJS path
        ({ atomOneDark, atomOneLight } = require('react-syntax-highlighter/dist/cjs/styles/hljs'));
      } catch {
        atomOneDark = atomOneLight = null; // fallback will render plain blocks
      }
    }
  }
})();

// ---------- markdown-it config (GFM-ish) ----------
const md = MarkdownIt({
  html: false,
  linkify: true,
  typographer: true,
})
  .enable(['table', 'strikethrough'])
  .disable(['smartquotes']); // keep quotes literal for code-heavy content

function CodeBlock({ code, language, isDark, nodeKey }) {
  // If styles failed to load, render a simple, styled Text block (no colors)
  if (!atomOneDark || !atomOneLight) {
    return (
      <View key={nodeKey} style={styles.fallbackCodeWrap}>
        <Text selectable style={styles.fallbackCodeText}>{code}</Text>
      </View>
    );
  }

  return (
    <View key={nodeKey} style={styles.codeBlockWrap}>
      <SyntaxHighlighter
        language={language || undefined}
        style={isDark ? atomOneDark : atomOneLight}
        PreTag={Text}
        CodeTag={Text}
        highlighter="hljs"
        customStyle={{ backgroundColor: 'transparent', padding: 0 }}
      >
        {code}
      </SyntaxHighlighter>
    </View>
  );
}

const MDViewer = memo(function MDViewer({ markdown, maxWidth }) {
  const isDark = useColorScheme() !== 'light';

  const rules = {
    // ```lang\n ... \n```
    fence: (node) => {
      const language = (node.info || '').trim();
      const code = node.content ?? '';
      return <CodeBlock key={node.key} nodeKey={node.key} code={code} language={language} isDark={isDark} />;
    },
    // 4-space indented code
    code_block: (node) => {
      const code = node.content ?? '';
      return <CodeBlock key={node.key} nodeKey={node.key} code={code} language={undefined} isDark={isDark} />;
    },
  };

  return (
    <ScrollView
      contentContainerStyle={[styles.container, maxWidth ? { maxWidth, alignSelf: 'center' } : null]}
      showsVerticalScrollIndicator={false}
    >
      <Markdown
        markdownit={md}
        rules={rules}
        onLinkPress={(url) => url && Linking.openURL(url).catch(() => {})}
        style={{
          body: { color: isDark ? '#E6E6E6' : '#1f2328', fontSize: 16, lineHeight: 24 },
          heading1: { fontSize: 28, fontWeight: '700', marginTop: 14, marginBottom: 10 },
          heading2: { fontSize: 24, fontWeight: '700', marginTop: 12, marginBottom: 8 },
          heading3: { fontSize: 20, fontWeight: '700', marginTop: 10, marginBottom: 6 },
          paragraph: { marginBottom: 10 },
          strong: { fontWeight: '700' },
          em: { fontStyle: 'italic' },
          link: { color: '#58a6ff' },

          // inline `code`
          code_inline: {
            fontFamily: Platform.select({ ios: 'Menlo', android: 'monospace' }),
            backgroundColor: isDark ? '#1f2430' : '#f2f4f8',
            paddingHorizontal: 6, paddingVertical: 2, borderRadius: 6,
          },

          blockquote: {
            borderLeftWidth: 4,
            borderLeftColor: isDark ? '#444' : '#ddd',
            paddingLeft: 12,
            marginVertical: 10,
            color: isDark ? '#CFCFCF' : '#444',
          },

          // tables
          table: {
            borderWidth: 1,
            borderColor: isDark ? '#2e2e2e' : '#e1e4e8',
            borderRadius: 8,
            overflow: 'hidden',
            marginVertical: 8,
          },
          thead: { backgroundColor: isDark ? '#151b23' : '#f6f8fa' },
          th: {
            paddingHorizontal: 10, paddingVertical: 8, fontWeight: '700',
            borderRightWidth: 1, borderRightColor: isDark ? '#2e2e2e' : '#e1e4e8',
          },
          tr: { borderTopWidth: 1, borderTopColor: isDark ? '#2e2e2e' : '#e1e4e8' },
          td: {
            paddingHorizontal: 10, paddingVertical: 8,
            borderRightWidth: 1, borderRightColor: isDark ? '#2e2e2e' : '#e1e4e8',
          },

          list_item: { marginVertical: 4 },
          hr: { borderBottomColor: isDark ? '#2e2e2e' : '#e1e4e8', borderBottomWidth: StyleSheet.hairlineWidth, marginVertical: 12 },
          image: { borderRadius: 10, overflow: 'hidden' },
        }}
      >
        {markdown}
      </Markdown>
    </ScrollView>
  );
});

const styles = StyleSheet.create({
  container: { padding: 12 },
  codeBlockWrap: {
    borderRadius: 10,
    padding: 12,
    backgroundColor: 'rgba(127,127,127,0.08)',
    marginVertical: 8,
    fontSize: 30,
  },
  // fallback (no colors)
  fallbackCodeWrap: {
    borderRadius: 10,
    padding: 12,
    backgroundColor: 'rgba(127,127,127,0.08)',
    marginVertical: 8,
  },
  fallbackCodeText: {
    color: '#e6e6e6',
    fontFamily: Platform.select({ ios: 'Menlo', android: 'monospace' }),
    fontSize: 14,
    lineHeight: 20,
  },
});

export default MDViewer;
