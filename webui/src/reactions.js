export const supportedReactions = [
  { value: "👍", name: "Thumbs up" },
  { value: "❤️", name: "Heart" },
  { value: "👌", name: "OK" },
  { value: "😂", name: "Laugh" },
  { value: "😮", name: "Wow" },
  { value: "🙏", name: "Pray" },
  { value: "🔥", name: "Fire" },
  { value: "🎉", name: "Party" },
  { value: "😢", name: "Sad" },
  { value: "👀", name: "Eyes" },
  { value: "👏", name: "Clap" },
  { value: "⭐", name: "Star" },
];

const aliases = new Map([
  [":thumbsup:", "👍"],
  ["+1", "👍"],
  [":heart:", "❤️"],
  ["<3", "❤️"],
  ["❤", "❤️"],
  [":ok:", "👌"],
  ["OK", "👌"],
  ["ok", "👌"],
  [":laugh:", "😂"],
  [":wow:", "😮"],
  [":pray:", "🙏"],
  [":fire:", "🔥"],
  [":party:", "🎉"],
  [":sad:", "😢"],
  [":eyes:", "👀"],
  [":clap:", "👏"],
  [":star:", "⭐"],
]);

export function normalizeReaction(value) {
  const raw = String(value || "").trim().normalize("NFC");
  const withoutVariation = raw.replace(/[\uFE0E\uFE0F]/g, "");
  return aliases.get(raw) || aliases.get(withoutVariation) || raw;
}
