/*
Copyright (C) 2025 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/

import react from '@vitejs/plugin-react';
import { defineConfig, transformWithEsbuild } from 'vite';
import pkg from '@douyinfe/vite-plugin-semi';
import path from 'path';
import { codeInspectorPlugin } from 'code-inspector-plugin';
const { vitePluginSemi } = pkg;

// https://vitejs.dev/config/
export default defineConfig({
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'),
    },
  },
  plugins: [
    codeInspectorPlugin({
      bundler: 'vite',
    }),
    {
      name: 'treat-js-files-as-jsx',
      async transform(code, id) {
        if (!/src\/.*\.js$/.test(id)) {
          return null;
        }
        return transformWithEsbuild(code, id, {
          loader: 'jsx',
          jsx: 'automatic',
        });
      },
    },
    react(),
    vitePluginSemi({
      cssLayer: true,
    }),
  ],
  optimizeDeps: {
    force: true,
    esbuildOptions: {
      loader: {
        '.js': 'jsx',
        '.json': 'json',
      },
    },
  },
  build: {
    rollupOptions: {
      output: {
        manualChunks(id) {
          if (id.includes('node_modules')) {
            // Heavy dependencies: split into their own chunks
            if (id.includes('mermaid') || id.includes('cytoscape')) {
              return 'mermaid';
            }
            if (id.includes('@visactor') || id.includes('vchart')) {
              return 'vchart';
            }
            if (id.includes('katex')) {
              return 'katex';
            }
            // Icon libraries
            if (
              id.includes('react-icons') ||
              id.includes('lucide-react') ||
              id.includes('@lobehub/icons')
            ) {
              return 'icons';
            }
            // React + Semi UI (must be in the same chunk to avoid circular dependency)
            if (
              id.includes('react-dom') ||
              id.includes('react/') ||
              id.includes('react-router') ||
              id.includes('@remix-run') ||
              id.includes('@douyinfe/semi') ||
              id.includes('semi-icons')
            ) {
              return 'react-vendor';
            }
            // Markdown / rehype / remark ecosystem
            if (
              id.includes('react-markdown') ||
              id.includes('remark-') ||
              id.includes('rehype-') ||
              id.includes('unified') ||
              id.includes('unist-') ||
              id.includes('bail') ||
              id.includes('is-plain-') ||
              id.includes('trough') ||
              id.includes('vfile') ||
              id.includes('mdast') ||
              id.includes('hast') ||
              id.includes('lowlight') ||
              id.includes('highlight.js')
            ) {
              return 'markdown';
            }
            // Other small deps — no grouping to avoid large chunks
          }
        },
      },
    },
  },
  server: {
    host: '0.0.0.0',
    proxy: {
      '/api': {
        target: 'http://localhost:3000',
        changeOrigin: true,
      },
      '/mj': {
        target: 'http://localhost:3000',
        changeOrigin: true,
      },
    },
  },
});
