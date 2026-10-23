export function decodeFrame(payload) {
  let frame;
  try {
    frame = JSON.parse(payload);
  } catch {
    throw new Error("二维码内容不是有效的文件传输帧");
  }

  if (!frame || typeof frame !== "object" || Array.isArray(frame)) {
    throw new Error("二维码帧格式无效");
  }
  if (frame.v !== 2) {
    throw new Error(`不支持的二维码协议版本：${String(frame.v)}`);
  }
  if (typeof frame.id !== "string" || !/^[0-9a-f]{32}$/.test(frame.id)) {
    throw new Error("传输编号无效");
  }

  if (frame.type === "manifest") {
    if (typeof frame.name !== "string" || frame.name.length === 0) {
      throw new Error("文件名无效");
    }
    if (!Number.isSafeInteger(frame.size) || frame.size < 0) {
      throw new Error("文件大小无效或超出浏览器可处理范围");
    }
    if (!Number.isSafeInteger(frame.chunkSize) || frame.chunkSize < 1 ||
        !Number.isSafeInteger(frame.total) || frame.total < 1) {
      throw new Error("二维码帧数或分片大小无效");
    }
    if (!/^[0-9a-f]{64}$/.test(frame.sha256)) {
      throw new Error("文件校验值无效");
    }
    const expectedTotal = Math.max(1, Math.ceil(frame.size / frame.chunkSize));
    if (frame.total !== expectedTotal) {
      throw new Error("二维码帧数与文件大小不匹配");
    }
    return {
      type: "manifest",
      manifest: {
        id: frame.id,
        name: frame.name,
        size: frame.size,
        chunkSize: frame.chunkSize,
        total: frame.total,
        sha256: frame.sha256,
      },
    };
  }

  if (frame.type === "data") {
    if (!Number.isSafeInteger(frame.index) || frame.index < 0) {
      throw new Error("二维码帧序号无效");
    }
    return { type: "data", frame, chunk: decodeBase64Url(frame.data) };
  }

  throw new Error(`不支持的二维码帧类型：${String(frame.type)}`);
}

function decodeBase64Url(value) {
  if (typeof value !== "string" || !/^[A-Za-z0-9_-]*$/.test(value) || value.length % 4 === 1) {
    throw new Error("二维码分片不是有效的 Base64URL 数据");
  }
  const base64 = value.replace(/-/g, "+").replace(/_/g, "/");
  const padded = base64 + "=".repeat((4 - (base64.length % 4)) % 4);
  const binary = atob(padded);
  return Uint8Array.from(binary, (character) => character.charCodeAt(0));
}

function sameTransfer(left, right) {
  return left.id === right.id &&
    left.name === right.name &&
    left.size === right.size &&
    left.chunkSize === right.chunkSize &&
    left.total === right.total &&
    left.sha256 === right.sha256;
}

function toHex(buffer) {
  return Array.from(new Uint8Array(buffer), (value) => value.toString(16).padStart(2, "0")).join("");
}

export class TransferAssembler {
  constructor(cryptoProvider = globalThis.crypto) {
    this.cryptoProvider = cryptoProvider;
    this.metadata = null;
    this.bytes = null;
    this.receivedChunks = null;
    this.received = 0;
  }

  get hasTransfer() {
    return this.metadata !== null;
  }

  add(payload) {
    const decoded = decodeFrame(payload);
    if (decoded.type === "manifest") {
      return this.addManifest(decoded.manifest);
    }
    if (!this.metadata) {
      return this.progress({ waitingForManifest: true });
    }

    const { frame, chunk } = decoded;
    if (frame.id !== this.metadata.id) {
      throw new Error("检测到不同文件的二维码，请重置后再扫描");
    }
    if (frame.index >= this.metadata.total) {
      throw new Error("二维码帧序号超出 metadata 声明范围");
    }
    const start = frame.index * this.metadata.chunkSize;
    const expectedLength = frame.index === this.metadata.total - 1
      ? this.metadata.size - start
      : this.metadata.chunkSize;
    if (chunk.length !== expectedLength) {
      throw new Error("二维码分片长度与 metadata 不匹配");
    }
    if (this.receivedChunks[frame.index]) {
      for (let offset = 0; offset < chunk.length; offset += 1) {
        if (this.bytes[start + offset] !== chunk[offset]) {
          throw new Error("同一帧收到不一致的数据");
        }
      }
      return this.progress();
    }

    this.bytes.set(chunk, start);
    this.receivedChunks[frame.index] = 1;
    this.received += 1;
    return this.progress({ added: true, addedIndex: frame.index });
  }

  addManifest(manifest) {
    if (this.metadata && !sameTransfer(this.metadata, manifest)) {
      throw new Error("检测到不同文件的二维码，请重置后再扫描");
    }
    if (this.metadata) {
      return this.progress();
    }

    try {
      this.bytes = new Uint8Array(manifest.size);
      this.receivedChunks = new Uint8Array(manifest.total);
    } catch (error) {
      if (error instanceof RangeError) {
        throw new Error("设备内存不足，无法接收此文件大小");
      }
      throw error;
    }
    this.metadata = manifest;
    return this.progress({ manifestAdded: true });
  }

  progress(extra = {}) {
    return {
      ...extra,
      received: this.received,
      total: this.metadata?.total ?? 0,
      complete: this.metadata !== null && this.received === this.metadata.total,
      name: this.metadata?.name ?? "",
      size: this.metadata?.size ?? 0,
      id: this.metadata?.id ?? "",
    };
  }

  async finalize() {
    if (!this.metadata || this.received !== this.metadata.total) {
      throw new Error("尚未收到全部二维码帧");
    }
    if (!this.cryptoProvider?.subtle) {
      throw new Error("此浏览器无法进行 SHA-256 校验，请使用 HTTPS 页面");
    }
    const digest = toHex(await this.cryptoProvider.subtle.digest("SHA-256", this.bytes));
    if (digest !== this.metadata.sha256) {
      throw new Error("文件校验失败，扫描到的数据可能不完整或已损坏");
    }
    return { name: this.metadata.name, bytes: this.bytes };
  }
}
