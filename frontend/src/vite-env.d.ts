/// <reference types="vite/client" />

interface ImportMetaEnv {
  readonly VITE_EXAMPLE_DB_HOST?: string;
}

interface ImportMeta {
  readonly env: ImportMetaEnv;
}
