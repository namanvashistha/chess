/**
 * The pieces and the board, from the Calm Chess app's own art: the same
 * silhouettes (in a 100 by 100 box, standing on a shared plinth) that the
 * iPhone and Android apps draw.
 */

export const PLINTH =
  "M22 86.5C22 84 23.5 82.5 26 82.5H74C76.5 82.5 78 84 78 86.5V87.5C78 89.5 76.5 90.5 74.5 90.5H25.5C23.5 90.5 22 89.5 22 87.5Z";

export const ART: Record<string, {body: string; detail: string; eye?: [number, number, number]}> = {
  p: {
    body: "M33 82.5C35 74 40 67 43 61H57C60 67 65 74 67 82.5Z M37 61C37 58 41 56.5 50 56.5C59 56.5 63 58 63 61Z M50 27A14 14 0 1 1 49.99 27Z",
    detail: "M38 61H62",
  },
  r: {
    body: "M30 82.5C32 76 34 70 35 64V44H65V64C66 70 68 76 70 82.5Z M30 44V24H38V30H45V24H55V30H62V24H70V44Z",
    detail: "M35 44H65M34 64H66",
  },
  b: {
    body: "M32 82.5C35 76 40 71 42 66H58C60 71 65 76 68 82.5Z M38 66C38 63 42 61.5 50 61.5C58 61.5 62 63 62 66Z M50 20C61 28 66 37 66 46C66 54 59 60 50 60C41 60 34 54 34 46C34 37 39 28 50 20Z M50 11A5 5 0 1 1 49.99 11Z",
    detail: "M44 33L57 46M40 66H60",
  },
  n: {
    body: "M31 82.5C32 73 37 66 41 61C34 60 27 59 23 56C18 53 17.5 47.5 20.5 43.5L37 25C40.5 20 45 15 49.5 10.5C52 14 54 18 55.5 22C68 26 76 37 75.5 52C75 64 72.5 73 71.5 82.5Z",
    detail: "",
    eye: [40, 33, 2.6],
  },
  q: {
    body: "M30 82.5C33 74 38 66 40 58H60C62 66 67 74 70 82.5Z M36 58C36 55 41 53.5 50 53.5C59 53.5 64 55 64 58Z M35 53.5L27 27L38.5 40L44 21.5L50 38L56 21.5L61.5 40L73 27L65 53.5Z M27 20A4 4 0 1 1 26.99 20Z M44 14.5A4 4 0 1 1 43.99 14.5Z M56 14.5A4 4 0 1 1 55.99 14.5Z M73 20A4 4 0 1 1 72.99 20Z",
    detail: "M38 58H62M36 47H64",
  },
  k: {
    body: "M30 82.5C33 74 38 66 40 58H60C62 66 67 74 70 82.5Z M36 58C36 55 41 53.5 50 53.5C59 53.5 64 55 64 58Z M35 53.5C33 46 30 39 31 34C32.5 28 41 27 50 27C59 27 67.5 28 69 34C70 39 67 46 65 53.5Z M46.5 26V19H40.5V13H46.5V6.5H53.5V13H59.5V19H53.5V26Z",
    detail: "M38 58H62M33 40H67",
  },
};

/** A square's column (0 = a) and row (0 = the eighth rank, at the top). */
export const at = (sq: string) => ({x: sq.charCodeAt(0) - 97, y: 8 - Number(sq[1])});

/** The pieces of a FEN placement, each named after the square it starts on. */
export const pieces = (placement: string) => {
  const out: {id: string; code: string}[] = [];
  placement.split("/").forEach((row, r) => {
    let c = 0;
    for (const ch of row) {
      if (/\d/.test(ch)) c += Number(ch);
      else {
        out.push({id: `${String.fromCharCode(97 + c)}${8 - r}`, code: ch});
        c += 1;
      }
    }
  });
  return out;
};

/** The app's evening palette: the board at night, under one warm light. */
export const NIGHT = {
  light: "#C1BBAE",
  dark: "#76827A",
  honey: "#D9B556",
  ray: "#8DCBB0",
  danger: "#FF8A8A",
  great: "#6FD69A",
  ink: "#ECEEF0",
  mist: "#9AA2A8",
  bgTop: "#16191D",
  bgBottom: "#0B0C0E",
};
