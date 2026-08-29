import { readFile, readdir, stat } from 'node:fs/promises';
import { dirname, relative, resolve, sep } from 'node:path';
import { fileURLToPath } from 'node:url';

const repositoryRoot = resolve(dirname(fileURLToPath(import.meta.url)), '..');
const excludedDirectories = new Set(['.git', '.test-build', 'dist', 'node_modules']);

async function listMarkdownFiles(directory) {
    const entries = await readdir(directory, { withFileTypes: true });
    const paths = await Promise.all(entries.map(async (entry) => {
        const path = resolve(directory, entry.name);
        if (entry.isDirectory()) {
            return excludedDirectories.has(entry.name) ? [] : listMarkdownFiles(path);
        }
        return entry.isFile() && entry.name.toLowerCase().endsWith('.md')
            ? [relative(repositoryRoot, path)]
            : [];
    }));
    return paths.flat().sort();
}

const markdownFiles = await listMarkdownFiles(repositoryRoot);

const inlineLinkPattern = /!?\[[^\]]*\]\(\s*(?:<([^>]+)>|([^\s)]+))(?:\s+["'][^"']*["'])?\s*\)/g;
const referenceLinkPattern = /^\s{0,3}\[[^\]]+\]:\s*(?:<([^>]+)>|(\S+))/;
const errors = [];

function isExternal(target) {
    return /^[a-z][a-z0-9+.-]*:/i.test(target) || target.startsWith('//');
}

function addTarget(targets, match) {
    const target = match[1] ?? match[2];
    if (target) targets.add(target);
}

for (const markdownPath of markdownFiles) {
    const absoluteMarkdownPath = resolve(repositoryRoot, markdownPath);
    const source = await readFile(absoluteMarkdownPath, 'utf8');
    const targets = new Set();
    let fenced = false;

    for (const line of source.split(/\r?\n/)) {
        if (/^\s*(```|~~~)/.test(line)) {
            fenced = !fenced;
            continue;
        }
        if (fenced) continue;

        inlineLinkPattern.lastIndex = 0;
        for (const match of line.matchAll(inlineLinkPattern)) addTarget(targets, match);
        const referenceMatch = line.match(referenceLinkPattern);
        if (referenceMatch) addTarget(targets, referenceMatch);
    }

    for (const rawTarget of targets) {
        if (isExternal(rawTarget) || rawTarget.startsWith('#')) continue;

        const pathPart = rawTarget.split(/[?#]/, 1)[0];
        if (!pathPart) continue;

        let decodedPath;
        try {
            decodedPath = decodeURIComponent(pathPart);
        } catch {
            errors.push(`${markdownPath}: invalid URL encoding in ${rawTarget}`);
            continue;
        }

        const targetPath = decodedPath.startsWith('/')
            ? resolve(repositoryRoot, decodedPath.slice(1))
            : resolve(dirname(absoluteMarkdownPath), decodedPath);
        const targetRelativePath = relative(repositoryRoot, targetPath);
        if (targetRelativePath === '..' || targetRelativePath.startsWith(`..${sep}`)) {
            errors.push(`${markdownPath}: link escapes repository: ${rawTarget}`);
            continue;
        }

        try {
            await stat(targetPath);
        } catch {
            errors.push(`${markdownPath}: missing target: ${rawTarget}`);
        }
    }
}

if (errors.length > 0) {
    console.error('Broken internal Markdown links:');
    for (const error of errors) console.error(`  - ${error}`);
    process.exitCode = 1;
} else {
    console.log(`Checked internal links in ${markdownFiles.length} Markdown files.`);
}
