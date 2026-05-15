// Shared photo-picker helper used by profile, group, and message flows.
//
// Why a single helper:
// 1. Safari's file picker misbehaves on a detached <input> — the element
//    must be in the DOM before `.click()` or the `change` event may never
//    fire. We attach it hidden, trigger, and remove it on completion.
// 2. Phone/gallery photos routinely run 3–10 MB. The server caps decoded
//    photos at 2 MiB, so sending the original bytes 1:1 fails every time
//    with a 400. We re-encode through a canvas (max 1280 px on the long
//    edge, JPEG quality 0.85) so realistic photos actually make it in.
// 3. Errors used to be swallowed — the picker closed and nothing happened.
//    This helper surfaces every failure (cancel, read error, decode error,
//    encode error) as a rejected promise so callers can show a message.
//
// Returns the raw base64 string (no "data:image/...;base64," prefix) on
// success, or rejects with an Error. Rejects with `PhotoCancelled` if the
// user dismissed the picker so callers can distinguish cancel from error.

const MAX_DIMENSION = 1280;
const JPEG_QUALITY = 0.85;

export class PhotoCancelled extends Error {
  constructor() {
    super("photo selection cancelled");
    this.name = "PhotoCancelled";
  }
}

export function pickAndEncodePhoto() {
  return new Promise((resolve, reject) => {
    const input = document.createElement("input");
    input.type = "file";
    input.accept = "image/*";
    input.style.position = "fixed";
    input.style.left = "-9999px";
    input.style.top = "-9999px";
    input.style.opacity = "0";
    document.body.appendChild(input);

    let settled = false;
    const cleanup = () => {
      // Defer removal so the change/cancel event has finished bubbling.
      setTimeout(() => {
        if (input.parentNode) input.parentNode.removeChild(input);
      }, 0);
    };
    const settle = (fn) => {
      if (settled) return;
      settled = true;
      cleanup();
      fn();
    };

    // Safari fires `cancel` when the picker is dismissed without a choice.
    // Other browsers don't fire anything, so we also catch the no-file case
    // in `change` and use a focus-based fallback (below) as a safety net.
    input.addEventListener("cancel", () => {
      settle(() => reject(new PhotoCancelled()));
    });

    input.addEventListener("change", () => {
      const file = input.files && input.files[0];
      if (!file) {
        settle(() => reject(new PhotoCancelled()));
        return;
      }
      encodeFile(file).then(
        (base64) => settle(() => resolve(base64)),
        (err) => settle(() => reject(err)),
      );
    });

    // Fallback for browsers that never emit `cancel`: when the window
    // regains focus after the picker closes, give the `change` handler
    // one tick to run — if it hasn't, treat it as a cancel.
    const onFocus = () => {
      setTimeout(() => {
        if (!settled && (!input.files || input.files.length === 0)) {
          settle(() => reject(new PhotoCancelled()));
        }
        window.removeEventListener("focus", onFocus);
      }, 300);
    };
    window.addEventListener("focus", onFocus);

    input.click();
  });
}

function encodeFile(file) {
  return new Promise((resolve, reject) => {
    const reader = new FileReader();
    reader.onerror = () => reject(new Error("could not read the selected file"));
    reader.onload = () => {
      const img = new Image();
      img.onerror = () => reject(new Error("selected file is not a valid image"));
      img.onload = () => {
        try {
          resolve(drawToBase64(img));
        } catch (err) {
          reject(err);
        }
      };
      img.src = String(reader.result);
    };
    reader.readAsDataURL(file);
  });
}

function drawToBase64(img) {
  const { width, height } = img;
  if (!width || !height) {
    throw new Error("image has zero dimensions");
  }

  // Preserve aspect ratio while clamping the long edge to MAX_DIMENSION.
  const scale = Math.min(1, MAX_DIMENSION / Math.max(width, height));
  const targetWidth = Math.max(1, Math.round(width * scale));
  const targetHeight = Math.max(1, Math.round(height * scale));

  const canvas = document.createElement("canvas");
  canvas.width = targetWidth;
  canvas.height = targetHeight;
  const ctx = canvas.getContext("2d");
  if (!ctx) throw new Error("canvas 2d context unavailable");
  ctx.drawImage(img, 0, 0, targetWidth, targetHeight);

  const dataUrl = canvas.toDataURL("image/jpeg", JPEG_QUALITY);
  const base64 = dataUrl.split(",")[1];
  if (!base64) throw new Error("canvas returned empty image data");
  return base64;
}
