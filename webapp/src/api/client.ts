// Shared fetch helper for every backend call: every non-ok response's body
// is the error message (plain text, see e.g. AddPost/EditPost's
// http.Error calls) — surfaced as a thrown Error, falling back to a generic
// message when the body is empty (some handlers, like a failed /login,
// write a bare status code with no body at all).
async function errorMessage(res: Response, fallback: string): Promise<string> {
  const text = await res.text().catch(() => '')
  return text || fallback
}

export const jsonHeaders: HeadersInit = { 'Content-Type': 'application/json' }

// For endpoints that respond with a JSON body on success. Paginated list
// endpoints also set X-Total-Count; endpoints that don't (e.g. search) leave
// `total` null rather than 0, so callers can tell "no total reported" apart
// from "zero results."
export async function requestJson<T>(path: string, init?: RequestInit): Promise<{ data: T; total: number | null }> {
  const res = await fetch(path, init)
  if (!res.ok) {
    throw new Error(await errorMessage(res, `Request failed (${res.status})`))
  }
  const totalHeader = res.headers.get('X-Total-Count')
  const total = totalHeader === null ? null : parseInt(totalHeader, 10)
  const data = (await res.json()) as T
  return { data, total }
}

// For endpoints whose success response isn't JSON (e.g. /register replies
// with a plain-text "Registration successful.") or whose body the caller
// doesn't need.
export async function requestVoid(path: string, init?: RequestInit): Promise<void> {
  const res = await fetch(path, init)
  if (!res.ok) {
    throw new Error(await errorMessage(res, `Request failed (${res.status})`))
  }
}
