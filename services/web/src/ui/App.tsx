import React, { useEffect, useMemo, useState } from 'react'
import { Line, LineChart, ResponsiveContainer, Tooltip, XAxis, YAxis } from 'recharts'

type PricePoint = { timestamp: number; price: number }

export function App() {
  const [points, setPoints] = useState<PricePoint[]>([])
  const [steamId, setSteamId] = useState<string | null>(null)

  useEffect(() => {
    fetch('/api/market/prices')
      .then((r) => r.json())
      .then(setPoints)
      .catch(() => setPoints([]))
  }, [])

  useEffect(() => {
    fetch('/api/auth/me', { credentials: 'include' })
      .then((r) => r.json())
      .then((d) => setSteamId(d.steam_id || null))
      .catch(() => setSteamId(null))
  }, [])

  const data = useMemo(
    () => points.map((p) => ({
      time: new Date(p.timestamp * 1000).toLocaleTimeString(),
      price: p.price,
    })),
    [points]
  )

  return (
    <div style={{ padding: 16, fontFamily: 'system-ui, sans-serif' }}>
      <header style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <h2>CS2 Trader</h2>
        <div>
          {steamId ? (
            <span>Logged in: {steamId}</span>
          ) : (
            <a href="/api/auth/steam/login"><button>Login with Steam</button></a>
          )}
        </div>
      </header>

      <section style={{ marginTop: 24 }}>
        <h3>Market Trend</h3>
        <div style={{ width: '100%', height: 300, border: '1px solid #ddd' }}>
          <ResponsiveContainer>
            <LineChart data={data}>
              <XAxis dataKey="time" hide={true} />
              <YAxis domain={[0, 'auto']} />
              <Tooltip />
              <Line type="monotone" dataKey="price" stroke="#2563eb" dot={false} />
            </LineChart>
          </ResponsiveContainer>
        </div>
      </section>
    </div>
  )
}

