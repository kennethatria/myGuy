<template>
  <!-- Shown when a listing has no photo: a shy cardboard box that peeks out
       and blinks. Drawn inline, so there is nothing to load or break. -->
  <figure class="no-photo">
    <svg viewBox="0 0 160 120" class="box" role="img" aria-label="A shy cardboard box peeking out">
      <ellipse cx="80" cy="108" rx="46" ry="6" class="shadow" />
      <g class="wobble">
        <rect x="38" y="52" width="84" height="54" rx="4" class="box-body" />
        <path d="M38 52 L80 64 L122 52" class="box-fold" />
        <g class="peek">
          <rect x="52" y="40" width="56" height="20" rx="10" class="dark" />
          <g class="eyes">
            <ellipse cx="70" cy="50" rx="5" ry="5.5" class="eye" />
            <ellipse cx="90" cy="50" rx="5" ry="5.5" class="eye" />
            <circle cx="71.5" cy="51" r="2.2" class="pupil" />
            <circle cx="91.5" cy="51" r="2.2" class="pupil" />
          </g>
        </g>
        <path d="M38 52 L26 36 L68 30 L80 52 Z" class="flap" />
        <path d="M122 52 L134 36 L92 30 L80 52 Z" class="flap flap-right" />
        <path d="M66 82 h28" class="tape" />
      </g>
    </svg>
    <figcaption>No photo yet. This one's a bit camera shy.</figcaption>
  </figure>
</template>

<style scoped>
.no-photo {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 0.5rem;
  margin: 0;
  padding: 1rem;
  text-align: center;
}

.box {
  width: min(220px, 70%);
  height: auto;
  overflow: visible;
}

figcaption {
  font-size: 0.9rem;
  color: var(--color-text-light, #6b7280);
}

.shadow { fill: rgba(15, 23, 42, 0.08); }
.box-body { fill: #d4a373; stroke: #a47148; stroke-width: 2; }
.box-fold { fill: none; stroke: #a47148; stroke-width: 2; stroke-linejoin: round; }
.flap { fill: #e0b585; stroke: #a47148; stroke-width: 2; stroke-linejoin: round; transform-origin: 38px 52px; }
.flap-right { transform-origin: 122px 52px; }
.tape { stroke: #f1d6a8; stroke-width: 7; stroke-linecap: round; }
.dark { fill: #3f2a1d; }
.eye { fill: #fff; }
.pupil { fill: #111827; }

/* Peek up, glance around, blink, duck back down */
.wobble { transform-origin: 80px 106px; animation: wobble 4s ease-in-out infinite; }
.peek { animation: peek 4s ease-in-out infinite; }
.flap { animation: lift-left 4s ease-in-out infinite; }
.flap-right { animation-name: lift-right; }
.pupil { animation: glance 4s ease-in-out infinite; }
.eyes { transform-origin: 80px 50px; animation: blink 4s infinite; }

@keyframes wobble {
  0%, 100% { transform: rotate(0); }
  10% { transform: rotate(-3deg); }
  20% { transform: rotate(3deg); }
  30% { transform: rotate(0); }
}
@keyframes peek {
  0%, 30%, 90%, 100% { transform: translateY(14px); }
  40%, 80% { transform: translateY(0); }
}
@keyframes lift-left {
  0%, 30%, 90%, 100% { transform: rotate(0); }
  40%, 80% { transform: rotate(-18deg); }
}
@keyframes lift-right {
  0%, 30%, 90%, 100% { transform: rotate(0); }
  40%, 80% { transform: rotate(18deg); }
}
@keyframes glance {
  0%, 45% { transform: translateX(0); }
  55% { transform: translateX(-3px); }
  65% { transform: translateX(3px); }
  75%, 100% { transform: translateX(0); }
}
@keyframes blink {
  0%, 58%, 62%, 100% { transform: scaleY(1); }
  60% { transform: scaleY(0.1); }
}

/* Still for people who'd rather not see motion: just the peeking box */
@media (prefers-reduced-motion: reduce) {
  .wobble, .peek, .flap, .pupil, .eyes { animation: none; }
  .peek { transform: none; }
}
</style>
