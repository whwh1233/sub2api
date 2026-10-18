import { createPointerBeam } from './pointerBeam'

// Canvas renderer adapted from design/homepage-neural-flow/animation.html.
// Geometry and request timing are local to each mounted hero.
type Point = { x: number; y: number; tangent?: number }
type Path = Point[] & {
  curves?: Path[]
  samples?: Point[]
  distances?: number[]
  arcLength?: number
}
type Edge = {
  layer: number
  from: number
  to: number
  p: Path
  brightness: number
  width: number
  delay: number
  keepScore?: number
}
type Leg = {
  p: Path
  bundle: Path[]
  start: number
  duration: number
  hold: number
}
type FlowRequest = {
  sector: number
  angle: number
  port: Point
  contact: Point
  legs: Leg[]
  travel: number
  linger: number
  total: number
  arrival: number
  offset: number
  speed: number
  transferStart: number
  fillSpan: number
  lastArrivalCycle: number
  strength: number
}
type RingCell = {
  at: number
  span: number
  previousEnergy: number
  wasFilled: boolean
  generation: number
}
type BandSource = {
  index: number
  age: number
  span: number
  settled: boolean
}
type BandSources = BandSource[] & { byIndex: (BandSource | null)[] }

export function createNeuralFlow(canvas: HTMLCanvasElement) {
  const context = canvas.getContext('2d'),
    base = document.createElement('canvas'),
    baseContext = base.getContext('2d')
  if (!context || !baseContext) return null
  const ctx: CanvasRenderingContext2D = context,
    b: CanvasRenderingContext2D = baseContext
  const beam = createPointerBeam<Path>({
    path: (start, end) => windingPath(start, end, 719),
    fibres: (path) => fibers(path, 7, R * 0.025),
    flow: (bundle, progress, strength) =>
      bundleFlow(bundle, progress, 0.34, strength),
    position: pos,
    glow: node,
    trail: (p, bundle, age, duration, strength) =>
      persistentTrail(
        {
          travel: duration,
          linger: 6,
          strength,
          legs: [{ p, bundle, start: 0, duration, hold: 0 }],
        },
        age,
      ),
  })
  const sectorCount = 72,
    ringCells: (RingCell | null)[] = Array.from(
      { length: sectorCount },
      () => null,
    ),
    ringCycle: { fadeAt: number | null; acceptAfter: number } = {
      fadeAt: null,
      acceptAfter: 0,
    }
  let W = 0,
    H = 0,
    R = 0,
    cx = 0,
    cy = 0,
    dpr = 1,
    desktopLayout = false,
    clock = 0
  let layers: Point[][] = [],
    edges: Edge[] = [],
    route: Edge[] = [],
    requests: FlowRequest[] = []
  let seed = 35
  const random = () => {
    seed = (Math.imul(seed, 1664525) + 1013904223) >>> 0
    return seed / 4294967296
  }
  const clamp = (x: number) => Math.max(0, Math.min(1, x))
  const timingNoise = (i: number, salt: number) => {
    const x = Math.sin((i + 1) * 127.1 + salt * 311.7) * 43758.5453
    return x - Math.floor(x)
  }
  const landingSeed = Math.random() * 10000
  const arrivalGaps = Array.from(
    { length: sectorCount },
    (_, i) => 0.22 + 0.65 * timingNoise(i, 7),
  )
  const gapScale = 36 / arrivalGaps.reduce((sum, gap) => sum + gap, 0)
  for (let i = 0; i < sectorCount; i++) arrivalGaps[i] *= gapScale
  const arrivalTimes: number[] = []
  let nextArrival = 0.12
  for (const gap of arrivalGaps) {
    arrivalTimes.push(nextArrival)
    nextArrival += gap
  }
  function bezier(p: Path, u: number) {
    const v = 1 - u
    return {
      x:
        v * v * v * p[0].x +
        3 * v * v * u * p[1].x +
        3 * v * u * u * p[2].x +
        u * u * u * p[3].x,
      y:
        v * v * v * p[0].y +
        3 * v * v * u * p[1].y +
        3 * v * u * u * p[2].y +
        u * u * u * p[3].y,
    }
  }
  function preparePath(p: Path) {
    const samples = [p[0]],
      distances = [0]
    let length = 0
    for (const curve of p.curves!)
      for (let j = 1; j <= 24; j++) {
        const q = bezier(curve, j / 24),
          previous = samples[samples.length - 1]
        length += Math.hypot(q.x - previous.x, q.y - previous.y)
        samples.push(q)
        distances.push(length)
      }
    p.samples = samples
    p.distances = distances
    p.arcLength = length
    return p
  }
  function pos(p: Path, u: number) {
    if (!p.samples) return bezier(p, u)
    const target = clamp(u) * p.arcLength!
    let low = 0,
      high = p.distances!.length - 1
    while (low + 1 < high) {
      const mid = (low + high) >> 1
      if (p.distances![mid] < target) low = mid
      else high = mid
    }
    const a = p.samples[low],
      b = p.samples[high],
      t =
        (target - p.distances![low]) /
        (p.distances![high] - p.distances![low] || 1)
    return { x: a.x + (b.x - a.x) * t, y: a.y + (b.y - a.y) * t }
  }
  function trace(c: CanvasRenderingContext2D, p: Path) {
    c.moveTo(p[0].x, p[0].y)
    for (const curve of p.curves || [p])
      c.bezierCurveTo(
        curve[1].x,
        curve[1].y,
        curve[2].x,
        curve[2].y,
        curve[3].x,
        curve[3].y,
      )
  }
  function windingPath(start: Point, end: Point, key: number) {
    const dx = end.x - start.x,
      dy = end.y - start.y,
      length = Math.hypot(dx, dy) || 1,
      nx = -dy / length,
      ny = dx / length,
      sign = Math.sin(key * 1.71) > 0 ? 1 : -1
    const bound = (q: Point) => {
      const d = Math.hypot(q.x - cx, q.y - cy),
        s = Math.min(1, (R * 0.875) / (d || 1))
      return { x: cx + (q.x - cx) * s, y: cy + (q.y - cy) * s }
    }
    const amplitude = Math.min(
        length * 0.24,
        R * (0.09 + 0.055 * (0.5 + 0.5 * Math.sin(key * 2.31))),
      ),
      knots = [start]
    for (let j = 1; j <= 3; j++) {
      const t = j / 4,
        shape =
          Math.sin(key * 0.73) > 0.65
            ? Math.sin(2 * Math.PI * t) * 0.65
            : Math.sin(Math.PI * t) *
              (0.85 + 0.15 * Math.cos(Math.PI * t + key)),
        offset = amplitude * sign * shape,
        drift = Math.sin(key + j * 2.1) * 0.025
      knots.push(
        bound({
          x: start.x + dx * (t + drift) + nx * offset,
          y: start.y + dy * (t + drift) + ny * offset,
        }),
      )
    }
    knots.push(end)
    const curves: Path[] = []
    for (let j = 0; j < 4; j++) {
      const a = knots[Math.max(0, j - 1)],
        b = knots[j],
        c = knots[j + 1],
        d = knots[Math.min(4, j + 2)]
      curves.push([
        b,
        bound({ x: b.x + (c.x - a.x) / 6, y: b.y + (c.y - a.y) / 6 }),
        bound({ x: c.x - (d.x - b.x) / 6, y: c.y - (d.y - b.y) / 6 }),
        c,
      ])
    }
    const p: Path = [start, curves[0][1], curves[3][2], end]
    p.curves = curves
    return preparePath(p)
  }
  function packet(p: Path, u: number, tail = 0.09, alpha = 1, width = 1) {
    if (u < 0 || u > 1) return
    ctx.beginPath()
    for (let i = 0; i <= 28; i++) {
      const q = pos(p, clamp(u - tail + (i * tail) / 28))
      i ? ctx.lineTo(q.x, q.y) : ctx.moveTo(q.x, q.y)
    }
    if (width > 2) {
      ctx.strokeStyle = `rgba(255,107,53,${alpha * 0.12})`
      ctx.lineWidth = width * 3
      ctx.stroke()
    }
    ctx.strokeStyle = `rgba(255,138,82,${alpha})`
    ctx.lineWidth = width
    ctx.stroke()
    if (width < 3) {
      const q = pos(p, u)
      ctx.fillStyle = `rgba(255,204,170,${alpha})`
      ctx.beginPath()
      ctx.arc(q.x, q.y, width > 1.5 ? 2 : 1, 0, Math.PI * 2)
      ctx.fill()
    }
  }
  function node(point: Point, a = 0.8, size = 2.3) {
    const p = point
    const halo = ctx.createRadialGradient(p.x, p.y, 0, p.x, p.y, size * 5)
    halo.addColorStop(0, `rgba(255,107,53,${a * 0.5})`)
    halo.addColorStop(1, 'rgba(255,107,53,0)')
    ctx.fillStyle = halo
    ctx.fillRect(p.x - size * 5, p.y - size * 5, size * 10, size * 10)
    ctx.beginPath()
    ctx.arc(p.x, p.y, size, 0, Math.PI * 2)
    ctx.fillStyle = `rgba(255,204,170,${a})`
    ctx.fill()
  }
  function fibers(p: Path, count: number, spread: number) {
    const dx = p[3].x - p[0].x,
      dy = p[3].y - p[0].y,
      len = Math.hypot(dx, dy) || 1,
      nx = -dy / len,
      ny = dx / len
    return Array.from({ length: count }, (_, i) => {
      const offset = (i / (count - 1) - 0.5) * spread,
        shift = (q: Point) => ({ x: q.x + nx * offset, y: q.y + ny * offset })
      if (p.curves) {
        const curves = p.curves.map((c) => [
            c[0],
            shift(c[1]),
            shift(c[2]),
            c[3],
          ]),
          fiber: Path = [p[0], curves[0][1], curves[curves.length - 1][2], p[3]]
        fiber.curves = curves
        return preparePath(fiber)
      }
      return [p[0], shift(p[1]), shift(p[2]), p[3]]
    })
  }
  function bundleFlow(bundle: Path[], u: number, tail = 0.13, strength = 1) {
    if (u < 0 || u > 1) return
    ctx.save()
    ctx.globalCompositeOperation = 'lighter'
    if (bundle.length > 3)
      packet(bundle[Math.floor(bundle.length / 2)], u, tail, strength * 0.12, 5)
    for (let i = 0; i < bundle.length; i++) {
      const delay = (i % 5) * 0.012,
        local = (u - delay) / (1 - delay)
      packet(
        bundle[i],
        local,
        tail * (0.8 + (i % 3) * 0.1),
        strength * (0.55 + (i % 3) * 0.1),
        0.7,
      )
    }
    ctx.restore()
  }
  function smoothFade(u: number) {
    u = clamp(u)
    return 1 - u * u * (3 - 2 * u)
  }
  function persistentTrail(
    request: Pick<FlowRequest, 'travel' | 'linger' | 'legs' | 'strength'>,
    age: number,
  ) {
    const retirement =
      age > request.travel
        ? smoothFade((age - request.travel) / request.linger)
        : 1
    for (const leg of request.legs) {
      if (age <= leg.start) break
      const u = clamp((age - leg.start) / leg.duration)
      // The grey mesh is the actual passed route, revealed behind its travelling head.
      for (const index of [0, leg.bundle.length - 1]) {
        const p = leg.bundle[index]
        for (let j = 0; j < 64; j++) {
          const from = j / 64,
            to = Math.min(u, (j + 1) / 64)
          if (to <= from) break
          const elapsed = age - leg.start - (leg.duration * (from + to)) / 2,
            fade =
              (1 - smoothFade(elapsed / 0.35)) *
              smoothFade(elapsed / 18) *
              retirement
          if (fade < 0.002) continue
          const a = pos(p, from),
            b = pos(p, to),
            heat = Math.exp(-elapsed / 2.4)
          ctx.beginPath()
          ctx.moveTo(a.x, a.y)
          ctx.lineTo(b.x, b.y)
          ctx.strokeStyle = `rgba(${Math.round(150 + heat * 85)},${Math.round(164 - heat * 28)},${Math.round(182 - heat * 96)},${fade * request.strength * 0.28})`
          ctx.lineWidth = 0.7
          ctx.stroke()
        }
      }
    }
  }
  function cellState(cell: RingCell | null, time: number) {
    if (!cell || time < cell.at) return { coverage: 0, energy: 0 }
    const age = time - cell.at,
      reveal = 1 - smoothFade(age / 1.15)
    return {
      coverage: cell.wasFilled ? 1 : reveal,
      energy:
        age < 1.15
          ? cell.previousEnergy + (1 - cell.previousEnergy) * reveal
          : 1,
    }
  }
  function recordArrival(
    request: FlowRequest,
    time: number,
    generation: number,
  ) {
    if (ringCycle.fadeAt !== null || time < ringCycle.acceptAfter) return
    const previous = cellState(ringCells[request.sector], time)
    ringCells[request.sector] = {
      at: time,
      span: request.fillSpan,
      previousEnergy: previous.energy,
      wasFilled: previous.coverage > 0.99 && previous.energy > 0.015,
      generation,
    }
  }
  function requestFrame(request: FlowRequest, time: number) {
    const elapsed = time * request.speed + request.offset,
      cycle = Math.floor(elapsed / request.total),
      age = elapsed - cycle * request.total
    let leg = request.legs[request.legs.length - 1]
    for (const candidate of request.legs) {
      if (age < candidate.start + candidate.duration + candidate.hold) {
        leg = candidate
        break
      }
    }
    const u = clamp((age - leg.start) / leg.duration),
      head = pos(leg.p, u),
      terminal = request.legs[request.legs.length - 1]
    return {
      request,
      cycle,
      age,
      leg,
      u,
      head,
      arrived:
        leg === terminal &&
        u === 1 &&
        Math.hypot(head.x - request.contact.x, head.y - request.contact.y) <
          0.001,
    }
  }
  function updateArrivals(time: number) {
    const frames = requests.map((request) => requestFrame(request, time))
    for (const frame of frames) {
      const request = frame.request
      if (!frame.arrived || frame.cycle <= request.lastArrivalCycle) continue
      recordArrival(
        request,
        time - (frame.age - request.arrival) / request.speed,
        frame.cycle,
      )
      request.lastArrivalCycle = frame.cycle
    }
    return frames
  }
  function bandSources(time: number) {
    const sources = ringCells.flatMap((cell, index) =>
      cell && time >= cell.at
        ? [
            {
              index,
              age: time - cell.at,
              span: cell.span,
              settled: cell.wasFilled,
            },
          ]
        : [],
    ) as BandSources
    sources.byIndex = Array(sectorCount).fill(null)
    for (const source of sources) sources.byIndex[source.index] = source
    return sources
  }
  function bandEnergy(position: number, sources: BandSources, depth = 0) {
    const index =
      ((Math.floor(position) % sectorCount) + sectorCount) % sectorCount
    const source = sources.byIndex[index]
    if (!source) return 0
    // A bright travelling patch, followed by a fading afterimage, never a held fill.
    const local = position - Math.floor(position),
      delay = Math.abs(local - 0.35) * 0.18
    const passed = source.age - 0.1 - delay - depth * 4.2
    const rise = 1 - smoothFade((passed + 0.22) / 0.48),
      trail = Math.exp(-Math.max(0, passed) / 2.35)
    const angular = 0.75 + 0.25 * Math.cos((local - 0.5) * Math.PI)
    return rise * trail * angular * smoothFade((source.age - 7) / 4)
  }
  function ringOpacity(time: number) {
    return ringCycle.fadeAt === null
      ? 1
      : smoothFade((time - ringCycle.fadeAt) / 7)
  }
  function updateRingCycle(time: number) {
    if (ringCycle.fadeAt !== null) {
      if (time >= ringCycle.fadeAt + 7.7) {
        ringCells.fill(null)
        ringCycle.fadeAt = null
        ringCycle.acceptAfter = time
      }
      return
    }
    const sources = bandSources(time)
    if (sources.length < sectorCount) return
    // Completion follows actual arrival coverage around the whole band, never a fill timer.
    for (const source of sources) if (!source.settled && source.age < 8) return
    ringCycle.fadeAt = time + 1.4
  }
  function drawArrivalBand(time: number) {
    const sources = bandSources(time),
      count = 720,
      step = (Math.PI * 2) / count
    ctx.save()
    ctx.globalAlpha = ringOpacity(time)
    ctx.translate(cx, cy)
    const copper = ctx.createRadialGradient(0, 0, R * 1.035, 0, 0, R * 1.245)
    copper.addColorStop(0, 'rgba(153,57,28,0)')
    copper.addColorStop(0.12, '#b94b29')
    copper.addColorStop(0.38, '#ff6b35')
    copper.addColorStop(0.7, '#ff9d61')
    copper.addColorStop(0.88, '#d46739')
    copper.addColorStop(1, 'rgba(153,57,28,0)')
    const gold = ctx.createRadialGradient(0, 0, R * 1.035, 0, 0, R * 1.245)
    gold.addColorStop(0, 'rgba(255,138,82,0)')
    gold.addColorStop(0.22, '#ff8a52')
    gold.addColorStop(0.58, '#ffd3a0')
    gold.addColorStop(0.78, '#ffb477')
    gold.addColorStop(1, 'rgba(255,138,82,0)')
    // Small, soft pools of light travel outward through the data texture.
    ctx.save()
    ctx.beginPath()
    ctx.arc(0, 0, R * 1.245, 0, Math.PI * 2)
    ctx.arc(0, 0, R * 1.035, 0, Math.PI * 2, true)
    ctx.clip()
    ctx.globalCompositeOperation = 'lighter'
    for (const source of sources) {
      if (source.age < 0.1 || source.age > 7.5) continue
      const progress = clamp((source.age - 0.1) / 4.2),
        envelope =
          (1 - smoothFade(source.age / 0.45)) *
          smoothFade((source.age - 4.5) / 3)
      ctx.save()
      ctx.rotate(((source.index + 0.3) * Math.PI * 2) / sectorCount)
      ctx.translate(R * (1.035 + 0.21 * progress), 0)
      ctx.scale(0.8, 1.25)
      const light = ctx.createRadialGradient(0, 0, 0, 0, 0, R * 0.06)
      light.addColorStop(0, `rgba(255,155,76,${envelope * 0.68})`)
      light.addColorStop(0.34, `rgba(255,113,47,${envelope * 0.36})`)
      light.addColorStop(0.72, `rgba(255,107,53,${envelope * 0.09})`)
      light.addColorStop(1, 'rgba(255,107,53,0)')
      ctx.fillStyle = light
      ctx.fillRect(-R * 0.06, -R * 0.06, R * 0.12, R * 0.12)
      ctx.restore()
    }
    ctx.restore()
    // Fine, broken data marks carry the illumination instead of solid radial bars.
    for (let row = 0; row < count; row++) {
      const angle = (row + 0.5) * step,
        position = ((row + 0.5) * sectorCount) / count,
        energy = bandEnergy(position, sources)
      if (energy < 0.002) continue
      const source = sources.byIndex[Math.floor(position)]!,
        age = Math.max(0, source.age - (position % 1) * source.span),
        heat = Math.exp(-age / 5.5)
      const length =
          R *
          (0.204 + 0.005 * Math.sin(angle * 7) + 0.003 * Math.sin(angle * 19)),
        outerEnergy = bandEnergy(position, sources, 1)
      ctx.save()
      ctx.globalAlpha *= outerEnergy * 0.018
      ctx.beginPath()
      ctx.arc(0, 0, R * 1.14, angle - step * 0.55, angle + step * 0.55)
      ctx.strokeStyle = copper
      ctx.lineWidth = R * 0.21
      ctx.stroke()
      ctx.restore()
      const slices = 16
      ctx.save()
      ctx.rotate(angle)
      for (let part = 0; part < slices; part++) {
        const grain = timingNoise(row + part * 719, 23)
        if (grain < 0.2) continue
        const depth = (part + 0.5) / slices,
          reveal = bandEnergy(position, sources, depth)
        if (reveal < 0.002) continue
        const x = R * 1.035 + (length * part) / slices,
          width = (length / slices) * (0.48 + grain * 0.45),
          front = (source.age - 0.1) / 4.2
        const pulse =
            Math.exp(-(((depth - front) / 0.16) ** 2)) *
            smoothFade((source.age - 4.5) / 3),
          intensity = Math.min(1, reveal * (0.48 + grain * 0.32) + pulse * 0.8)
        ctx.save()
        ctx.globalAlpha *= intensity
        ctx.fillStyle = copper
        ctx.fillRect(x, -0.4, width, 0.8)
        ctx.globalAlpha *= Math.min(1, 0.4 + heat * 0.5 + pulse * 0.75)
        ctx.fillStyle = gold
        ctx.fillRect(x, -0.4, width, 0.8)
        ctx.restore()
      }
      ctx.restore()
      for (const radius of [0.91, 1.28]) {
        ctx.beginPath()
        ctx.arc(0, 0, R * radius, angle - step * 0.55, angle + step * 0.55)
        ctx.strokeStyle = `rgba(255,138,82,${(radius > 1 ? outerEnergy : energy) * (0.07 + heat * 0.08)})`
        ctx.lineWidth = 0.6
        ctx.stroke()
      }
    }
    ctx.restore()
    for (const source of sources) {
      const age = source.age
      if (age >= 0.9 || ringCycle.fadeAt !== null) continue
      const angle = (source.index * Math.PI * 2) / sectorCount
      const contact = {
          x: cx + Math.cos(angle) * R * 1.035,
          y: cy + Math.sin(angle) * R * 1.035,
        },
        impact = 0.84 + timingNoise(source.index, 11) * 0.28
      ctx.save()
      ctx.globalCompositeOperation = 'lighter'
      node(
        contact,
        smoothFade(age / 0.75) * impact,
        3.2 + 0.9 * smoothFade(age / 0.18),
      )
      ctx.restore()
    }
  }
  function drawTransitBand(frames: ReturnType<typeof requestFrame>[]) {
    const tickStep = (Math.PI * 2) / 480
    ctx.save()
    ctx.translate(cx, cy)
    ctx.globalCompositeOperation = 'lighter'
    for (const frame of frames) {
      const { request, age, leg } = frame,
        terminal = request.legs[request.legs.length - 1]
      if (
        leg !== terminal ||
        frame.u < request.transferStart ||
        (age - request.arrival) / request.speed > 1.8
      )
        continue
      const centreTick = Math.round(request.angle / tickStep)
      for (let offset = -5; offset <= 5; offset++) {
        const spread = Math.exp((-offset * offset) / 8),
          angle = (centreTick + offset) * tickStep
        ctx.save()
        ctx.rotate(angle)
        for (let cell = 0; cell < 8; cell++) {
          if (cell === 3) continue
          const radius = 0.947 + cell * 0.009
          // These dashes react only after the real moving head crosses their radius.
          const passageU =
            request.transferStart +
            ((radius - 0.91) / 0.125) * (1 - request.transferStart)
          const elapsed =
            (age - terminal.start - terminal.duration * passageU) /
              request.speed -
            Math.abs(offset) * 0.035
          if (elapsed <= 0 || elapsed > 1.8) continue
          const light =
              (1 - smoothFade(elapsed / 0.09)) *
              smoothFade(elapsed / 1.8) *
              spread,
            hot = Math.exp(-elapsed * 4)
          ctx.fillStyle = `rgba(255,${Math.round(127 + hot * 72)},${Math.round(62 + hot * 69)},${light * 0.72})`
          ctx.fillRect(R * radius, -0.45, R * 0.0065, 0.9)
        }
        ctx.restore()
      }
    }
    ctx.restore()
  }
  function build() {
    seed = 7823
    const counts = [1, 31, 47, 29, 1]
    const anchors = [
      [-0.84, -0.12],
      [-0.45, -0.34],
      [0.02, 0.35],
      [0.46, -0.26],
      [0.84, 0.16],
    ]
    const clusters = [
      [-0.31, -0.25, 0.26],
      [0.16, 0.35, 0.23],
      [0.43, -0.27, 0.19],
    ]
    function bounded(p: Point, limit = 0.875) {
      const dx = p.x - cx,
        dy = p.y - cy,
        d = Math.hypot(dx, dy),
        scale = d > R * limit ? (R * limit) / d : 1
      return { x: cx + dx * scale, y: cy + dy * scale }
    }
    // Processing groups are logical only: their positions overlap across an irregular disk.
    // Perimeter fibers, scattered relays and offset clusters deliberately have no shared columns.
    layers = counts.map((n, k) =>
      Array.from({ length: n }, (_, j) => {
        let x, y
        if (j === 0) {
          x = anchors[k][0]
          y = anchors[k][1]
        } else {
          const mode = random(),
            a = random() * Math.PI * 2
          if (mode < 0.37) {
            const rr = 0.65 + random() * 0.22
            x = Math.cos(a) * rr
            y = Math.sin(a) * rr
          } else if (mode < 0.67) {
            const rr = 0.8 * Math.sqrt(random())
            x = Math.cos(a) * rr
            y = Math.sin(a) * rr
          } else {
            const cluster = clusters[Math.floor(random() * clusters.length)],
              rr = cluster[2] * Math.sqrt(random())
            x = cluster[0] + Math.cos(a) * rr
            y = cluster[1] + Math.sin(a) * rr
          }
        }
        if (desktopLayout) {
          x = -0.48 + x * 0.48
          y *= 0.9
        }
        return {
          ...bounded({ x: cx + x * R, y: cy + y * R }),
          tangent: random() * Math.PI * 2,
        }
      }),
    )
    edges = []
    for (let k = 0; k < 4; k++)
      for (let i = 0; i < layers[k].length; i++)
        for (let j = 0; j < layers[k + 1].length; j++) {
          const start = layers[k][i],
            end = layers[k + 1][j],
            dx = end.x - start.x,
            dy = end.y - start.y
          if (Math.hypot(dx, dy) < R * 0.06 && (i !== 0 || j !== 0)) continue
          const sweep = R * (0.045 + random() * 0.22),
            tension = 0.18 + random() * 0.19
          const c1 = bounded({
            x: start.x + dx * tension + Math.cos(start.tangent!) * sweep,
            y: start.y + dy * tension + Math.sin(start.tangent!) * sweep,
          })
          const c2 = bounded({
            x: end.x - dx * tension - Math.cos(end.tangent!) * sweep,
            y: end.y - dy * tension - Math.sin(end.tangent!) * sweep,
          })
          edges.push({
            layer: k,
            from: i,
            to: j,
            p: [start, c1, c2, end],
            brightness: 0.035 + Math.pow(random(), 2) * 0.23,
            width: 0.36 + random() * 0.38,
            delay: random() * 0.25,
          })
        }
    route = [0, 1, 2, 3].map((k) =>
      edges.find((e) => e.layer === k && e.from === 0 && e.to === 0)!,
    )
    // Retain exactly 60% of the background fibers, favouring local bundles over
    // cross-disk clutter. Keep one incoming/outgoing edge per relay and the full spine.
    const primary = new Set(route),
      background = edges.filter((e) => !primary.has(e))
    const keepCount = Math.round(background.length * 0.6)
    const gaps = [
      [-0.4, 0.24, 0.17, 0.19],
      [0.24, -0.02, 0.19, 0.14],
    ]
    for (const e of background) {
      let crossings = 0
      for (let j = 1; j < 9; j++) {
        const q = pos(e.p, j / 9),
          x = (q.x - cx) / R,
          y = (q.y - cy) / R
        for (const [gx, gy, rx, ry] of gaps) {
          const d = ((x - gx) / rx) ** 2 + ((y - gy) / ry) ** 2
          if (d < 1) crossings += 1 - d
        }
      }
      const p = e.p[0],
        q = e.p[3],
        length = Math.hypot(p.x - q.x, p.y - q.y) / R
      let affinity = 0
      for (const [gx, gy, spread] of clusters) {
        const dp = Math.hypot((p.x - cx) / R - gx, (p.y - cy) / R - gy),
          dq = Math.hypot((q.x - cx) / R - gx, (q.y - cy) / R - gy)
        affinity = Math.max(
          affinity,
          Math.exp(-(dp * dp + dq * dq) / (spread * spread * 3)),
        )
      }
      const variation =
        ((Math.imul(e.from + 11, 73856093) ^
          Math.imul(e.to + 7, 19349663) ^
          Math.imul(e.layer + 3, 83492791)) >>>
          0) /
        4294967296
      e.keepScore =
        affinity * 0.85 +
        Math.max(0, 1 - length) * 0.3 +
        variation * 0.28 -
        crossings * 1.8
    }
    const retained = new Set(route)
    for (let k = 0; k < 4; k++) {
      const ranked = background
        .filter((e) => e.layer === k)
        .sort((a, b) => b.keepScore! - a.keepScore!)
      for (let i = 0; i < layers[k].length; i++) {
        if (![...retained].some((e) => e.layer === k && e.from === i)) {
          const e = ranked.find((e) => e.from === i)
          if (e) retained.add(e)
        }
      }
      for (let j = 0; j < layers[k + 1].length; j++) {
        if (![...retained].some((e) => e.layer === k && e.to === j)) {
          const e = ranked.find((e) => e.to === j)
          if (e) retained.add(e)
        }
      }
    }
    for (const e of background.sort((a, b) => b.keepScore! - a.keepScore!)) {
      if (retained.size >= keepCount + route.length) break
      retained.add(e)
    }
    edges = edges.filter((e) => retained.has(e))
    for (const e of edges) if (!primary.has(e)) e.brightness *= 0.57
    for (const e of edges)
      e.p = windingPath(e.p[0], e.p[3], e.layer * 137 + e.from * 13 + e.to * 7)
    function connection(start: Point, end: Point, bend: number) {
      return windingPath(start, end, (bend / R) * 31 + start.x / R + end.y / R)
    }
    // Independent, phase-staggered requests cover the entire disk. Each one follows
    // connected graph edges and returns to its own perimeter port.
    const visibleHalfAngle = Math.asin(
        Math.min(0.95, (H * 0.42) / (R * 1.035)),
      ),
      firstVisible = Math.ceil(
        ((Math.PI - visibleHalfAngle) * sectorCount) / (Math.PI * 2),
      ),
      lastVisible = Math.floor(
        ((Math.PI + visibleHalfAngle) * sectorCount) / (Math.PI * 2),
      )
    const targets = Array.from(
        {
          length: desktopLayout ? lastVisible - firstVisible + 1 : sectorCount,
        },
        (_, i) => (desktopLayout ? firstVisible + i : i),
      ),
      landingOrder: number[] = []
    let bag: number[] = []
    for (let i = 0; i < 72; i++) {
      if (!bag.length) bag = targets.slice()
      const recent = landingOrder.slice(
          -Math.min(6, Math.floor(targets.length * 0.6)),
        ),
        eligible = bag.filter((sector) => !recent.includes(sector)),
        choices = eligible.length ? eligible : bag,
        sector =
          choices[Math.floor(timingNoise(i, landingSeed) * choices.length)]
      landingOrder.push(sector)
      bag.splice(bag.indexOf(sector), 1)
    }
    requests = Array.from({ length: 72 }, (_, i) => {
      const sector = landingOrder[i],
        angle = (sector / sectorCount) * Math.PI * 2,
        port = {
          x: cx + Math.cos(angle) * R * 0.91,
          y: cy + Math.sin(angle) * R * 0.91,
        },
        contact = {
          x: cx + Math.cos(angle) * R * 1.035,
          y: cy + Math.sin(angle) * R * 1.035,
        }
      const from = i % layers[1].length,
        targetAngle = angle + 1.4,
        targetRadius = R * (0.15 + (0.65 * ((i * 17) % 71)) / 71)
      const target = {
        x: cx + Math.cos(targetAngle) * targetRadius,
        y: cy + Math.sin(targetAngle) * targetRadius,
      }
      const first = edges
        .filter((e) => e.layer === 1 && e.from === from)
        .sort(
          (a, b) =>
            Math.hypot(a.p[3].x - target.x, a.p[3].y - target.y) -
            Math.hypot(b.p[3].x - target.x, b.p[3].y - target.y),
        )[0]
      const endTarget = {
        x: cx + Math.cos(angle + 3.1) * R * 0.72,
        y: cy + Math.sin(angle + 3.1) * R * 0.72,
      }
      const second = edges
        .filter((e) => e.layer === 2 && e.from === first.to)
        .sort(
          (a, b) =>
            Math.hypot(a.p[3].x - endTarget.x, a.p[3].y - endTarget.y) -
            Math.hypot(b.p[3].x - endTarget.x, b.p[3].y - endTarget.y),
        )[0]
      const paths = [
        connection(port, first.p[0], R * 0.12 * Math.sin(angle)),
        first.p,
        second.p,
        connection(second.p[3], port, -R * 0.18 * Math.cos(angle)),
      ]
      // The transfer across the inner rings is part of the same visible path.
      // Its terminal point is on the filled band, not the inner perimeter.
      const returning = paths[3],
        radial = (f: number) => ({
          x: port.x + (contact.x - port.x) * f,
          y: port.y + (contact.y - port.y) * f,
        })
      returning.curves!.push([port, radial(1 / 3), radial(2 / 3), contact])
      returning[3] = contact
      preparePath(returning)
      const legs = paths.map((p, j) => {
        let length = 0,
          previous = p[0]
        for (let n = 1; n <= 12; n++) {
          const q = pos(p, n / 12)
          length += Math.hypot(q.x - previous.x, q.y - previous.y)
          previous = q
        }
        return {
          p,
          start: 0,
          bundle: fibers(p, i % 9 === 0 ? 7 : 3, R * 0.025),
          duration: 2.6 + (length / R) * 1.7,
          hold: j === 1 ? 1.6 : j === 2 ? 1.2 : 0.55,
        }
      })
      const speed = 0.78 + timingNoise(i, 3) * 0.3,
        moving = legs.reduce((sum, leg) => sum + leg.duration, 0),
        holds = legs.reduce((sum, leg) => sum + leg.hold, 0),
        pace = Math.min(1, (27 * speed - holds) / moving)
      let travel = 0
      for (const leg of legs) {
        leg.duration *= pace
        leg.start = travel
        travel += leg.duration + leg.hold
      }
      const total = sectorCount * 0.5 * speed,
        linger = total - travel,
        arrival = legs[3].start + legs[3].duration
      // Compensate for travel time; the visible contacts follow the shuffled landing order.
      const offset =
        (((arrival - arrivalTimes[i] * speed) % total) + total) % total
      const transferStart =
        returning.distances![(returning.curves!.length - 1) * 24] /
        returning.arcLength!
      return {
        sector,
        angle,
        port,
        contact,
        legs,
        travel,
        linger,
        total,
        arrival,
        offset,
        speed,
        transferStart,
        fillSpan: arrivalGaps[i],
        lastArrivalCycle: Math.floor(
          (clock * speed + offset - arrival) / total,
        ),
        strength: i % 9 === 0 ? 0.95 : 0.36 + (i % 5) * 0.045,
      }
    })
    b.setTransform(dpr, 0, 0, dpr, 0, 0)
    b.clearRect(0, 0, W, H)
    b.save()
    b.translate(cx, cy)
    for (const rr of [0.89, 0.91, 0.94, 1.02, 1.25, 1.28]) {
      b.beginPath()
      b.arc(0, 0, R * rr, 0, Math.PI * 2)
      b.strokeStyle = rr === 0.91 || rr === 1.28 ? '#ff6b352c' : '#ff6b3538'
      b.lineWidth = rr === 0.91 || rr === 1.28 ? 1 : 0.6
      b.stroke()
    }
    for (let j = 0; j < 480; j++) {
      const a = (j / 480) * Math.PI * 2
      b.save()
      b.rotate(a)
      b.fillStyle = j % 8 ? '#a484701c' : '#d09b742c'
      b.fillRect(R * 0.95, -0.3, R * 0.012, 0.6)
      b.fillRect(R * 0.982, -0.3, R * (j % 8 ? 0.018 : 0.024), 0.6)
      b.restore()
    }
    for (let j = 0; j < sectorCount; j++) {
      const a = ((j + 0.5) / sectorCount) * Math.PI * 2
      b.save()
      b.rotate(a)
      for (let row = 0; row < 16; row++) {
        for (let col = 0; col < 8; col++) {
          if (random() > 0.28) {
            b.fillStyle = `rgba(255,169,122,${0.1 + random() * 0.23})`
            b.fillRect(
              R * (1.045 + row * 0.012),
              (col - 3.5) * 2.1,
              Math.max(1.1, R * 0.0075),
              1,
            )
          }
        }
      }
      b.restore()
    }
    b.restore()
    drawScaffold(b)
  }

  function drawScaffold(c: CanvasRenderingContext2D) {
    // Cache the original scaffold with the ring; pointer input never changes it.
    for (const edge of edges) {
      if ((edge.from * 7 + edge.to * 3) % 9) continue
      c.beginPath()
      trace(c, edge.p)
      c.strokeStyle = `rgba(148,163,184,${edge.brightness * 0.16})`
      c.lineWidth = edge.width
      c.stroke()
    }
    for (const group of layers)
      for (const point of group) {
        const p = point
        c.beginPath()
        c.arc(p.x, p.y, 1, 0, Math.PI * 2)
        c.fillStyle = '#020617'
        c.fill()
        c.strokeStyle = '#64748b90'
        c.lineWidth = 0.55
        c.stroke()
      }
  }

  function draw() {
    if (!W) return
    ctx.setTransform(dpr, 0, 0, dpr, 0, 0)
    ctx.fillStyle = '#020617'
    ctx.fillRect(0, 0, W, H)
    const atmosphere = ctx.createRadialGradient(
      cx,
      cy,
      R * 0.15,
      cx,
      cy,
      R * 1.5,
    )
    atmosphere.addColorStop(0, 'rgba(15,23,42,.88)')
    atmosphere.addColorStop(0.64, 'rgba(15,23,42,.36)')
    atmosphere.addColorStop(1, 'rgba(2,6,23,0)')
    ctx.fillStyle = atmosphere
    ctx.fillRect(0, 0, W, H)
    ctx.drawImage(base, 0, 0, W, H)
    // Fill is arrival-driven. Only a completely lit band starts the slow recovery phase.
    updateRingCycle(clock)
    const frames = updateArrivals(clock)
    drawArrivalBand(clock)
    for (const frame of frames) persistentTrail(frame.request, frame.age)
    drawTransitBand(frames)
    for (const frame of frames) {
      const { request, age } = frame
      if (age >= request.travel) {
        const fade = smoothFade((age - request.travel) / request.linger)
        node(request.contact, fade * 0.18, 1.2)
        continue
      }
      for (let i = 0; i < request.legs.length; i++) {
        const leg = request.legs[i],
          local = age - leg.start
        if (local < 0) break
        if (local < leg.duration) {
          const u = frame.u
          bundleFlow(leg.bundle, u, 0.34, request.strength)
          if (i === request.legs.length - 1)
            node(frame.head, request.strength * 0.65, 1.8)
          else if (u > 0.9)
            node(leg.p[3], (u - 0.9) * 7 * request.strength, 1.7)
          if (i === 0 && u < 0.15) node(request.port, (1 - u / 0.15) * 0.7, 2.2)
          break
        }
        if (local < leg.duration + leg.hold) {
          const pulse =
            0.5 + 0.5 * Math.sin(((local - leg.duration) / leg.hold) * Math.PI)
          bundleFlow(leg.bundle, 1, 0.24, request.strength * 0.35)
          node(
            leg.p[3],
            (0.22 + pulse * 0.5) * request.strength,
            i === 1 ? 2.5 : 1.8,
          )
          break
        }
      }
    }
    beam.draw()
  }
  function resize() {
    const s = canvas.getBoundingClientRect()
    if (!s.width) return
    const wasDesktop = desktopLayout
    W = s.width
    H = s.height
    dpr = Math.min(devicePixelRatio || 1, 2)
    desktopLayout = W >= 768
    cx = W * (desktopLayout ? 1.015 : 0.72)
    cy = H * (desktopLayout ? 0.54 : 0.78)
    R = desktopLayout
      ? Math.max(W * 0.5, H * 0.56)
      : Math.min(W * 0.72, H * 0.38)
    beam.resize(W, H, cx, cy, R)
    if (wasDesktop !== desktopLayout) {
      ringCells.fill(null)
      ringCycle.fadeAt = null
      ringCycle.acceptAfter = clock
    }
    canvas.width = base.width = Math.round(W * dpr)
    canvas.height = base.height = Math.round(H * dpr)
    build()
    draw()
  }

  return {
    resize,
    render(time: number) {
      const dt = Math.min(0.05, Math.max(0, time - clock))
      clock = time
      beam.advance(dt)
      draw()
    },
    movePointer(x: number, y: number) {
      beam.move(x, y)
    },
    clearPointer() {
      beam.clear()
    },
    dispose() {
      canvas.width = canvas.height = base.width = base.height = 0
      layers = []
      edges = []
      route = []
      requests = []
      ringCells.fill(null)
      beam.dispose()
    },
  }
}
