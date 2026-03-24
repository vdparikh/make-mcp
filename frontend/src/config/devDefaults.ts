/**
 * Example values for UI templates. Set VITE_EXAMPLE_DB_HOST (see env.example.sh); never hardcode hostnames in source.
 */
export function examplePostgresConnectionString(): string {
  const host = import.meta.env.VITE_EXAMPLE_DB_HOST?.trim();
  if (!host) {
    return 'postgres://user:password@CONFIGURE_VITE_EXAMPLE_DB_HOST:5432/app_db?sslmode=disable';
  }
  return `postgres://user:password@${host}:5432/app_db?sslmode=disable`;
}
