export function isURL(str: string): boolean {
  return str.startsWith("http://") || str.startsWith("https://");
}
