# QR file receiver

This standalone static site scans the QR sequence shown by MoYu Assistant and reconstructs the selected file in the mobile browser. File contents are processed locally; the receiver does not upload data or require a server-side API.

## Build and host

Use Node.js 18 or newer:

```sh
npm install
npm test
npm run build
```

Publish the contents of `dist/` to any static HTTPS host. The page must be opened from HTTPS (or `localhost`) because browsers restrict camera access and Web Crypto on insecure origins. A network connection is needed to load the hosted page; the actual file transfer is only through the displayed QR codes.

## Scanning

Open the hosted page on the phone, allow camera access, and start scanning before or during sender playback. The desktop sender displays a metadata QR and then cycles through all data frames, repeating the metadata QR each cycle. The receiver scans continuously with a short retry delay; it does not queue raw camera frames, since frames can become stale faster than they can be decoded. Missing frames are recovered when the sender repeats the sequence.

After camera permission is granted, use **选择摄像头** to switch among video inputs exposed by the browser. Some phones expose several rear lenses separately, while others expose only one logical rear camera; in that case the page cannot select an individual lens. **自动选择** requests the rear-facing camera when available.

The sender defaults to 10 frames per second and also offers 2, 1, 0.5, and 0.2 FPS modes. Slower settings hold each QR on screen longer for phones that need extra time to focus and decode; the receiver's matrix shows which chunks still need to be picked up. Use **Reset** before starting a different transfer. The receiver assembles chunks in memory and offers the file for download only after all chunks arrive and the reconstructed bytes pass SHA-256 verification.

## Payload format

The protocol uses UTF-8 JSON objects with version `v: 2`. The metadata QR has `type: "manifest"`:

| Field | Meaning |
| --- | --- |
| `id` | 16-byte random transfer ID encoded as 32 lowercase hex characters |
| `name` | Original file name (not a local path) |
| `size` | Original file size in bytes |
| `chunkSize` | Bytes per data chunk |
| `total` | Number of data frames; at least one, including for empty files |
| `sha256` | Lowercase SHA-256 of the complete original file |

Each data QR has `type: "data"` and contains the transfer `id`, a zero-based `index`, and that chunk's bytes encoded as unpadded Base64URL. The sender currently uses 700-byte chunks to reduce QR density and improve phone-camera readability. Data frames may arrive out of order; duplicates do not increment progress, and frames are ignored until the matching manifest is received. The scrollable status matrix is fixed at 20 columns; each row begins with its one-based first chunk number (1, 21, 41, ...), and a square lights when that chunk is accepted.

There is no application-level file-size cap. The sender keeps the source file in memory, and the receiver allocates an in-memory file buffer and a per-chunk status map after reading the manifest. Available device memory and the number of QR frames are practical limits; files around 10 MiB are the target production use case.
