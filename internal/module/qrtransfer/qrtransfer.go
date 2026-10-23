//go:build module_qr_transfer

package qrtransfer

import (
	"context"
	"fmt"
	"io"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/skip2/go-qrcode"

	"moyu-assistant/internal/i18n"
	"moyu-assistant/internal/module"
)

const frameInterval = 2 * time.Second

func init() {
	module.Register(&QRTransferModule{})
}

type QRTransferModule struct {
	state *appState
}

func (m *QRTransferModule) Name() string {
	return i18n.T("二维码文件传输", "QR File Transfer")
}

func (m *QRTransferModule) Description() string {
	return i18n.T("将本地文件拆分为二维码序列", "Split a local file into a QR code sequence")
}

func (m *QRTransferModule) Icon() fyne.Resource {
	return theme.DocumentIcon()
}

func (m *QRTransferModule) Category() string {
	return i18n.T("文件传输", "File Transfer")
}

func (m *QRTransferModule) OnInit() {
	m.state = &appState{}
}

func (m *QRTransferModule) OnDestroy() {
	m.stopPlayback()
}

type appState struct {
	mu             sync.Mutex
	transfer       *Transfer
	current        int
	cancelPlayback context.CancelFunc
	openButton     *widget.Button
	playButton     *widget.Button
	status         *widget.Label
	frameLabel     *widget.Label
	progress       *widget.ProgressBar
	qrImage        *canvas.Image
	placeholder    fyne.CanvasObject
}

func (m *QRTransferModule) CreateUI(window fyne.Window) fyne.CanvasObject {
	if m.state == nil {
		m.state = &appState{}
	}

	header := module.CreateHeader(
		i18n.T("二维码文件传输", "QR File Transfer"),
		i18n.T("在桌面显示二维码序列，并在移动端逐帧扫描以还原文件", "Display a QR sequence for a mobile device to scan and reconstruct"),
	)

	status := widget.NewLabel(i18n.T("请选择要传输的文件", "Select a file to transfer"))
	frameLabel := widget.NewLabel(i18n.T("尚未生成二维码", "No QR codes generated"))
	progress := widget.NewProgressBar()
	progress.Hide()
	qrImage := canvas.NewImageFromResource(theme.DocumentIcon())
	qrImage.FillMode = canvas.ImageFillContain
	qrImage.Hide()
	placeholder := container.NewCenter(widget.NewLabel(i18n.T("二维码将在此处显示", "The QR code will appear here")))

	var openButton *widget.Button
	openButton = widget.NewButton(i18n.T("选择文件", "Select File"), func() {
		m.stopPlayback()
		status.SetText(i18n.T("正在读取文件…", "Reading file…"))
		openButton.Disable()
		dialog.ShowFileOpen(func(reader fyne.URIReadCloser, err error) {
			if err != nil {
				openButton.Enable()
				m.setStatus(i18n.T("选择文件失败", "Failed to select file") + ": " + err.Error())
				dialog.ShowError(err, window)
				return
			}
			if reader == nil {
				openButton.Enable()
				m.setStatus(i18n.T("未选择文件", "No file selected"))
				return
			}
			go m.loadFile(reader)
		}, window)
	})

	previousButton := widget.NewButton(i18n.T("上一帧", "Previous"), func() {
		m.moveFrame(-1)
	})
	playButton := widget.NewButton(i18n.T("自动播放", "Play"), m.togglePlayback)
	nextButton := widget.NewButton(i18n.T("下一帧", "Next"), func() {
		m.moveFrame(1)
	})

	m.state.mu.Lock()
	m.state.openButton = openButton
	m.state.playButton = playButton
	m.state.status = status
	m.state.frameLabel = frameLabel
	m.state.progress = progress
	m.state.qrImage = qrImage
	m.state.placeholder = placeholder
	m.state.mu.Unlock()

	preview := container.NewMax(placeholder, container.NewGridWrap(fyne.NewSize(420, 420), qrImage))
	controls := container.NewHBox(previousButton, playButton, nextButton)
	footer := container.NewVBox(frameLabel, progress, container.NewCenter(controls))

	return container.NewBorder(
		header,
		container.NewVBox(status, footer),
		nil,
		nil,
		container.NewCenter(preview),
	)
}

func (m *QRTransferModule) loadFile(reader fyne.URIReadCloser) {
	fileName := reader.URI().Name()
	data, readErr := io.ReadAll(reader)
	closeErr := reader.Close()
	if readErr != nil {
		m.setStatus(i18n.T("读取文件失败", "Failed to read file") + ": " + readErr.Error())
		m.enableFilePicker()
		return
	}
	if closeErr != nil {
		m.setStatus(i18n.T("关闭文件失败", "Failed to close file") + ": " + closeErr.Error())
		m.enableFilePicker()
		return
	}

	transfer, err := NewTransfer(fileName, data)
	if err != nil {
		m.setStatus(i18n.T("准备传输失败", "Could not prepare transfer") + ": " + err.Error())
		m.enableFilePicker()
		return
	}

	m.state.mu.Lock()
	m.state.transfer = transfer
	m.state.current = 0
	m.state.mu.Unlock()

	m.enableFilePicker()
	m.showFrame(0)
	m.setStatus(fmt.Sprintf(
		i18n.T("%s · %d 字节 · 共 %d 个二维码", "%s · %d bytes · %d QR codes"),
		transfer.Name,
		transfer.Size,
		transfer.Total,
	))
}

func (m *QRTransferModule) showFrame(index int) {
	m.state.mu.Lock()
	transfer := m.state.transfer
	qrImage := m.state.qrImage
	placeholder := m.state.placeholder
	frameLabel := m.state.frameLabel
	progress := m.state.progress
	m.state.mu.Unlock()
	if transfer == nil || qrImage == nil {
		return
	}

	payload, err := transfer.Payload(index)
	if err != nil {
		m.setStatus(i18n.T("生成二维码失败", "Failed to generate QR code") + ": " + err.Error())
		return
	}
	pngData, err := qrcode.Encode(payload, qrcode.Medium, 480)
	if err != nil {
		m.setStatus(i18n.T("生成二维码失败", "Failed to generate QR code") + ": " + err.Error())
		return
	}

	qrImage.Resource = fyne.NewStaticResource("qr-transfer.png", pngData)
	qrImage.Show()
	placeholder.Hide()
	frameLabel.SetText(fmt.Sprintf(
		i18n.T("二维码 %d / %d · 每帧约 2 秒", "QR %d / %d · about 2 seconds per frame"),
		index+1,
		transfer.Total,
	))
	progress.SetValue(float64(index+1) / float64(transfer.Total))
	progress.Show()
	qrImage.Refresh()
}

func (m *QRTransferModule) moveFrame(delta int) {
	m.stopPlayback()
	m.state.mu.Lock()
	if m.state.transfer == nil {
		m.state.mu.Unlock()
		return
	}
	index := m.state.current + delta
	if index < 0 {
		index = 0
	}
	if index >= m.state.transfer.Total {
		index = m.state.transfer.Total - 1
	}
	m.state.current = index
	m.state.mu.Unlock()
	m.showFrame(index)
}

func (m *QRTransferModule) togglePlayback() {
	m.state.mu.Lock()
	if m.state.cancelPlayback != nil {
		cancel := m.state.cancelPlayback
		m.state.cancelPlayback = nil
		playButton := m.state.playButton
		m.state.mu.Unlock()
		cancel()
		playButton.SetText(i18n.T("自动播放", "Play"))
		return
	}
	if m.state.transfer == nil || m.state.transfer.Total < 2 {
		m.state.mu.Unlock()
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	m.state.cancelPlayback = cancel
	playButton := m.state.playButton
	m.state.mu.Unlock()

	playButton.SetText(i18n.T("暂停播放", "Pause"))
	go func() {
		ticker := time.NewTicker(frameInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				m.state.mu.Lock()
				if m.state.transfer == nil || m.state.current >= m.state.transfer.Total-1 {
					m.state.cancelPlayback = nil
					m.state.mu.Unlock()
					playButton.SetText(i18n.T("自动播放", "Play"))
					return
				}
				m.state.current++
				index := m.state.current
				m.state.mu.Unlock()
				m.showFrame(index)
			}
		}
	}()
}

func (m *QRTransferModule) stopPlayback() {
	m.state.mu.Lock()
	cancel := m.state.cancelPlayback
	playButton := m.state.playButton
	m.state.cancelPlayback = nil
	m.state.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	if playButton != nil {
		playButton.SetText(i18n.T("自动播放", "Play"))
	}
}

func (m *QRTransferModule) enableFilePicker() {
	m.state.mu.Lock()
	openButton := m.state.openButton
	m.state.mu.Unlock()
	if openButton != nil {
		openButton.Enable()
	}
}

func (m *QRTransferModule) setStatus(message string) {
	m.state.mu.Lock()
	status := m.state.status
	m.state.mu.Unlock()
	if status != nil {
		status.SetText(message)
	}
}
