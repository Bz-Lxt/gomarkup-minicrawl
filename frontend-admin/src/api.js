const jsonHeaders = { 'Content-Type': 'application/json' }

async function parse(res) {
  const data = await res.json().catch(() => ({}))
  if (!res.ok) {
    const err = new Error(data?.error?.message || `HTTP ${res.status}`)
    err.code = data?.error?.code
    err.status = res.status
    throw err
  }
  return data
}

export const api = {
  health: () => fetch('/api/health').then(parse),
  stats: () => fetch('/api/stats').then(parse),
  timeseries: () => fetch('/api/stats/timeseries').then(parse),
  graph: () => fetch('/api/graph').then(parse),
  keywords: (limit = 24) => fetch(`/api/keywords/top?limit=${limit}`).then(parse),
  search: (q, highlight = true) =>
    fetch(`/api/search?q=${encodeURIComponent(q)}&highlight=${highlight}`).then(parse),
  tasks: () => fetch('/api/crawl/tasks').then(parse),
  task: (id) => fetch(`/api/crawl/tasks/${id}`).then(parse),
  createTask: (body) =>
    fetch('/api/crawl/tasks', { method: 'POST', headers: jsonHeaders, body: JSON.stringify(body) }).then(parse),
  stop: (id) => fetch(`/api/crawl/tasks/${id}/stop`, { method: 'POST' }).then(parse),
  pause: (id) => fetch(`/api/crawl/tasks/${id}/pause`, { method: 'POST' }).then(parse),
  resume: (id) => fetch(`/api/crawl/tasks/${id}/resume`, { method: 'POST' }).then(parse),
  clearIndex: () => fetch('/api/index/clear', { method: 'POST' }).then(parse),
}

export function visOrigin() {
  return `${window.location.protocol}//${window.location.hostname}:18423`
}
