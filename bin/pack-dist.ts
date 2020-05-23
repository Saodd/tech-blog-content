import { stat } from 'node:fs/promises';
import { dirname, isAbsolute, resolve, sep } from 'node:path';
import { fileURLToPath } from 'node:url';
import { create } from 'tar';

const repoRoot = resolve(dirname(fileURLToPath(import.meta.url)), '..');
const distDir = resolve(repoRoot, 'dist');
const outputArg = process.argv[2] ?? 'dist.tar.gz';
const outputFile = isAbsolute(outputArg)
  ? outputArg
  : resolve(repoRoot, outputArg);

async function ensureDistDirectory() {
  try {
    const stats = await stat(distDir);

    if (!stats.isDirectory()) {
      throw new Error(`${distDir} exists but is not a directory.`);
    }
  } catch (error) {
    if (error instanceof Error && 'code' in error && error.code === 'ENOENT') {
      throw new Error(`Cannot find dist directory: ${distDir}`);
    }

    throw error;
  }
}

function ensureOutputOutsideDist() {
  const normalizedDistDir = distDir.endsWith(sep) ? distDir : `${distDir}${sep}`;

  if (outputFile === distDir || outputFile.startsWith(normalizedDistDir)) {
    throw new Error('Output archive must be outside the dist directory.');
  }
}

async function packDist() {
  await ensureDistDirectory();
  ensureOutputOutsideDist();

  await create(
    {
      cwd: distDir,
      file: outputFile,
      gzip: true,
      portable: true,
    },
    ['.'],
  );

  // oxlint-disable-next-line no-console
  console.log(`Packed dist to ${outputFile}`);
}

await packDist();
