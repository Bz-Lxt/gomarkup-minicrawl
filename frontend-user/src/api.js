async function parse(res) {
  const data = await res.json().catch(() => ({}))
  if (!res.ok) {
    const err = new Error(data?.error?.message || `HTTP ${res.status}`)
    throw err
  }
  return data
}

export const api = {
  stats: () => fetch('/api/stats').then(parse),
  timeseries: () => fetch('/api/stats/timeseries').then(parse),
  graph: () => fetch('/api/graph?limit=500').then(parse),
  keywords: () => fetch('/api/keywords/top?limit=18').then(parse),
  search: (q) => fetch(`/api/search?q=${encodeURIComponent(q)}&highlight=true`).then(parse),
}

export function adminOrigin() {
  return `${window.location.protocol}//${window.location.hostname}:18422`
}
