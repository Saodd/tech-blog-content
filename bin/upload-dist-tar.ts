import { readFile, stat } from 'node:fs/promises';
import { basename, dirname, isAbsolute, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';

const BACKEND_ORIGIN = process.env.BACKEND_ORIGIN || 'https://lewinblog.com';
const TAR_ENDPOINT = '/api/blog/staff/tar';
const STAFF_TOKEN = process.env.BLOG_POST_TOKEN;

const repoRoot = resolve(dirname(fileURLToPath(import.meta.url)), '..');
const tarArg = process.argv[2] ?? 'dist.tar.gz';
const tarFile = isAbsolute(tarArg) ? tarArg : resolve(repoRoot, tarArg);

type UploadResponse = {
  code?: number;
  data?: unknown;
  ts?: number;
};

async function ensureTarFile() {
  try {
    const stats = await stat(tarFile);

    if (!stats.isFile()) {
      throw new Error(`${tarFile} exists but is not a file.`);
    }
  } catch (error) {
    if (error instanceof Error && 'code' in error && error.code === 'ENOENT') {
      throw new Error(`Cannot find tar archive: ${tarFile}`);
    }

    throw error;
  }
}

function getUploadUrl() {
  if (!BACKEND_ORIGIN) {
    throw new Error('BACKEND_ORIGIN is empty. Fill it before uploading.');
  }

  return new URL(TAR_ENDPOINT, BACKEND_ORIGIN).toString();
}

function parseUploadResponse(body: string): UploadResponse | null {
  try {
    const value = JSON.parse(body);

    if (value && typeof value === 'object') {
      return value as UploadResponse;
    }
  } catch {
    return null;
  }

  return null;
}

async function uploadDistTar() {
  await ensureTarFile();

  const tarBuffer = await readFile(tarFile);
  const formData = new FormData();
  formData.append(
    'file',
    new Blob([tarBuffer], { type: 'application/gzip' }),
    basename(tarFile),
  );

  const response = await fetch(getUploadUrl(), {
    body: formData,
    headers: {
      'X-STAFF-TOKEN': STAFF_TOKEN,
    },
    method: 'POST',
  });
  const body = await response.text();

  if (!response.ok) {
    throw new Error(
      `Upload failed with ${response.status} ${response.statusText}: ${body}`,
    );
  }

  const uploadResponse = parseUploadResponse(body);

  if (
    typeof uploadResponse?.code === 'number' &&
    uploadResponse.code !== 0
  ) {
    throw new Error(
      `Upload failed with response code ${uploadResponse.code}: ${body}`,
    );
  }

  // oxlint-disable-next-line no-console
  console.log(`Uploaded ${tarFile} to ${TAR_ENDPOINT}`);
}

await uploadDistTar();
