import { BrowserQRCodeReader } from "@zxing/browser";
import { TransferAssembler } from "./protocol.js";
import "./style.css";

const camera = document.querySelector("#camera");
const cameraSelect = document.querySelector("#camera-select");
const startButton = document.querySelector("#start-scan");
const stopButton = document.querySelector("#stop-scan");
const resetButton = document.querySelector("#reset-transfer");
const scanStatus = document.querySelector("#scan-status");
const transferStatus = document.querySelector("#transfer-status");
const progress = document.querySelector("#transfer-progress");
const matrixContainer = document.querySelector("#matrix-container");
const chunkMatrix = document.querySelector("#chunk-matrix");
const matrixCaption = document.querySelector("#matrix-caption");
const downloadLink = document.querySelector("#download-file");

let assembler = new TransferAssembler();
let controls = null;
let scanning = false;
let completing = false;
let downloadURL = null;
let lastScanError = "";
let matrixTransferId = "";
let selectedCameraId = "";
let matrixReceivedColor = "#1f883d";

const matrixColumns = 20;
const matrixCellPitch = 10;
const matrixCellSize = 8;
const matrixPadding = 4;
const matrixLabelWidth = 44;

function setScanStatus(message, state = "") {
  scanStatus.textContent = message;
  scanStatus.dataset.state = state;
}

function stopCamera() {
  scanning = false;
  controls?.stop();
  controls = null;
  for (const track of camera.srcObject?.getTracks() ?? []) {
    track.stop();
  }
  camera.srcObject = null;
  startButton.disabled = false;
  stopButton.disabled = true;
}

async function refreshCameraOptions() {
  const devices = await BrowserQRCodeReader.listVideoInputDevices();
  const activeDeviceId =
    camera.srcObject?.getVideoTracks()[0]?.getSettings().deviceId ?? "";
  const autoOption = document.createElement("option");
  autoOption.value = "";
  autoOption.textContent = "自动选择（优先后置）";
  cameraSelect.replaceChildren(autoOption);

  for (const [index, device] of devices.entries()) {
    const option = document.createElement("option");
    option.value = device.deviceId;
    option.textContent = device.label || `摄像头 ${index + 1}`;
    cameraSelect.append(option);
  }

  const selectedDeviceExists = devices.some(
    (device) => device.deviceId === selectedCameraId,
  );
  if (!selectedDeviceExists) {
    selectedCameraId = devices.some(
      (device) => device.deviceId === activeDeviceId,
    )
      ? activeDeviceId
      : "";
  }
  cameraSelect.value = selectedCameraId;
  cameraSelect.disabled = devices.length < 2;
}

function renderProgress(state) {
  progress.value = state.total === 0 ? 0 : state.received / state.total;
  if (!state.total) {
    transferStatus.textContent = "尚未接收到二维码。";
    matrixContainer.hidden = true;
    matrixCaption.hidden = true;
    resetButton.disabled = true;
    return;
  }
  const megabytes = (state.size / (1024 * 1024)).toFixed(2);
  transferStatus.textContent =
    `${state.name} · ${state.received} / ${state.total} 分片 · ${megabytes} MiB`;
  if (state.manifestAdded || matrixTransferId !== state.id) {
    initializeMatrix(state);
  }
  if (Number.isInteger(state.addedIndex)) {
    lightMatrixCell(state.addedIndex);
  }
  matrixContainer.hidden = false;
  matrixCaption.hidden = false;
  resetButton.disabled = false;
}

function initializeMatrix(state) {
  matrixTransferId = state.id;
  const rows = Math.ceil(state.total / matrixColumns);
  chunkMatrix.width =
    matrixPadding * 2 + matrixLabelWidth + matrixColumns * matrixCellPitch;
  chunkMatrix.height = matrixPadding * 2 + rows * matrixCellPitch;

  const context = chunkMatrix.getContext("2d");
  const colors = getComputedStyle(document.documentElement);
  const emptyColor = colors.getPropertyValue("--matrix-empty-color").trim();
  const labelColor = colors.getPropertyValue("--matrix-label-color").trim();
  matrixReceivedColor =
    colors.getPropertyValue("--matrix-received-color").trim() || "#1f883d";
  context.font = "9px system-ui, sans-serif";
  context.textBaseline = "middle";
  for (let index = 0; index < state.total; index += 1) {
    const row = Math.floor(index / matrixColumns);
    const column = index % matrixColumns;
    const x = matrixPadding + matrixLabelWidth + column * matrixCellPitch;
    const y = matrixPadding + row * matrixCellPitch;
    if (column === 0) {
      context.fillStyle = labelColor;
      context.fillText(String(index + 1), matrixPadding, y + matrixCellPitch / 2);
    }
    context.fillStyle = emptyColor;
    context.fillRect(x, y, matrixCellSize, matrixCellSize);
  }
  chunkMatrix.setAttribute(
    "aria-label",
    `已接收 ${state.received} / ${state.total} 个文件分片`,
  );
}

function lightMatrixCell(index) {
  const context = chunkMatrix.getContext("2d");
  const column = index % matrixColumns;
  const row = Math.floor(index / matrixColumns);
  const x = matrixPadding + matrixLabelWidth + column * matrixCellPitch;
  const y = matrixPadding + row * matrixCellPitch;
  context.fillStyle = matrixReceivedColor;
  context.fillRect(x, y, matrixCellSize, matrixCellSize);
  chunkMatrix.setAttribute(
    "aria-label",
    `已接收 ${assembler.progress().received} / ${assembler.progress().total} 个文件分片`,
  );
}

function safeFileName(name) {
  const sanitized = name.replace(/[\\/:*?"<>|\u0000-\u001f]/g, "_").trim();
  return sanitized || "received-file";
}

async function finishTransfer() {
  completing = true;
  resetButton.disabled = true;
  stopCamera();
  setScanStatus("正在校验并还原文件…");
  try {
    const result = await assembler.finalize();
    const blob = new Blob([result.bytes], { type: "application/octet-stream" });
    if (downloadURL) {
      URL.revokeObjectURL(downloadURL);
    }
    downloadURL = URL.createObjectURL(blob);
    downloadLink.href = downloadURL;
    downloadLink.download = safeFileName(result.name);
    downloadLink.hidden = false;
    setScanStatus("文件已校验完成，可以下载。", "success");
  } catch (error) {
    setScanStatus(`还原失败：${error.message}`, "error");
  } finally {
    completing = false;
    resetButton.disabled = false;
  }
}

function handleResult(result) {
  if (!result || !scanning || completing) {
    return;
  }
  try {
    const state = assembler.add(result.getText());
    lastScanError = "";
    if (state.waitingForManifest) {
      setScanStatus("正在等待 metadata 二维码；发送端会在循环中重复播放。");
      return;
    }
    if (state.manifestAdded) {
      renderProgress(state);
      setScanStatus(`已读取文件信息，共 ${state.total} 个分片；正在接收。`);
    } else if (state.added) {
      renderProgress(state);
      setScanStatus(`已接收 ${state.received} / ${state.total} 个分片，继续对准桌面二维码。`);
    }
    if (state.complete) {
      void finishTransfer();
    }
  } catch (error) {
    if (error.message !== lastScanError) {
      setScanStatus(`未接收：${error.message}`, "error");
      lastScanError = error.message;
    }
  }
}

async function startCamera(deviceId = selectedCameraId) {
  if (!window.isSecureContext) {
    setScanStatus("摄像头要求 HTTPS 安全连接；请通过 HTTPS 地址打开此网页。", "error");
    return;
  }
  if (!navigator.mediaDevices?.getUserMedia) {
    setScanStatus("当前浏览器无法访问摄像头。", "error");
    return;
  }

  startButton.disabled = true;
  stopButton.disabled = false;
  setScanStatus("正在请求摄像头权限…");
  scanning = true;
  try {
    const reader = new BrowserQRCodeReader(undefined, {
      delayBetweenScanAttempts: 10,
      delayBetweenScanSuccess: 0,
    });
    const onResult = (result) => handleResult(result);
    const startedControls = deviceId
      ? await reader.decodeFromVideoDevice(deviceId, camera, onResult)
      : await reader.decodeFromConstraints(
          { audio: false, video: { facingMode: { ideal: "environment" } } },
          camera,
          onResult,
        );
    if (scanning) {
      controls = startedControls;
      try {
        await refreshCameraOptions();
      } catch (error) {
        setScanStatus(`摄像头已开启，但无法读取摄像头列表：${error.message}`, "error");
        return;
      }
      if (scanning) {
        setScanStatus("摄像头已开启，请将桌面二维码对准取景框。");
      }
    } else {
      startedControls.stop();
    }
  } catch (error) {
    stopCamera();
    setScanStatus(`无法开启摄像头：${error.message}`, "error");
  }
}

function switchCamera() {
  selectedCameraId = cameraSelect.value;
  if (!scanning) {
    return;
  }
  stopCamera();
  void startCamera(selectedCameraId);
}

function resetTransfer() {
  stopCamera();
  assembler = new TransferAssembler();
  completing = false;
  lastScanError = "";
  matrixTransferId = "";
  const matrixContext = chunkMatrix.getContext("2d");
  matrixContext.clearRect(0, 0, chunkMatrix.width, chunkMatrix.height);
  matrixContainer.hidden = true;
  matrixCaption.hidden = true;
  if (downloadURL) {
    URL.revokeObjectURL(downloadURL);
    downloadURL = null;
  }
  downloadLink.removeAttribute("href");
  downloadLink.removeAttribute("download");
  downloadLink.hidden = true;
  renderProgress(assembler.progress());
  setScanStatus("接收已重置。点击“开始扫描”继续。");
}

startButton.addEventListener("click", () => void startCamera());
stopButton.addEventListener("click", stopCamera);
cameraSelect.addEventListener("change", switchCamera);
resetButton.addEventListener("click", resetTransfer);
window.addEventListener("beforeunload", () => {
  stopCamera();
  if (downloadURL) {
    URL.revokeObjectURL(downloadURL);
  }
});

renderProgress(assembler.progress());
