/**
 * Fotos das palavras da família, no IndexedDB do próprio aparelho
 * (`games_be_a_ba` → store `photos`, chave = id da palavra). Nunca são
 * enviadas. Antes de guardar, a foto é reduzida para no máximo 512 px.
 */

const DB_NAME = 'games_be_a_ba';
const STORE = 'photos';
export const PHOTO_MAX = 512;

function open(): Promise<IDBDatabase> {
  return new Promise((resolve, reject) => {
    const req = indexedDB.open(DB_NAME, 1);
    req.onupgradeneeded = () => req.result.createObjectStore(STORE);
    req.onsuccess = () => resolve(req.result);
    req.onerror = () => reject(req.error);
  });
}

async function run<T>(mode: IDBTransactionMode, fn: (s: IDBObjectStore) => IDBRequest<T>): Promise<T> {
  const db = await open();
  try {
    return await new Promise<T>((resolve, reject) => {
      const req = fn(db.transaction(STORE, mode).objectStore(STORE));
      req.onsuccess = () => resolve(req.result);
      req.onerror = () => reject(req.error);
    });
  } finally {
    db.close();
  }
}

export const savePhoto = (id: string, blob: Blob) => run('readwrite', (s) => s.put(blob, id));
export const deletePhoto = (id: string) => run('readwrite', (s) => s.delete(id));
export const loadPhoto = (id: string) => run<Blob | undefined>('readonly', (s) => s.get(id) as IDBRequest<Blob | undefined>);

/** Reduz a imagem escolhida para caber em `PHOTO_MAX` e devolve um JPEG. */
export async function shrinkPhoto(file: Blob): Promise<Blob> {
  const bitmap = await createImageBitmap(file);
  const scale = Math.min(1, PHOTO_MAX / Math.max(bitmap.width, bitmap.height));
  const w = Math.max(1, Math.round(bitmap.width * scale));
  const h = Math.max(1, Math.round(bitmap.height * scale));
  const canvas = document.createElement('canvas');
  canvas.width = w;
  canvas.height = h;
  canvas.getContext('2d')?.drawImage(bitmap, 0, 0, w, h);
  bitmap.close();
  return new Promise((resolve, reject) =>
    canvas.toBlob((b) => (b ? resolve(b) : reject(new Error('toBlob'))), 'image/jpeg', 0.85)
  );
}
