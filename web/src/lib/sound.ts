// Short audio cues through Web Audio: a recorded ballpoint stroke when a set
// is logged, and a synthesized rubber-stamp thump and two-note chime when a
// workout is finished. On by default; the choice is kept per device. Phones
// in silent mode mute them anyway.

// Inlined into the bundle as a data URL, so it works offline without the
// service worker having to know about it.
import penStroke from "../assets/sounds/pen-stroke.wav?inline";

const KEY = "overlift:sound";

let ctx: AudioContext | null = null;
let pen: Promise<AudioBuffer | null> | null = null;

export function soundOn(): boolean {
  try {
    return localStorage.getItem(KEY) !== "off";
  } catch {
    return true;
  }
}

export function setSound(on: boolean): void {
  try {
    localStorage.setItem(KEY, on ? "on" : "off");
  } catch {
    // private mode: the choice lasts for this page only
  }
  if (on) unlock();
}

// Browsers only start audio from a user gesture, so call this synchronously
// in the click handler; cues may then play after an await.
export function unlock(): void {
  if (!soundOn()) return;
  try {
    ctx ??= new AudioContext();
    if (ctx.state === "suspended") void ctx.resume();
    pen ??= decode(ctx, penStroke);
  } catch {
    ctx = null; // no Web Audio: cues stay silent
  }
}

function ready(): AudioContext | null {
  return soundOn() && ctx && ctx.state !== "closed" ? ctx : null;
}

function decode(a: AudioContext, url: string): Promise<AudioBuffer | null> {
  return fetch(url)
    .then((r) => r.arrayBuffer())
    .then((b) => a.decodeAudioData(b))
    .catch(() => null); // undecodable: the cue stays silent
}

// A burst of filtered noise: the scratch of a pen, the slap of a stamp.
function noise(a: AudioContext, at: number, dur: number, type: BiquadFilterType, freq: number, peak: number) {
  const len = Math.ceil(a.sampleRate * dur);
  const buf = a.createBuffer(1, len, a.sampleRate);
  const data = buf.getChannelData(0);
  for (let i = 0; i < len; i++) data[i] = Math.random() * 2 - 1;
  const src = a.createBufferSource();
  src.buffer = buf;
  const filter = a.createBiquadFilter();
  filter.type = type;
  filter.frequency.value = freq;
  const gain = a.createGain();
  gain.gain.setValueAtTime(peak, at);
  gain.gain.exponentialRampToValueAtTime(0.0001, at + dur);
  src.connect(filter).connect(gain).connect(a.destination);
  src.start(at);
}

function tone(a: AudioContext, at: number, type: OscillatorType, from: number, to: number, dur: number, peak: number) {
  const osc = a.createOscillator();
  osc.type = type;
  osc.frequency.setValueAtTime(from, at);
  if (to !== from) osc.frequency.exponentialRampToValueAtTime(to, at + dur);
  const gain = a.createGain();
  gain.gain.setValueAtTime(0.0001, at);
  gain.gain.exponentialRampToValueAtTime(peak, at + 0.008);
  gain.gain.exponentialRampToValueAtTime(0.0001, at + dur);
  osc.connect(gain).connect(a.destination);
  osc.start(at);
  osc.stop(at + dur + 0.02);
}

// A set logged: the pen stroke, the same one every time. Usually decoded long
// before the first set is saved; if not, it plays as soon as it is.
export function cueLog(): void {
  const a = ready();
  if (!a || !pen) return;
  void pen.then((buf) => {
    if (!buf || !ready()) return;
    const src = a.createBufferSource();
    src.buffer = buf;
    const gain = a.createGain();
    gain.gain.value = 0.8;
    src.connect(gain).connect(a.destination);
    src.start();
  });
}

// The workout finished: the stamp lands `delayMs` from now, then a rising
// fifth rings out.
export function cueFinish(delayMs = 0): void {
  const a = ready();
  if (!a) return;
  const t = a.currentTime + delayMs / 1000;
  tone(a, t, "sine", 160, 48, 0.22, 0.7);
  noise(a, t, 0.07, "lowpass", 900, 0.5);
  tone(a, t + 0.16, "triangle", 659.25, 659.25, 0.5, 0.13);
  tone(a, t + 0.3, "triangle", 987.77, 987.77, 0.8, 0.13);
}
