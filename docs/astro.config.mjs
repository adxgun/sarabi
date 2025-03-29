// @ts-check
import { defineConfig } from 'astro/config';
import starlight from '@astrojs/starlight';

import markdoc from '@astrojs/markdoc';

// https://astro.build/config
export default defineConfig({
    integrations: [starlight({
        title: 'Documentation',
        social: {
            github: 'https://github.com/adxgun/sarabi',
        },
        sidebar: [
            {
                label: 'Introduction',
                autogenerate: { directory: '/guides/introduction'},
            },
            {
                label: 'Getting Started',
                autogenerate: { directory: '/guides/getting-started'},
            },
            {
                label: 'Core Concepts',
                autogenerate: { directory: '/guides/core-concepts'},
            },
            {
                label: 'Reference',
                autogenerate: { directory: 'reference' },
            },
        ],
		}), markdoc()],
});