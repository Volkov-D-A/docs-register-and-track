import { chmod, readFile, readdir, writeFile } from 'node:fs/promises';
import { dirname, join, relative, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';

const scriptDirectory = dirname(fileURLToPath(import.meta.url));
const bindingsDirectory = resolve(scriptDirectory, '..', 'wailsjs');
const runtimeDirectory = join(bindingsDirectory, 'runtime');

async function listFiles(directory) {
    const entries = await readdir(directory, { withFileTypes: true });
    const nestedFiles = await Promise.all(entries.map((entry) => {
        const path = join(directory, entry.name);
        return entry.isDirectory() ? listFiles(path) : [path];
    }));
    return nestedFiles.flat();
}

for (const path of await listFiles(bindingsDirectory)) {
    const source = await readFile(path, 'utf8');
    const normalized = `${source
        .replace(/\r\n?/g, '\n')
        .split('\n')
        .map((line) => line.replace(/[\t ]+$/g, ''))
        .join('\n')
        .replace(/\n*$/g, '')}\n`;

    if (normalized !== source) {
        await writeFile(path, normalized);
    }
    if (!relative(runtimeDirectory, path).startsWith('..')) {
        await chmod(path, 0o644);
    }
}
