type Point = { x: number; y: number }
type BeamPath = { arcLength?: number }
type Renderer<P> = {
  path: (start: Point, end: Point) => P
  fibres: (path: P) => P[]
  flow: (bundle: P[], progress: number, strength: number) => void
  position: (path: P, progress: number) => Point
  glow: (point: Point, strength: number, size: number) => void
  trail: (
    path: P,
    bundle: P[],
    age: number,
    duration: number,
    strength: number,
  ) => void
}

// Share the original network's winding paths, fibre packets, arrival glow and
// grey passed-route trails. The mouse only steers this one additional request.
export function createPointerBeam<P extends BeamPath>(renderer: Renderer<P>) {
  let source: Point = { x: 0, y: 0 },
    target: Point = { x: 0, y: 0 },
    tip: Point = { x: 0, y: 0 }
  let builtTip: Point | null = null,
    path: P | null = null,
    bundle: P[] = []
  let radius = 1,
    active = false,
    opacity = 0,
    progress = 0,
    hold = 0,
    duration = 4
  let previous: { path: P; bundle: P[]; duration: number; age: number }[] = []

  function updatePath() {
    if (builtTip && Math.hypot(tip.x - builtTip.x, tip.y - builtTip.y) < 0.75)
      return
    path = renderer.path(source, tip)
    bundle = renderer.fibres(path)
    duration = 2.6 + ((path.arcLength || 1) / radius) * 1.7
    builtTip = { ...tip }
  }

  return {
    resize(_w: number, _h: number, cx: number, cy: number, r: number) {
      source = { x: cx - r * 0.91, y: cy }
      radius = r
      active = false
      opacity = 0
      path = null
      builtTip = null
      progress = 0
      hold = 0
      bundle = []
      previous = []
    },
    move(x: number, y: number) {
      target = { x, y }
      if (!active && opacity < 0.01) {
        tip = { x, y }
        builtTip = null
        progress = 0
        hold = 0
        previous = []
      }
      active = true
    },
    clear() {
      active = false
    },
    advance(dt: number) {
      const follow = 1 - Math.exp(-dt * 8)
      tip.x += (target.x - tip.x) * follow
      tip.y += (target.y - tip.y) * follow
      opacity +=
        ((active ? 1 : 0) - opacity) * (1 - Math.exp(-dt * (active ? 7 : 3)))
      for (const trail of previous) trail.age += dt
      previous = previous.filter((trail) => trail.age < trail.duration + 6)
      if (opacity < 0.005) return
      updatePath()
      if (progress < 1) progress = Math.min(1, progress + (dt * 0.9) / duration)
      else {
        hold += dt
        if (hold >= 0.55 && path) {
          previous.push({ path, bundle, duration, age: duration + hold })
          previous = previous.slice(-3)
          progress = 0
          hold = 0
        }
      }
    },
    draw() {
      if (
        opacity < 0.005 ||
        !path ||
        Math.hypot(tip.x - source.x, tip.y - source.y) < 12
      )
        return
      const strength = 0.95 * opacity
      for (const trail of previous)
        renderer.trail(
          trail.path,
          trail.bundle,
          trail.age,
          trail.duration,
          strength,
        )
      renderer.trail(
        path,
        bundle,
        progress * duration + hold,
        duration,
        strength,
      )
      if (progress < 1) {
        renderer.flow(bundle, progress, strength)
        renderer.glow(renderer.position(path, progress), strength * 0.65, 1.8)
        if (progress < 0.15)
          renderer.glow(source, (1 - progress / 0.15) * strength * 0.7, 2.2)
      } else {
        const fade = Math.max(0, 1 - hold / 0.55)
        renderer.flow(bundle, 1, strength * 0.35 * fade)
        renderer.glow(tip, strength * fade * 0.7, 1.8)
      }
    },
    dispose() {
      active = false
      opacity = 0
      path = null
      bundle = []
      previous = []
    },
  }
}
