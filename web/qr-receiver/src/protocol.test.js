import assert from "node:assert/strict";
import test from "node:test";
import { webcrypto } from "node:crypto";

import { TransferAssembler, decodeFrame } from "./protocol.js";

const helloManifest =
  '{"v":2,"type":"manifest","id":"0123456789abcdef0123456789abcdef","name":"hello.txt","size":11,"chunkSize":700,"total":1,"sha256":"b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9"}';
const helloData =
  '{"v":2,"type":"data","id":"0123456789abcdef0123456789abcdef","index":0,"data":"aGVsbG8gd29ybGQ"}';
const CHUNK_SIZE = 700;

function toHex(bytes) {
  return Array.from(new Uint8Array(bytes), (value) => value.toString(16).padStart(2, "0")).join("");
}

function payloadFor({ id, index, data }) {
  return JSON.stringify({
    v: 2,
    type: "data",
    id,
    index,
    data: Buffer.from(data).toString("base64url"),
  });
}

async function makeTwoFrameTransfer() {
  const bytes = Uint8Array.from(
    { length: CHUNK_SIZE + 3 },
    (_, index) => index % 251,
  );
  const sha256 = toHex(await webcrypto.subtle.digest("SHA-256", bytes));
  return {
    bytes,
    manifest: JSON.stringify({
      v: 2,
      type: "manifest",
      id: "0123456789abcdef0123456789abcdef",
      name: "binary.dat",
      size: bytes.length,
      chunkSize: CHUNK_SIZE,
      total: 2,
      sha256,
    }),
    frames: [
      payloadFor({
        id: "0123456789abcdef0123456789abcdef",
        index: 0,
        data: bytes.slice(0, CHUNK_SIZE),
      }),
      payloadFor({
        id: "0123456789abcdef0123456789abcdef",
        index: 1,
        data: bytes.slice(CHUNK_SIZE),
      }),
    ],
  };
}

test("decodes the shared hello-world manifest and data fixtures", async () => {
  const decodedManifest = decodeFrame(helloManifest);
  assert.equal(decodedManifest.manifest.name, "hello.txt");
  assert.equal(new TextDecoder().decode(decodeFrame(helloData).chunk), "hello world");

  const assembler = new TransferAssembler(webcrypto);
  assembler.add(helloManifest);
  assembler.add(helloData);
  const file = await assembler.finalize();
  assert.equal(file.name, "hello.txt");
  assert.equal(new TextDecoder().decode(file.bytes), "hello world");
});

test("assembles frames scanned out of order and ignores identical duplicates", async () => {
  const { bytes, manifest, frames } = await makeTwoFrameTransfer();
  const assembler = new TransferAssembler(webcrypto);

  assert.equal(assembler.add(frames[1]).waitingForManifest, true);
  assert.equal(assembler.add(manifest).manifestAdded, true);
  assert.equal(assembler.add(frames[1]).received, 1);
  assert.equal(assembler.add(frames[1]).added, undefined);
  assert.equal(assembler.add(frames[0]).complete, true);
  const file = await assembler.finalize();
  assert.equal(file.name, "binary.dat");
  assert.deepEqual(file.bytes, bytes);
});

test("rejects a completed transfer with an incorrect checksum", async () => {
  const { frames, manifest } = await makeTwoFrameTransfer();
  const corruptedManifest = JSON.parse(manifest);
  corruptedManifest.sha256 = "0".repeat(64);
  const assembler = new TransferAssembler(webcrypto);
  assembler.add(JSON.stringify(corruptedManifest));
  assembler.add(frames[0]);
  assembler.add(frames[1]);
  await assert.rejects(assembler.finalize(), /校验失败/);
});

test("rejects malformed or inconsistent manifest metadata", () => {
  assert.throws(() => decodeFrame("not-json"), /不是有效的文件传输帧/);
  const manifest = JSON.parse(helloManifest);
  manifest.total = 2;
  assert.throws(() => decodeFrame(JSON.stringify(manifest)), /帧数与文件大小不匹配/);
});

test("rejects data frames from a different transfer while assembling", async () => {
  const { manifest, frames } = await makeTwoFrameTransfer();
  const assembler = new TransferAssembler(webcrypto);
  assembler.add(manifest);
  const other = JSON.parse(frames[0]);
  other.id = "ffffffffffffffffffffffffffffffff";
  assert.throws(() => assembler.add(JSON.stringify(other)), /不同文件/);
});

test("reassembles the expected 10 MiB transfer entirely in memory", async () => {
  const bytes = Uint8Array.from(
    { length: 10 * 1024 * 1024 },
    (_, index) => index % 251,
  );
  const sha256 = toHex(await webcrypto.subtle.digest("SHA-256", bytes));
  const total = Math.ceil(bytes.length / CHUNK_SIZE);
  const assembler = new TransferAssembler(webcrypto);
  assembler.add(JSON.stringify({
    v: 2,
    type: "manifest",
    id: "abcdef0123456789abcdef0123456789",
    name: "ten-mebibytes.bin",
    size: bytes.length,
    chunkSize: CHUNK_SIZE,
    total,
    sha256,
  }));
  for (let index = 0; index < total; index += 1) {
    const start = index * CHUNK_SIZE;
    assembler.add(payloadFor({
      id: "abcdef0123456789abcdef0123456789",
      index,
      data: bytes.subarray(start, Math.min(bytes.length, start + CHUNK_SIZE)),
    }));
  }
  assert.equal(assembler.progress().complete, true);
  const file = await assembler.finalize();
  assert.deepEqual(file.bytes, bytes);
});
