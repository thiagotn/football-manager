// Site 100% estático (adapter-static): tudo é pré-renderizado.
// `trailingSlash: 'always'` gera `/enche-ai/index.html`, casando com o
// `scope`/`start_url` `/enche-ai/` do manifesto do jogo.
export const prerender = true;
export const trailingSlash = 'always';
